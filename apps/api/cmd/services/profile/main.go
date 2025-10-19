package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/qolzam/telar/apps/api/internal/platform"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/profile"
	"github.com/qolzam/telar/apps/api/profile/services"
	pb "github.com/qolzam/telar/protos/gen/go/profilepb"
	"google.golang.org/grpc"
)

func main() {
	cfg, err := platformconfig.LoadFromEnv()
	if err != nil {
		log.Fatalf("Failed to load platform config: %v", err)
	}

	app := fiber.New()

	baseService, err := platform.NewBaseService(context.Background(), cfg)
	if err != nil {
		log.Fatalf("Failed to create base service: %v", err)
	}

	profileService := services.NewService(baseService, cfg)

	// Create database indexes on startup
	log.Println("🔧 Creating database indexes for Profile service...")
	indexCtx, indexCancel := context.WithTimeout(context.Background(), 30*time.Second)
	if err := profileService.CreateIndexes(indexCtx); err != nil {
		indexCancel()
		log.Printf("⚠️  Warning: Failed to create indexes (may already exist): %v", err)
	} else {
		indexCancel()
		log.Println("✅ Profile database indexes created successfully")
	}

	profileHandler := profile.NewProfileHandler(profileService, cfg.JWT, cfg.HMAC)

	profileHandlers := &profile.ProfileHandlers{
		ProfileHandler: profileHandler,
	}

	profile.RegisterRoutes(app, profileHandlers, cfg)

	// Start gRPC server if in microservices mode
	if os.Getenv("START_GRPC_SERVER") == "true" {
		grpcPort := os.Getenv("GRPC_PORT")
		if grpcPort == "" {
			grpcPort = "50051"
		}

		go func() {
			lis, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
			if err != nil {
				log.Fatalf("Failed to listen on gRPC port %s: %v", grpcPort, err)
			}

			grpcServer := grpc.NewServer()
			pb.RegisterProfileServiceServer(grpcServer, profile.NewGrpcServer(profileService))

			log.Printf("🚀 Profile gRPC Server listening on port %s", grpcPort)
			if err := grpcServer.Serve(lis); err != nil {
				log.Fatalf("Failed to serve gRPC: %v", err)
			}
		}()
	}

	log.Printf("Starting Profile Service on port 8081")
	log.Fatal(app.Listen(":8081"))
}



