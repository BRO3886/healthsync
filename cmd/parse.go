package cmd

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/BRO3886/healthsync/internal/parser"
	"github.com/BRO3886/healthsync/internal/storage"
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
	spinFrames := []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}
	frame := 0

	progress := func(records int64, workouts int64) {
		spin := spinFrames[frame%len(spinFrames)]
		frame++
		elapsed := time.Since(start).Round(time.Second)
		rate := float64(records+workouts) / time.Since(start).Seconds()
		fmt.Fprintf(os.Stderr, "\r  %s %s records · %s workouts · %.0f/s · %s   ",
			spin,
			formatCommas(records),
			formatCommas(workouts),
			rate,
			elapsed,
		)
		if verbose && time.Since(lastLog) > 5*time.Second {
			elapsed := time.Since(start).Round(time.Millisecond)
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

	fmt.Fprintf(os.Stderr, "\r\033[2K") // clear the spinner line
	fmt.Printf("  Records:  %s\n", formatCommas(result.Total))
	fmt.Printf("  Workouts: %s\n", formatCommas(result.Workouts))
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

// formatCommas formats an int64 with thousands separators (e.g. 1234567 → "1,234,567").
func formatCommas(n int64) string {
	s := strconv.FormatInt(n, 10)
	out := make([]byte, 0, len(s)+(len(s)-1)/3)
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, byte(c))
	}
	return string(out)
}
