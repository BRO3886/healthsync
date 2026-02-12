package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/BRO3886/healthsync/internal/storage"
)

var (
	queryFrom   string
	queryTo     string
	queryLimit  int
	queryFormat string
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
	rootCmd.AddCommand(queryCmd)
}

func runQuery(cmd *cobra.Command, args []string) error {
	table := args[0]

	if _, ok := storage.TableNameMap[table]; !ok {
		return fmt.Errorf("unknown table: %q (valid: %s)", table, strings.Join(storage.ValidTableNames(), ", "))
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

	rows, err := db.QueryRows(params)
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

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Header
	for i, col := range columns {
		if i > 0 {
			fmt.Fprint(w, "\t")
		}
		fmt.Fprint(w, strings.ToUpper(col))
	}
	fmt.Fprintln(w)

	// Separator
	for i, col := range columns {
		if i > 0 {
			fmt.Fprint(w, "\t")
		}
		fmt.Fprint(w, strings.Repeat("-", len(col)+2))
	}
	fmt.Fprintln(w)

	// Rows
	for _, row := range rows {
		for i, col := range columns {
			if i > 0 {
				fmt.Fprint(w, "\t")
			}
			val := row[col]
			if val == nil {
				fmt.Fprint(w, "")
			} else {
				fmt.Fprintf(w, "%v", val)
			}
		}
		fmt.Fprintln(w)
	}

	return w.Flush()
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
