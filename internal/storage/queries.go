package storage

import (
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// QueryParams holds filters for querying health data.
type QueryParams struct {
	Table  string
	From   string // start date filter (inclusive)
	To     string // end date filter (inclusive)
	Limit  int
	Offset int
}

// TableNameMap maps CLI-friendly names to actual table names.
var TableNameMap = map[string]string{
	"heart-rate": "heart_rate",
	"heart_rate": "heart_rate",
	"steps":      "steps",
	"spo2":       "spo2",
	"vo2max":     "vo2_max",
	"vo2_max":    "vo2_max",
	"sleep":      "sleep",
	"workouts":   "workouts",
}

// ValidTableNames returns the list of valid CLI table names.
func ValidTableNames() []string {
	return []string{"heart-rate", "steps", "spo2", "vo2max", "sleep", "workouts"}
}

// QueryRows executes a query against the specified table and returns rows as maps.
func (db *DB) QueryRows(params QueryParams) ([]map[string]interface{}, error) {
	tableName, ok := TableNameMap[params.Table]
	if !ok {
		return nil, fmt.Errorf("unknown table: %q (valid: %s)", params.Table, strings.Join(ValidTableNames(), ", "))
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE 1=1", tableName)
	var args []interface{}

	if params.From != "" {
		query += " AND start_date >= ?"
		args = append(args, params.From)
	}
	if params.To != "" {
		query += " AND start_date <= ?"
		args = append(args, params.To)
	}

	query += " ORDER BY start_date DESC"

	if params.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, params.Limit)
	}
	if params.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, params.Offset)
	}

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying %s: %w", tableName, err)
	}
	defer rows.Close()

	return scanRows(rows)
}

// CountRows returns the total row count for a table.
func (db *DB) CountRows(table string) (int64, error) {
	tableName, ok := TableNameMap[table]
	if !ok {
		return 0, fmt.Errorf("unknown table: %q", table)
	}

	var count int64
	err := db.conn.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)).Scan(&count)
	return count, err
}

func scanRows(rows *sql.Rows) ([]map[string]interface{}, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		row := make(map[string]interface{}, len(columns))
		for i, col := range columns {
			val := values[i]
			// Convert []byte to string for readability
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		results = append(results, row)
	}

	return results, rows.Err()
}

// sourcePriority returns a numeric priority for a source name.
// Apple Watch > iPhone > everything else.
func sourcePriority(source string) int {
	s := strings.ToLower(source)
	if strings.Contains(s, "apple watch") {
		return 2
	}
	if strings.Contains(s, "iphone") {
		return 1
	}
	return 0
}

// stepsRecord is an internal type for the dedup algorithm.
type stepsRecord struct {
	source    string
	startDate string
	endDate   string
	value     float64
	unit      string
}

// deduplicateSteps removes overlapping step records, preferring higher-priority sources.
func deduplicateSteps(records []stepsRecord) []stepsRecord {
	// Sort by start_date ASC
	sort.Slice(records, func(i, j int) bool {
		return records[i].startDate < records[j].startDate
	})

	var accepted []stepsRecord

	for _, r := range records {
		rPriority := sourcePriority(r.source)
		overlaps := false

		for i := len(accepted) - 1; i >= 0; i-- {
			a := accepted[i]
			// Since accepted is sorted by start_date, if a.endDate <= r.startDate
			// then no earlier accepted records can overlap either.
			if a.endDate <= r.startDate {
				break
			}
			// Intervals overlap: a.startDate < r.endDate && r.startDate < a.endDate
			if a.startDate < r.endDate && r.startDate < a.endDate {
				overlaps = true
				if rPriority > sourcePriority(a.source) {
					// Replace the lower-priority accepted record
					accepted = append(accepted[:i], accepted[i+1:]...)
					accepted = append(accepted, r)
				}
				break
			}
		}

		if !overlaps {
			accepted = append(accepted, r)
		}
	}

	return accepted
}

// QueryStepsDailyTotal returns deduplicated step totals aggregated by calendar day.
func (db *DB) QueryStepsDailyTotal(params QueryParams) ([]map[string]interface{}, error) {
	// Fetch all step records for the date range (no limit)
	query := "SELECT source_name, start_date, end_date, value, unit FROM steps WHERE 1=1"
	var args []interface{}

	if params.From != "" {
		query += " AND start_date >= ?"
		args = append(args, params.From)
	}
	if params.To != "" {
		query += " AND start_date <= ?"
		args = append(args, params.To)
	}

	query += " ORDER BY start_date ASC"

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying steps for dedup: %w", err)
	}
	defer rows.Close()

	var records []stepsRecord
	for rows.Next() {
		var r stepsRecord
		if err := rows.Scan(&r.source, &r.startDate, &r.endDate, &r.value, &r.unit); err != nil {
			return nil, fmt.Errorf("scanning step row: %w", err)
		}
		records = append(records, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	deduped := deduplicateSteps(records)

	// Aggregate by calendar day (first 10 chars of start_date = "YYYY-MM-DD")
	dailyTotals := make(map[string]float64)
	var dayOrder []string
	for _, r := range deduped {
		day := r.startDate[:10]
		if _, exists := dailyTotals[day]; !exists {
			dayOrder = append(dayOrder, day)
		}
		dailyTotals[day] += r.value
	}

	sort.Strings(dayOrder)

	results := make([]map[string]interface{}, 0, len(dayOrder))
	for _, day := range dayOrder {
		results = append(results, map[string]interface{}{
			"date":  day,
			"total": strconv.FormatFloat(dailyTotals[day], 'f', 0, 64),
		})
	}

	return results, nil
}
