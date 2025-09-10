// AI Engine - Community Knowledge Engine for Telar Platform
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/qolzam/telar/apps/ai-engine/internal/api"
	"github.com/qolzam/telar/apps/ai-engine/internal/config"
)

const (
	serviceName    = "ai-engine"
	serviceVersion = "v1.0.0"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	handler := api.NewHandler()
	app := api.Router(handler)

	go func() {
		addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
		log.Printf("Starting %s %s on %s", serviceName, serviceVersion, addr)
		log.Printf("LLM Provider: %s", cfg.LLM.Provider)
		log.Printf("Weaviate URL: %s", cfg.Weaviate.URL)
		
		if err := app.Listen(addr); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}
