// @title			Validafianza API
// @version			1.0
// @description		API para validación de pólizas de fianza de afianzadoras mexicanas.
// @contact.name	Validafianza
// @license.name	Private
// @host			localhost:3000
// @BasePath		/
package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	fiberSwagger "github.com/gofiber/swagger"

	_ "validafianza-server/docs"
	"validafianza-server/internal/config"
	"validafianza-server/internal/handlers"
	applogger "validafianza-server/internal/logger"
	"validafianza-server/internal/middleware"
	"validafianza-server/internal/repository"
	"validafianza-server/internal/scrapers"
	"validafianza-server/internal/scrapers/polizas"
	"validafianza-server/internal/services"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	applogger.Init(cfg.Env)

	repo, err := repository.NewPostgresRepo(cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to db", "error", err)
		os.Exit(1)
	}
	defer repo.Close()

	scrapers.Register(polizas.NewMapfreScraper())
	scrapers.Register(polizas.NewBerkleyScraper())
	scrapers.Register(polizas.NewSofimexScraper())
	scrapers.Register(polizas.NewDoramaScraper())
	scrapers.Register(polizas.NewTokioScraper())
	scrapers.Register(polizas.NewAtlasScraper())
	scrapers.Register(polizas.NewAsertaScraper())
	scrapers.Register(polizas.NewInsurgentesScraper())

	cacheRepo := repository.NewCacheRepo(repo.Pool())
	rateLimitRepo := repository.NewRateLimitRepo(repo.Pool())
	validationService := services.NewValidationServices(repo, cacheRepo, cfg.CacheTTL)

	app := fiber.New(fiber.Config{
		AppName: "Validafianza API v1.0",
	})

	app.Use(recover.New())
	app.Use(middleware.RequestLogger())
	app.Use(cors.New())

	app.Get("/swagger/*", fiberSwagger.New(fiberSwagger.Config{
		URL:         "doc.json",
		DeepLinking: true,
	}))

	healthHandler := handlers.NewHealthHandler(repo)
	polizaHandler := handlers.NewPolizaHandler(validationService)

	app.Get("/health", healthHandler.Check)

	api := app.Group("/api/v1")

	polizasGroup := api.Group("/polizas")
	polizasGroup.Use(middleware.RateLimit(rateLimitRepo, cfg.RateLimitMax, cfg.RateLimitWindowSec))
	polizasGroup.Post("/validate", polizaHandler.ValidatePoliza)
	polizasGroup.Get("/providers", polizaHandler.ListProviders)

	go func() {
		addr := fmt.Sprintf(":%s", cfg.Port)
		slog.Info("server starting",
			"addr", addr,
			"env", cfg.Env,
			"swagger", fmt.Sprintf("http://localhost%s/swagger/index.html", addr),
		)
		if err := app.Listen(addr); err != nil {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server")
	if err := app.Shutdown(); err != nil {
		slog.Error("forced shutdown", "error", err)
	}
	slog.Info("server exited")
}
