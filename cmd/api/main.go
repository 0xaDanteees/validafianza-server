package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
    "github.com/gofiber/fiber/v2/middleware/cors"
    "github.com/gofiber/fiber/v2/middleware/logger"
    "github.com/gofiber/fiber/v2/middleware/recover"
    
    "validafianza-server/internal/config"
    "validafianza-server/internal/handlers"
    "validafianza-server/internal/repository"
    "validafianza-server/internal/services"
)


func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	repo, err := repository.NewPostgresRepo(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to db: %w", err)
	}

	defer repo.Close()

	cacheRepo := repository.NewCacheRepo(repo.Pool())
	validationService := services.NewValidationServices(repo, cacheRepo, cfg.CacheTTL)

	app := fiber.New(fiber.config{
		AppName: "Validafianza API v1.0",
	})

	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New())

	healthHandler := handlers.NewHealthHandler(repo)
	polizaHandler := handlers.NewPolizaHandler(validationService)

	app.Get("/health", HealthHandler.Check)

	api := app.Group("/api/v1")

	polizas := api.Group("/polizas")
	polizas.Post("/validate", polizaHandler.ValidatePoliza)
	poliza.Get("/providers", polizaHandler.ListProviders)

	go func() {
		addr := fmt.Sprintf(":%s", cfg.Port)
		log.Printf("Ser er starting on %s", addr)
		if err := app.Listen(addr); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	if err := app.Shutdown(); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("server exited")
}