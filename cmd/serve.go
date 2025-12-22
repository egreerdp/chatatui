package cmd

import (
	"log"

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
		h := hub.NewHub()
		handler := api.NewHandler(h)
		srv := server.NewChatServer(handler, ":8080")

		log.Println("Server started")
		err := srv.Start()
		if err != nil {
			panic(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
