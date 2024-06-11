package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/charmbracelet/log"
)

func main() {
	log.Info("Starting server...")

	srv := NewServer()
	srv.Start()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	done := make(chan struct{})
	go func() {
		<-ctx.Done()
		srv.Stop()
		done <- struct{}{}
	}()

	<-done
	log.Info("Bye!")
}
