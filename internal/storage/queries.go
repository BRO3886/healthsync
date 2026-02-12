package storage

import (
	"database/sql"
	"fmt"
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
