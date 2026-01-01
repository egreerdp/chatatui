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
	"github.com/egreerdp/chatatui/internal/repository"
	"github.com/egreerdp/chatatui/internal/server"
	"github.com/egreerdp/chatatui/internal/server/api"
	"github.com/egreerdp/chatatui/internal/server/hub"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the serve",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.LoadServerConfig()

		database := repository.NewPostgresDB(cfg.DatabaseDSN)
		handler := api.NewHandler(hub.NewHub(), database, cfg)
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

		err := srv.Stop(ctx)
		if err != nil {
			panic(err)
		}

		log.Println("stopped")
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
