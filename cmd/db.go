package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"

	"github.com/BRO3886/healthsync/internal/storage"
)

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Database management commands",
}

var dbInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show database statistics",
	RunE:  runDBInfo,
}

func init() {
	dbCmd.AddCommand(dbInfoCmd)
	rootCmd.AddCommand(dbCmd)
}

func runDBInfo(cmd *cobra.Command, args []string) error {
	db, err := storage.Open(dbPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	info, err := db.GetDBInfo(dbPath)
	if err != nil {
		return fmt.Errorf("getting db info: %w", err)
	}

	fmt.Printf("Database: %s (%.1f MB)\n", info.Path, info.FileSizeMB)
	fmt.Printf("Total rows: %s across %d tables\n\n", formatCommas(info.TotalRows), len(info.Tables))

	if len(info.Tables) == 0 {
		fmt.Println("No data found. Run 'healthsync parse <file>' to import data.")
		return nil
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleLight)
	t.Style().Format.Header = text.FormatUpper
	t.Style().Options.SeparateRows = false

	headers := []string{"TABLE", "ROWS", "LAST RECORD"}
	t.SetColumnConfigs([]table.ColumnConfig{
		{Name: "TABLE", Align: text.AlignLeft, AlignHeader: text.AlignLeft},
		{Name: "ROWS", Align: text.AlignRight, AlignHeader: text.AlignRight},
		{Name: "LAST RECORD", Align: text.AlignLeft, AlignHeader: text.AlignLeft},
	})

	header := make(table.Row, len(headers))
	for i, h := range headers {
		header[i] = h
	}
	t.AppendHeader(header)

	for _, ti := range info.Tables {
		t.AppendRow(table.Row{
			strings.ReplaceAll(ti.Name, "_", "-"),
			formatCommas(ti.RowCount),
			ti.LastDate,
		})
	}

	t.Render()
	return nil
}
