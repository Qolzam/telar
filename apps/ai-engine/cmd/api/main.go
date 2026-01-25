// AI Engine - Community Knowledge Engine for Telar Platform
package main

import (
	"context"
	"log"

	"github.com/joho/godotenv"
	"github.com/qolzam/telar/apps/ai-engine/internal/config"
)

const (
	serviceName    = "ai-engine"
	serviceVersion = "v1.0.0"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: No .env file found or failed to load: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Println("✅ Configuration loaded and validated successfully.")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	services, err := initializeServices(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize services: %v", err)
	}

	performHealthChecks(ctx, services)

	app := startServer(services, cfg)
	waitForShutdown(app, cancel)
}
