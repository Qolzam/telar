package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func Router(handler *Handler) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName: "AI Engine v1.0.0",
	})
	
	app.Use(logger.New())
	app.Use(recover.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Content-Type,Authorization",
	}))
	
	app.Get("/health", handler.Health)
	
	v1 := app.Group("/api/v1")
	v1.Post("/ingest", handler.Ingest)
	v1.Post("/query", handler.Query)
	
	return app
}
