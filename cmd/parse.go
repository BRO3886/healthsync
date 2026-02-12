package cmd

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/sidv/health-sync/internal/parser"
	"github.com/sidv/health-sync/internal/storage"
)

var verbose bool

var parseCmd = &cobra.Command{
	Use:   "parse <file.zip|export.xml>",
	Short: "Parse an Apple Health export file into the database",
	Args:  cobra.ExactArgs(1),
	RunE:  runParse,
}

func init() {
	parseCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose logging")
	rootCmd.AddCommand(parseCmd)
}

func runParse(cmd *cobra.Command, args []string) error {
	path := args[0]

	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", path)
	}
	if err != nil {
		return fmt.Errorf("stat file: %w", err)
	}

	if verbose {
		log.Printf("[verbose] input file: %s (%.2f MB)", path, float64(info.Size())/(1024*1024))
		log.Printf("[verbose] database: %s", dbPath)
	}

	db, err := storage.Open(dbPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	fmt.Printf("Parsing %s...\n", path)
	fmt.Printf("Database: %s\n\n", dbPath)

	start := time.Now()
	lastLog := time.Now()

	progress := func(records int64, workouts int64) {
		fmt.Fprintf(os.Stderr, "\r  Records: %d | Workouts: %d", records, workouts)
		if verbose && time.Since(lastLog) > 5*time.Second {
			elapsed := time.Since(start).Round(time.Millisecond)
			rate := float64(records+workouts) / time.Since(start).Seconds()
			log.Printf("[verbose] progress: %d records, %d workouts (%.0f/s, elapsed %s)", records, workouts, rate, elapsed)
			lastLog = time.Now()
		}
	}

	if verbose {
		log.Printf("[verbose] starting XML parse...")
	}

	result, err := parser.ParseFile(path, db, progress)
	if err != nil {
		return fmt.Errorf("parsing file: %w", err)
	}

	elapsed := time.Since(start)

	fmt.Fprintf(os.Stderr, "\r")
	fmt.Printf("  Records: %d\n", result.Total)
	fmt.Printf("  Workouts: %d\n", result.Workouts)
	fmt.Printf("  Duration: %s\n\n", elapsed.Round(time.Millisecond))

	if verbose {
		rate := float64(result.Total+result.Workouts) / elapsed.Seconds()
		log.Printf("[verbose] parse complete: %.0f records/sec", rate)
	}

	// Print per-table counts
	fmt.Println("Table counts:")
	for _, name := range storage.ValidTableNames() {
		count, err := db.CountRows(name)
		if err != nil {
			if verbose {
				log.Printf("[verbose] error counting %s: %v", name, err)
			}
			continue
		}
		fmt.Printf("  %-12s %d\n", name, count)
	}

	return nil
}
