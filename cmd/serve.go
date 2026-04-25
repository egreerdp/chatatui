package cmd

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/EwanGreer/chatatui/internal/config"
	"github.com/EwanGreer/chatatui/internal/middleware"
	"github.com/EwanGreer/chatatui/internal/repository"
	"github.com/EwanGreer/chatatui/internal/server"
	"github.com/EwanGreer/chatatui/internal/server/api"
	"github.com/EwanGreer/chatatui/internal/server/hub"
	"github.com/EwanGreer/chatatui/internal/service"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the server",
	Long:  `Start the server`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadServerConfig()
		if err != nil {
			slog.Error("failed to load config", "error", err)
			os.Exit(1)
		}
		if err := cfg.Validate(); err != nil {
			slog.Error("invalid config", "error", err)
			os.Exit(1)
		}

		logLevel := slog.LevelInfo
		debug := os.Getenv("DEBUG")
		if debug == "1" || debug == "true" {
			logLevel = slog.LevelDebug
		}
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel})))

		rateLimiter, err := middleware.NewRateLimiter(cfg.RedisURL, cfg.RateLimitRequests, cfg.RateLimitWindowSecs)
		if err != nil {
			slog.Warn("rate limiter disabled", "error", err)
		}

		database, err := repository.NewPostgresDB(cfg.DatabaseDSN)
		if err != nil {
			slog.Error("failed to initialize database", "error", err)
			os.Exit(1)
		}

		if cfg.RedisURL == "" {
			slog.Error("redis_url is required")
			os.Exit(1)
		}
		opt, err := redis.ParseURL(cfg.RedisURL)
		if err != nil {
			slog.Error("invalid redis_url", "error", err)
			os.Exit(1)
		}
		broker := hub.NewRedisBroker(redis.NewClient(opt))

		hb := hub.NewHub(broker)
		svc := service.NewChatService(database.Rooms(), database.Messages(), cfg.MessageHistoryLimit)
		handler := api.NewHandler(hb, database.Users(), database.Users(), database.Rooms(), svc, cfg, rateLimiter)
		srv := server.NewChatServer(handler, cfg.Addr, hb.Shutdown)

		go func() {
			slog.Info("server started", "addr", cfg.Addr)
			if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				slog.Error("server failed", "error", err)
				os.Exit(1)
			}
		}()

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
		<-quit

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err = srv.Stop(ctx); err != nil {
			slog.Error("graceful shutdown failed", "error", err)
			os.Exit(1)
		}

		slog.Info("server stopped")
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
