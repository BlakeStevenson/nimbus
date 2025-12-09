package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/blakestevenson/nimbus/internal/auth"
	"github.com/blakestevenson/nimbus/internal/auth/providers"
	"github.com/blakestevenson/nimbus/internal/config"
	"github.com/blakestevenson/nimbus/internal/configstore"
	"github.com/blakestevenson/nimbus/internal/db"
	"github.com/blakestevenson/nimbus/internal/db/generated"
	httpserver "github.com/blakestevenson/nimbus/internal/http"
	"github.com/blakestevenson/nimbus/internal/logging"
	"github.com/blakestevenson/nimbus/internal/media"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func main() {
	// Load .env file if it exists (for development)
	_ = godotenv.Load()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger, err := logging.NewLogger(cfg.IsDevelopment())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting Nimbus server",
		zap.String("environment", cfg.Environment),
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
	)

	// Initialize database
	dbPool, err := db.Connect(context.Background(), cfg.DatabaseURL, logger)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer dbPool.Close()

	logger.Info("Connected to database")

	// Initialize queries
	queries := generated.New(dbPool)

	// Initialize services
	mediaService := media.NewService(queries, logger)
	configStore := configstore.New(queries)

	// Initialize JWT manager
	jwtManager := auth.NewJWTManager(cfg.JWTSecret, 0, 0) // Use default expiry times

	// Initialize auth providers
	passwordProvider := providers.NewPasswordProvider(queries)

	// Initialize auth service
	authService := auth.NewService(queries, jwtManager, passwordProvider, logger)

	// Get library root path from config
	libraryRootPath := "/media" // Default path
	if rootPath, err := configStore.Get(context.Background(), "library.root_path"); err == nil {
		// rootPath is a JSON string, so we need to unmarshal it
		var path string
		if err := json.Unmarshal(rootPath, &path); err == nil {
			libraryRootPath = path
		}
	}

	// Initialize HTTP router
	router := httpserver.NewRouter(mediaService, authService, configStore, queries, libraryRootPath, logger)

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("HTTP server listening", zap.String("address", addr))
		serverErrors <- server.ListenAndServe()
	}()

	// Listen for shutdown signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Block until a signal or error is received
	select {
	case err := <-serverErrors:
		logger.Fatal("Server error", zap.Error(err))

	case sig := <-shutdown:
		logger.Info("Shutdown signal received", zap.String("signal", sig.String()))

		// Give outstanding requests a deadline for completion
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Gracefully shutdown the server
		if err := server.Shutdown(ctx); err != nil {
			logger.Error("Graceful shutdown failed", zap.Error(err))
			if err := server.Close(); err != nil {
				logger.Error("Failed to close server", zap.Error(err))
			}
		}

		logger.Info("Server stopped")
	}
}
