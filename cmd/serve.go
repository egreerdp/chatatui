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

	"github.com/egreerdp/chatatui/internal/config"
	"github.com/egreerdp/chatatui/internal/middleware"
	"github.com/egreerdp/chatatui/internal/repository"
	"github.com/egreerdp/chatatui/internal/server"
	"github.com/egreerdp/chatatui/internal/server/api"
	"github.com/egreerdp/chatatui/internal/server/hub"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the server",
	Long:  `Start the server`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.LoadServerConfig()

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

		database := repository.NewPostgresDB(cfg.DatabaseDSN)
		handler := api.NewHandler(hub.NewHub(), database, cfg, rateLimiter)
		srv := server.NewChatServer(handler, cfg.Port, database)

		go func() {
			slog.Info("server started", "addr", cfg.Port)
			err := srv.Start()
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				panic(err)
			}
		}()

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
		<-quit

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err = srv.Stop(ctx)
		if err != nil {
			panic(err)
		}

		slog.Info("server stopped")
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
