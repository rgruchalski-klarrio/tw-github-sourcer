package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/go-github/v51/github"
	"github.com/segmentio/kafka-go"
)

const kafkaTopicName = "github-data"

func main() {

	ctx, cancelFunc := context.WithCancel(context.Background())

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	client := github.NewClient(nil)

	// Asynchronous wait for the interrupt handler.
	// When processed, the context will be cancelled and it will signal
	// that we are done waiting for next event batch.
	go func() {
		<-c
		cancelFunc()
	}()

	// setup Kafka producer:
	dialer := &kafka.Dialer{
		Timeout:   10 * time.Second,
		DualStack: true,
	}

	producer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:      []string{"127.0.0.1:9092"},
		Topic:        kafkaTopicName,
		Dialer:       dialer,
		RequiredAcks: -1, // require confirmation from all replicas
		BatchSize:    1,
		MaxAttempts:  3,
	})

loop:
	for {

		events, response, err := client.Activity.ListEvents(context.TODO(), &github.ListOptions{
			PerPage: 100,
		})

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

		// Handle error, if any
		if err != nil {
			if _, ok := err.(*github.RateLimitError); ok {
				fmt.Println("ERROR: rate limit")
				select {
				case <-time.After(time.Second * time.Duration(queryIntervalSecs)):
					continue loop
				case <-ctx.Done():
					break loop
				}
			} else if _, ok := err.(*github.AbuseRateLimitError); ok {
				fmt.Println("ERROR: secondary rate limit")
				select {
				case <-time.After(time.Second * time.Duration(queryIntervalSecs)):
					continue loop
				case <-ctx.Done():
					break loop
				}
			} else {
				fmt.Println("ERROR: github API error", err.Error())
				select {
				case <-time.After(time.Second * time.Duration(queryIntervalSecs)):
					continue loop
				case <-ctx.Done():
					break loop
				}
			}
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
			fmt.Println("Kafka statistics", producer.Stats())
		}

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
