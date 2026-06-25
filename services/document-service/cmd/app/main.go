package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Temych228/DocflowWeb/services/document-service/internal/app"
	"github.com/Temych228/DocflowWeb/services/document-service/internal/config"
	"github.com/Temych228/DocflowWeb/services/document-service/pkg/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	lg := logger.New()

	application, err := app.New(cfg, lg)
	if err != nil {
		log.Fatalf("failed to initialize app: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := application.Run(ctx); err != nil {
			log.Fatalf("failed to start application: %v", err)
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
