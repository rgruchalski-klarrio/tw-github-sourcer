package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Klarrio/tw-github-sourcer/defaults"
	"github.com/segmentio/kafka-go"
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

var (
	latestPayload     = ""
	latestPayloadLock = &sync.Mutex{}
)

func initFlags() {

	flag.StringVar(&defaults.KafkaBroker,
		"broker",
		defaults.KafkaBroker,
		"Kafka broker address")

	flag.StringVar(&defaults.KafkaRollupsTopicName,
		"consumer-topic",
		defaults.KafkaRollupsTopicName,
		"Topic name to consume messages from")

	flag.StringVar(&defaults.KafkaConsumerGroupID,
		"consumer-group",
		defaults.KafkaConsumerGroupID,
		"Kafka consumer group")

	flag.StringVar(&defaults.ServerBindHostPort,
		"server-bind-host-port",
		defaults.ServerBindHostPort,
		"Host port to bind the HTTP server on")

	flag.Parse()

}

func getRollups(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		latestPayloadLock.Lock()
		defer latestPayloadLock.Unlock()
		if latestPayload == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Add("content-type", "application/json")
		fmt.Fprint(w, latestPayload)
		return
	}
	w.WriteHeader(http.StatusMethodNotAllowed)
}

func main() {

	initFlags()
	ctx := handleExit()

	fmt.Println("Starting a consumer with group", defaults.KafkaConsumerGroupID)

	// Construct a Kafka consumer:
	consumer := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{defaults.KafkaBroker},
		Topic:   defaults.KafkaRollupsTopicName,
		GroupID: defaults.KafkaConsumerGroupID,

		Dialer: &kafka.Dialer{
			Timeout:   10 * time.Second,
			DualStack: true,
		},
	})

	go func() {
		consumer.SetOffset(0)
	loop:
		for {
			m, err := consumer.ReadMessage(ctx)
			if err != nil {
				select {
				case <-ctx.Done():
					break loop
				default:
					fmt.Println("ERROR: Kafka consume failed", err.Error())
					continue loop
				}
			}
			// For brevity we are going to ignore the fact that this
			fmt.Println("message at offset", m.Offset, "=", string(m.Value))

			// Update the latest stored value:
			latestPayloadLock.Lock()
			latestPayload = string(m.Value)
			latestPayloadLock.Unlock()

			select {
			case <-ctx.Done():
				break loop
			default:
				// continue
			}
		}
		consumer.Close()
	}()

	fmt.Println("Starting the HTTP server. Binding on", defaults.ServerBindHostPort)

	http.HandleFunc("/rollups", getRollups)

	go func() {
		// http.ListenAndServe is blocking. This code must run asynchronously.
		if err := http.ListenAndServe(defaults.ServerBindHostPort, nil); err != nil {
			fmt.Println("ERROR: HTTP server failed", err.Error())
		}
	}()

	// Wait for the server to stop:
	<-ctx.Done()

}
