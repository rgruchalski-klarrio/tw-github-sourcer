package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/go-github/v51/github"
	"github.com/segmentio/kafka-go"
)

var (
	kafkaBroker                = "127.0.0.1:9092"
	kafkaProducerTopicName     = "github-data"
	maximumGithubEventsPerPage = 100
	githubEventsPerPage        = maximumGithubEventsPerPage

	startupInfoReported = false
)

func handleExit() context.Context {
	ctx, cancelFunc := context.WithCancel(context.Background())
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	// Asynchronous wait for the interrupt handler.
	// When processed, the context will be cancelled and it will signal
	// that we are done waiting for next event batch.
	go func() {
		<-c
		cancelFunc()
	}()
	return ctx
}

func initFlags() {
	flag.StringVar(&kafkaBroker,
		"broker",
		kafkaBroker,
		"Kafka broker address")
	flag.StringVar(&kafkaProducerTopicName,
		"producer-topic",
		kafkaProducerTopicName,
		"Topic name to produce messages to")
	flag.IntVar(&githubEventsPerPage,
		"events-per-page",
		githubEventsPerPage,
		"Number of events per page. Maximum 100.")
	flag.Parse()

	// Validate input:
	if githubEventsPerPage > maximumGithubEventsPerPage {
		githubEventsPerPage = maximumGithubEventsPerPage
	}
}

func main() {

	initFlags()
	ctx := handleExit()

	// Construct a GitHub client:
	client := github.NewClient(nil)

	// Construct a Kafka producer:
	producer := kafka.NewWriter(kafka.WriterConfig{
		Brokers: []string{kafkaBroker},
		Topic:   kafkaProducerTopicName,
		Dialer: &kafka.Dialer{
			Timeout:   10 * time.Second,
			DualStack: true,
		},
		BatchSize:    githubEventsPerPage,
		RequiredAcks: -1, // require confirmation from all replicas
		MaxAttempts:  3,
	})

loop:
	for {

		events, response, err := client.Activity.ListEvents(ctx, &github.ListOptions{
			PerPage: githubEventsPerPage,
		})

		// Handle error, if any
		if err != nil {
			if _, ok := err.(*github.RateLimitError); ok {
				fmt.Println("ERROR: rate limit")
				select {
				case <-time.After(time.Second):
					continue loop
				case <-ctx.Done():
					break loop
				}
			} else if _, ok := err.(*github.AbuseRateLimitError); ok {
				fmt.Println("ERROR: secondary rate limit")
				select {
				case <-time.After(time.Second):
					continue loop
				case <-ctx.Done():
					break loop
				}
			} else {
				fmt.Println("ERROR: github API error", err.Error())
				select {
				case <-time.After(time.Second):
					continue loop
				case <-ctx.Done():
					break loop
				}
			}
		}

		// GitHub API is rate limited. The good thing is, the response tells us
		// what the limit is. We can calculate how often we should query the API:
		queryIntervalSecs := 3600 / response.Rate.Limit
		if response.Rate.Remaining == 0 {
			rateLimitResumeIntervalSecs := response.Rate.Reset.Sub(time.Now().UTC()).Seconds()
			fmt.Println("Program is rate limited until", response.Rate.Reset.String(), "and will resume in", rateLimitResumeIntervalSecs, "seconds")
			if rateLimitResumeIntervalSecs > 60 {
				rateLimitResumeIntervalSecs = 60
			}
			select {
			case <-time.After(time.Second * time.Duration(rateLimitResumeIntervalSecs)):
				continue loop
			case <-ctx.Done():
				break loop
			}
		}

		if !startupInfoReported {
			fmt.Println("Processing a maximum of", githubEventsPerPage, "GitHub events every", queryIntervalSecs, "seconds.\nEvents will be produced to the Kafka topic", kafkaProducerTopicName, "at", kafkaBroker, "\n...")
			startupInfoReported = true
		}

		kafkaMessages := []kafka.Message{}

		for _ /* index */, ev := range events {
			// Kafka transports byte arrays, we have structured data here.
			// 1. Serialize the data to a JSON string:
			rawData, err := json.Marshal(ev)
			if err != nil {
				fmt.Println("ERROR: failed serializing event data to JSON string", err.Error())
				continue
			}
			kafkaMessages = append(kafkaMessages, kafka.Message{
				Value: rawData,
			})
		}

		if len(kafkaMessages) > 0 {
			writerErr := producer.WriteMessages(ctx, kafkaMessages...)
			if writerErr != nil {
				fmt.Println("ERROR: Kafka producer failed write", writerErr.Error())
			}
			producerStats := producer.Stats()
			rawStats, marshalErr := json.Marshal(&producerStats)
			if marshalErr == nil {
				fmt.Println("Kafka statistics", string(rawStats))
			} else {
				fmt.Println("ERROR: Kafka stats could not be serialized", marshalErr.Error())
			}
		}

		fmt.Println("Next iteration in", queryIntervalSecs, "seconds...")

		select {
		case <-time.After(time.Second * time.Duration(queryIntervalSecs)):
			// continue, nothing to do here
		case <-ctx.Done():
			fmt.Println("Program finished, exit")
			break loop
		}
	}

	fmt.Println("All done.")
}
