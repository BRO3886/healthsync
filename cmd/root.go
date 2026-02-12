package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/BRO3886/healthsync/internal/storage"
)

var dbPath string

var rootCmd = &cobra.Command{
	Use:   "healthsync",
	Short: "Sync Apple Health export data into a local SQLite database",
	Long: `healthsync parses Apple Health export files (.zip or .xml) and stores
the data in a local SQLite database for easy querying.

Supports heart rate, steps, SpO2, VO2 Max, sleep analysis, and workouts.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", storage.DefaultDBPath(), "path to SQLite database")
}
