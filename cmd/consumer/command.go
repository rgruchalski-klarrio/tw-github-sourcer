package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Klarrio/tw-github-sourcer/defaults"
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

	flag.StringVar(&defaults.ServerBindHostPort,
		"server-bind-host-port",
		defaults.ServerBindHostPort,
		"Host port to bind the HTTP server on")

	flag.Parse()

}

func getRollups(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		fmt.Fprint(w, "OK")
		return
	}
	w.WriteHeader(http.StatusMethodNotAllowed)
}

func main() {

	initFlags()
	ctx := handleExit()

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
