package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Temych228/DocflowWeb/services/notification-service/internal/app"
	"github.com/Temych228/DocflowWeb/services/notification-service/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	application, err := app.New(cfg)
	if err != nil {
		log.Fatalf("failed to initialize app: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := application.Run(ctx); err != nil {
			log.Fatalf("failed to start app: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	cancel()
	if err := application.Shutdown(context.Background()); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}
