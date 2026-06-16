package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"

	"github.com/Temych228/DocflowWeb/services/auth-service/internal/app"
	"github.com/Temych228/DocflowWeb/services/auth-service/internal/config"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("[INFO] .env file not found, using environment variables")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("[FATAL] load config: %v", err)
	}

	a, err := app.New(cfg)
	if err != nil {
		log.Fatalf("[FATAL] init app: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := a.Run(ctx); err != nil {
		log.Fatalf("[FATAL] run app: %v", err)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[INFO] shutdown signal received")
	if err := a.Shutdown(ctx); err != nil {
		log.Printf("[ERROR] shutdown: %v", err)
	}
}
