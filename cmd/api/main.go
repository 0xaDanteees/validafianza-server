// @title			Validafianza API
// @version			1.0
// @description		API para validación de pólizas de fianza de afianzadoras mexicanas.
// @contact.name	Validafianza
// @license.name	Private
// @host			localhost:3000
// @BasePath		/
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
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
	"validafianza-server/internal/services/providers"
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

	authRepo := repository.NewAuthRepository(repo.Pool())
	emailProvider := providers.NewResendProvider()
	smsProvider := providers.NewFreeSMSProvider()
	notificationSvc := services.NewNotificationService(emailProvider, smsProvider)
	authService := services.NewAuthService(authRepo, notificationSvc)

	sancionRepo := repository.NewSancionRepository(repo.Pool())
	sancionService := services.NewSancionService(sancionRepo, cfg.CacheTTL)

	app := fiber.New(fiber.Config{
		AppName: "Validafianza API v1.0",
	})

	app.Use(recover.New())
	app.Use(middleware.RequestLogger())

	corsConfig := cors.Config{
		AllowOrigins: cfg.AllowedOrigins,
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
		MaxAge:       86400,
	}

	if cfg.AllowedOrigins != "*" {
		corsConfig.AllowCredentials = true
	}

	app.Use(cors.New(corsConfig))

	app.Get("/swagger/*", fiberSwagger.New(fiberSwagger.Config{
		URL:         "doc.json",
		DeepLinking: true,
	}))

	healthHandler := handlers.NewHealthHandler(repo)
	polizaHandler := handlers.NewPolizaHandler(validationService)
	authHandler := handlers.NewAuthHandler(authService)
	sancionHandler := handlers.NewSancionHandler(sancionService)
	authMiddleware := middleware.NewAuthMiddleware(authService)
	rateLimitMw := middleware.NewRateLimitMiddleware(authRepo)

	app.Get("/health", healthHandler.Check)

	api := app.Group("/api")

	authGroup := api.Group("/auth")
	authGroup.Post("/register-or-login", rateLimitMw.LimitByIP(10, 60), authHandler.RegisterOrLogin)
	authGroup.Post("/verify", rateLimitMw.LimitByIP(5, 60), authHandler.VerifyPasscode)
	authGroup.Get("/profile", authMiddleware.Authenticate, authHandler.GetProfile)
	authGroup.Post("/logout", authMiddleware.Authenticate, authHandler.Logout)

	v1 := api.Group("/v1")
	polizasGroup := v1.Group("/polizas")
	polizasGroup.Use(authMiddleware.Authenticate)
	polizasGroup.Use(middleware.RateLimit(rateLimitRepo, cfg.RateLimitMax, cfg.RateLimitWindowSec))
	polizasGroup.Post("/validate", polizaHandler.ValidatePoliza)
	polizasGroup.Get("/providers", polizaHandler.ListProviders)

	sancionesGroup := v1.Group("/sanciones")
	sancionesGroup.Use(authMiddleware.Authenticate)
	sancionesGroup.Use(middleware.RateLimit(rateLimitRepo, cfg.RateLimitMax, cfg.RateLimitWindowSec))
	sancionesGroup.Post("/scg", sancionHandler.ValidarRFCSCG)
	sancionesGroup.Post("/sfp", sancionHandler.ValidarRFCSFP)

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
