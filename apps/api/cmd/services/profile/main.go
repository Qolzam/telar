package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/qolzam/telar/apps/api/internal/database/postgres"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/profile"
	profileRepository "github.com/qolzam/telar/apps/api/profile/repository"
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

	// Create postgres client for repositories
	ctx := context.Background()
	pgConfig := &dbi.PostgreSQLConfig{
		Host:               cfg.Database.Postgres.Host,
		Port:               cfg.Database.Postgres.Port,
		Username:           cfg.Database.Postgres.Username,
		Password:           cfg.Database.Postgres.Password,
		Database:           cfg.Database.Postgres.Database,
		SSLMode:            cfg.Database.Postgres.SSLMode,
		MaxOpenConnections: cfg.Database.Postgres.MaxOpenConns,
		MaxIdleConnections: cfg.Database.Postgres.MaxIdleConns,
		MaxLifetime:        int(cfg.Database.Postgres.ConnMaxLifetime.Seconds()),
		ConnectTimeout:     10,
	}
	pgClient, err := postgres.NewClient(ctx, pgConfig, pgConfig.Database)
	if err != nil {
		log.Fatalf("Failed to create postgres client: %v", err)
	}

	// Create repository
	profileRepo := profileRepository.NewPostgresProfileRepository(pgClient)

	// Create profile service with repository
	profileService := services.NewProfileService(profileRepo, cfg)

	// Create profile service client adapter for gRPC
	profileServiceClient := profile.NewDirectCallAdapter(profileService)

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
			pb.RegisterProfileServiceServer(grpcServer, profile.NewGrpcServer(profileServiceClient))

			log.Printf("🚀 Profile gRPC Server listening on port %s", grpcPort)
			if err := grpcServer.Serve(lis); err != nil {
				log.Fatalf("Failed to serve gRPC: %v", err)
			}
		}()
	}

	log.Printf("Starting Profile Service on port 8081")
	log.Fatal(app.Listen(":8081"))
}



