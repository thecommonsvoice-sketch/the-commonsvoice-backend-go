package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/config"
	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/database"
	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/middleware"
	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/models"
	"github.com/thecommonsvoice-sketch/the-commonsvoice-backend-go/internal/router"
)

func main() {

	// create logger

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// load config
	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// load database

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}

	// auto migration

	if cfg.RunMigrations {
		err = db.AutoMigrate(
			&models.User{},
			&models.RefreshToken{},
			&models.Category{},
			&models.Article{},
			&models.ArticleVideo{},
			&models.Comment{},
			&models.Bookmark{},
			&models.LatestNews{},
		)

		if err != nil {
			logger.Error("migration failed", "error", err)
			os.Exit(1)
		}

		logger.Info("migrations complete")
	}

	psql, _ := db.DB()
	defer psql.Close()

	// create mux router

	mux := http.NewServeMux()

	// add router
	router.Register(mux, db, cfg)

	addr := ":" + strconv.Itoa(cfg.Port)

	logger.Info("server is starting", "addr", addr)

	handler := middleware.Recovery(middleware.CORS(cfg.FrontendURL)(middleware.RequestID(middleware.Logger(mux))))

	server := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	<-shutdown

	logger.Info("shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", "error", err)
		os.Exit(1)
	}

	logger.Info("server stopped")
}
