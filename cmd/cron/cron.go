package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {

	done := make(chan bool, 1)
	signalsChannel := make(chan os.Signal, 1)
	signal.Notify(signalsChannel, syscall.SIGINT, syscall.SIGTERM)
	log.Println("start waiting for signal")
	_, cancel := context.WithCancel(context.Background())

	go func() {
		log.Println("....")
	}()

	go func() {
		log.Println("Waiting for signal")
		sig := <-signalsChannel
		log.Println("Got signal:", sig)
		log.Println("Terminating...")
		cancel()
		done <- true
	}()
}
