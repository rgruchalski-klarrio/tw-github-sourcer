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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/segmentio/kafka-go"
)

var (
	kafkaBroker                = "127.0.0.1:9092"
	kafkaProducerTopicName     = "rollups"
	maximumGithubEventsPerPage = 100
	githubEventsPerPage        = maximumGithubEventsPerPage

	startupInfoReported = false

	slidingWindowDuration = time.Minute * time.Duration(5)
	programStartedAt      = time.Now()
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
		BatchSize:    1,  // back to one because we have one rollups object to publish per iteration
		RequiredAcks: -1, // require confirmation from all replicas
		MaxAttempts:  3,
	})

	registry := prometheus.NewRegistry()
	summaries := map[string]prometheus.Summary{}

loop:
	for {

		// Query GitHub for the list of events:
		events, response, err := client.Activity.ListEvents(ctx, &github.ListOptions{
			PerPage: githubEventsPerPage,
		})

		// Handle error, if any:
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
			fmt.Println("Program is rate limited until",
				response.Rate.Reset.String(),
				"and will resume in",
				rateLimitResumeIntervalSecs,
				"seconds")

			if rateLimitResumeIntervalSecs > 60 {
				// QUESTION: why do we do this?
				rateLimitResumeIntervalSecs = 60
			}

			select {
			case <-time.After(time.Second * time.Duration(rateLimitResumeIntervalSecs)):
				// Continue the loop here because we have exhausted rate limits.
				// This happens after the duration of when the rate limit should have been reset.
				continue loop
			case <-ctx.Done():
				break loop
			}

		}

		if !startupInfoReported {
			// Do this once:
			startupInfoReported = true
			fmt.Println("Processing a maximum of", githubEventsPerPage, "GitHub events every", queryIntervalSecs,
				"seconds.\nEvents will be produced to the Kafka topic", kafkaProducerTopicName, "at", kafkaBroker, "\n...")
		}

		for _, ev := range events {

			// Data is processed synchronously.
			// 1. Ensure that we have a metric for the event type:

			if _, ok := summaries[*ev.Type]; !ok {

				summary := prometheus.NewSummary(prometheus.SummaryOpts{
					Namespace: "tw",
					Subsystem: "github",
					Name:      *ev.Type,
					MaxAge:    slidingWindowDuration,
				})
				if err := registry.Register(summary); err != nil {
					fmt.Println("ERROR: Failed registering a summary for event type", *ev.Type, "reason", err.Error())
				} else {
					summaries[*ev.Type] = summary
					// Observe the metric immediately:
					summaries[*ev.Type].Observe(float64(1))
				}

			} else {
				// Observe the metric for the type event type we definitely have in the map:
				summaries[*ev.Type].Observe(float64(1))
			}

		}

		// Once we have iterated over individual events, collect metrics and put them in the rollups object:
		kafkaPayload := newRollups(programStartedAt, slidingWindowDuration)
		// Gather all metrics and put them in the Kafka payload:
		metrics, gatherErr := registry.Gather()
		if gatherErr != nil {
			fmt.Println("ERROR: Failed gathering metrics", gatherErr.Error())
		} else {
			for _, metricFamily := range metrics {
				metric := metricFamily.GetMetric()
				for _, metricValue := range metric {
					// We know that we have put summaries into our metric registry.
					// This operation is safe.
					// QUESTION: The underlying map is not safe for async usage. When would this operation not be safe?
					kafkaPayload.PutData(*metricFamily.Name, *metricValue.Summary.SampleSum)
				}
			}
		}

		// Serialize the Kafka payload to bytes:
		kafkaPayloadBytes, jsonErr := json.Marshal(kafkaPayload)
		if jsonErr != nil {
			fmt.Println("ERROR: Failed serializing rollups", jsonErr.Error())
			// don't continue the loop because we want to obey GitHub rate limits
		} else {

			writerErr := producer.WriteMessages(ctx, kafka.Message{
				Value: kafkaPayloadBytes,
			})
			if writerErr != nil {
				fmt.Println("ERROR: Kafka producer write failed", writerErr.Error())
			} else {
				// QUESTION: there's a problem left to resolve.
				// Because we are capturing the name from the metric, each event type
				// in the resulting map is prefixed with summary.Namespace and summary.Subsystem.
				// What can we do about it?
				fmt.Println("Written rollups to Kafka", string(kafkaPayloadBytes))
			}

		}

		fmt.Println("Next iteration at", time.Now().Add(time.Second*time.Duration(queryIntervalSecs)), "...")

		select {
		case <-time.After(time.Second * time.Duration(queryIntervalSecs)):
			// continue, nothing to do here
		case <-ctx.Done():
			fmt.Println("Program finished, exit")
			break loop
		}
	}

	fmt.Println("All done. Program will exit.")
}
