package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func handleTermination() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	<-c
	fmt.Println("\nReceived interrupt signal, cleaning up...")
	cleanup()
	os.Exit(0)
}

func cleanup() {
	os.RemoveAll("./temp")
}
