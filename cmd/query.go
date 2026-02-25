package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"

	"github.com/BRO3886/healthsync/internal/storage"
)

var (
	queryFrom   string
	queryTo     string
	queryLimit  int
	queryFormat string
	queryTotal  bool
)

var queryCmd = &cobra.Command{
	Use:   "query <table>",
	Short: "Query health data from the database",
	Long: fmt.Sprintf(`Query health data from the local SQLite database.

Available tables: %s`, strings.Join(storage.ValidTableNames(), ", ")),
	Args: cobra.ExactArgs(1),
	RunE: runQuery,
}

func init() {
	queryCmd.Flags().StringVar(&queryFrom, "from", "", "filter records from this date (inclusive, e.g. 2024-01-01)")
	queryCmd.Flags().StringVar(&queryTo, "to", "", "filter records to this date (inclusive, e.g. 2024-12-31)")
	queryCmd.Flags().IntVar(&queryLimit, "limit", 50, "maximum number of records to return")
	queryCmd.Flags().StringVar(&queryFormat, "format", "table", "output format: table, json, csv")
	queryCmd.Flags().BoolVar(&queryTotal, "total", false, "show deduplicated daily totals (steps, active-energy, basal-energy)")
	rootCmd.AddCommand(queryCmd)
}

func runQuery(cmd *cobra.Command, args []string) error {
	table := args[0]

	if _, ok := storage.TableNameMap[table]; !ok {
		return fmt.Errorf("unknown table: %q (valid: %s)", table, strings.Join(storage.ValidTableNames(), ", "))
	}

	totalSupportedTables := map[string]bool{
		"steps": true, "active-energy": true, "active_energy": true,
		"basal-energy": true, "basal_energy": true,
	}
	if queryTotal && !totalSupportedTables[table] {
		return fmt.Errorf("--total is only supported for: steps, active-energy, basal-energy")
	}

	db, err := storage.Open(dbPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	params := storage.QueryParams{
		Table: table,
		From:  queryFrom,
		To:    queryTo,
		Limit: queryLimit,
	}

	var rows []map[string]interface{}
	switch {
	case queryTotal && table == "steps":
		rows, err = db.QueryStepsDailyTotal(params)
	case queryTotal && (table == "active-energy" || table == "active_energy"):
		rows, err = db.QueryActiveEnergyDailyTotal(params)
	case queryTotal && (table == "basal-energy" || table == "basal_energy"):
		rows, err = db.QueryBasalEnergyDailyTotal(params)
	default:
		rows, err = db.QueryRows(params)
	}
	if err != nil {
		return err
	}

	if len(rows) == 0 {
		fmt.Println("No records found.")
		return nil
	}

	switch queryFormat {
	case "json":
		return outputJSON(rows)
	case "csv":
		return outputCSV(rows)
	case "table":
		return outputTable(rows)
	default:
		return fmt.Errorf("unknown format: %q (valid: table, json, csv)", queryFormat)
	}
}

func outputJSON(rows []map[string]any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(rows)
}

func outputCSV(rows []map[string]any) error {
	if len(rows) == 0 {
		return nil
	}

	columns := sortedKeys(rows[0])
	w := csv.NewWriter(os.Stdout)
	defer w.Flush()

	w.Write(columns)
	for _, row := range rows {
		record := make([]string, len(columns))
		for i, col := range columns {
			record[i] = fmt.Sprintf("%v", row[col])
		}
		w.Write(record)
	}
	return nil
}

func outputTable(rows []map[string]any) error {
	if len(rows) == 0 {
		return nil
	}

	columns := sortedKeys(rows[0])
	// Filter out id and created_at for cleaner output
	filtered := make([]string, 0, len(columns))
	for _, c := range columns {
		if c != "id" && c != "created_at" {
			filtered = append(filtered, c)
		}
	}
	columns = filtered

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleLight)
	t.Style().Format.Header = text.FormatUpper
	t.Style().Options.SeparateRows = false

	colCfgs := make([]table.ColumnConfig, len(columns))
	for i, col := range columns {
		colCfgs[i] = table.ColumnConfig{
			Name:        strings.ToUpper(col),
			Align:       text.AlignLeft,
			AlignHeader: text.AlignLeft,
		}
	}
	t.SetColumnConfigs(colCfgs)

	header := make(table.Row, len(columns))
	for i, col := range columns {
		header[i] = strings.ToUpper(col)
	}
	t.AppendHeader(header)

	for _, row := range rows {
		r := make(table.Row, len(columns))
		for i, col := range columns {
			val := row[col]
			if val == nil {
				r[i] = ""
			} else {
				r[i] = fmt.Sprintf("%v", val)
			}
		}
		t.AppendRow(r)
	}

	t.Render()
	return nil
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
