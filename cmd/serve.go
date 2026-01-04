package cmd

import (
	"context"
	"errors"
	"log"
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

		rateLimiter, err := middleware.NewRateLimiter(cfg.RedisURL, cfg.RateLimitRequests, cfg.RateLimitWindowSecs)
		if err != nil {
			log.Printf("warning: rate limiter disabled: %v\n", err)
		}

		database := repository.NewPostgresDB(cfg.DatabaseDSN)
		handler := api.NewHandler(hub.NewHub(), database, cfg, rateLimiter)
		srv := server.NewChatServer(handler, cfg.Port, database)

		go func() {
			log.Printf("server running on %s\n", cfg.Port)
			err := srv.Start()
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				panic(err)
			}
		}()

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err = srv.Stop(ctx)
		if err != nil {
			panic(err)
		}

		log.Println("stopped")
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
