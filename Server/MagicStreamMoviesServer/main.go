package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/reuien/MagicStreamMovies/Server/MagicStreamMoviesServer/config"
	"github.com/reuien/MagicStreamMovies/Server/MagicStreamMoviesServer/database"
	"github.com/reuien/MagicStreamMovies/Server/MagicStreamMoviesServer/middleware"
	"github.com/reuien/MagicStreamMovies/Server/MagicStreamMoviesServer/routes"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration_invalid", "error", err)
		os.Exit(1)
	}
	logger.Info("configuration_loaded", "environment", cfg.Environment, "port", cfg.Port, "database", cfg.Database, "ai_model", cfg.AI.Model, "ai_base_url_configured", cfg.AI.BaseURL != "")

	client, err := database.Connect(cfg.MongoURI, cfg.Database)
	if err != nil {
		logger.Error("database_initialization_failed", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := client.Disconnect(context.Background()); err != nil {
			logger.Error("database_disconnect_failed", "error", err)
		}
	}()
	indexContext, cancelIndexes := context.WithTimeout(context.Background(), 10*time.Second)
	if err := database.EnsureIndexes(indexContext); err != nil {
		cancelIndexes()
		logger.Error("database_index_initialization_failed", "error", err)
		os.Exit(1)
	}
	cancelIndexes()

	router := gin.New()
	router.Use(middleware.RequestLogger(logger), gin.Recovery())
	routes.SetupUnprotectedRoutes(router, client)
	routes.SetupProtectedRoutes(router, client, cfg.AI)
	server := &http.Server{Addr: ":" + cfg.Port, Handler: router, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		logger.Info("server_started", "address", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server_failed", "error", err)
		}
	}()

	shutdownSignal, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-shutdownSignal.Done()
	shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	if err := server.Shutdown(shutdownContext); err != nil {
		logger.Error("server_shutdown_failed", "error", err)
	} else {
		logger.Info("server_stopped")
	}
}
