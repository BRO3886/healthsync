package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/sidv/health-sync/internal/server"
	"github.com/sidv/health-sync/internal/storage"
)

var (
	serverPort int
	serverHost string
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the HTTP server for receiving health data uploads",
	RunE:  runServer,
}

func init() {
	serverCmd.Flags().IntVar(&serverPort, "port", 8080, "port to listen on")
	serverCmd.Flags().StringVar(&serverHost, "host", "0.0.0.0", "host to bind to")
	rootCmd.AddCommand(serverCmd)
}

func runServer(cmd *cobra.Command, args []string) error {
	db, err := storage.Open(dbPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	fmt.Printf("Database: %s\n", dbPath)

	return server.Start(server.Config{
		Host: serverHost,
		Port: serverPort,
		DB:   db,
	})
}
