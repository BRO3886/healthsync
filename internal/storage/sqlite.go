package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

// DB wraps the sql.DB connection and provides health data operations.
type DB struct {
	conn *sql.DB
}

// DefaultDBPath returns ~/.healthsync/healthsync.db
func DefaultDBPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "healthsync.db"
	}
	return filepath.Join(home, ".healthsync", "healthsync.db")
}

// Open opens (or creates) the SQLite database at the given path.
func Open(dbPath string) (*DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating db directory: %w", err)
	}

	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Enable WAL mode and foreign keys
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA foreign_keys=ON",
	}
	for _, p := range pragmas {
		if _, err := conn.Exec(p); err != nil {
			conn.Close()
			return nil, fmt.Errorf("setting pragma %q: %w", p, err)
		}
	}

	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("migrating schema: %w", err)
	}

	return db, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}

// Conn returns the underlying *sql.DB for direct access.
func (db *DB) Conn() *sql.DB {
	return db.conn
}

func (db *DB) migrate() error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS heart_rate (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_name TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			value REAL NOT NULL,
			unit TEXT NOT NULL,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source_name, start_date, end_date, value)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_heart_rate_start_date ON heart_rate(start_date)`,

		`CREATE TABLE IF NOT EXISTS steps (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_name TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			value REAL NOT NULL,
			unit TEXT NOT NULL DEFAULT 'count',
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source_name, start_date, end_date, value)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_steps_start_date ON steps(start_date)`,

		`CREATE TABLE IF NOT EXISTS spo2 (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_name TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			value REAL NOT NULL,
			unit TEXT NOT NULL DEFAULT '%',
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source_name, start_date, end_date, value)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_spo2_start_date ON spo2(start_date)`,

		`CREATE TABLE IF NOT EXISTS vo2_max (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_name TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			value REAL NOT NULL,
			unit TEXT NOT NULL,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source_name, start_date, end_date, value)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_vo2_max_start_date ON vo2_max(start_date)`,

		`CREATE TABLE IF NOT EXISTS sleep (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_name TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			value TEXT NOT NULL,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source_name, start_date, end_date, value)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_sleep_start_date ON sleep(start_date)`,

		`CREATE TABLE IF NOT EXISTS workouts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			activity_type TEXT NOT NULL,
			source_name TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			duration REAL,
			duration_unit TEXT,
			total_distance REAL,
			total_distance_unit TEXT,
			total_energy_burned REAL,
			total_energy_burned_unit TEXT,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(activity_type, start_date, end_date, source_name)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_workouts_start_date ON workouts(start_date)`,
	}

	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, stmt := range statements {
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("executing %q: %w", stmt[:40], err)
		}
	}

	return tx.Commit()
}

// InsertStats tracks how many rows were inserted vs skipped (duplicates).
type InsertStats struct {
	Table    string
	Inserted int64
	Skipped  int64
}

// BatchInsertRecords inserts health records in batches using INSERT OR IGNORE.
// records should be a slice of []interface{} where each element is a row's values.
func (db *DB) BatchInsertRecords(table string, columns []string, records [][]interface{}) (*InsertStats, error) {
	if len(records) == 0 {
		return &InsertStats{Table: table}, nil
	}

	stats := &InsertStats{Table: table}
	batchSize := 1000

	placeholders := "(" + strings.Repeat("?,", len(columns)-1) + "?)"
	baseQuery := fmt.Sprintf("INSERT OR IGNORE INTO %s (%s) VALUES ", table, strings.Join(columns, ", "))

	for i := 0; i < len(records); i += batchSize {
		end := i + batchSize
		if end > len(records) {
			end = len(records)
		}
		batch := records[i:end]

		tx, err := db.conn.Begin()
		if err != nil {
			return stats, fmt.Errorf("beginning transaction: %w", err)
		}

		valuePlaceholders := make([]string, len(batch))
		args := make([]interface{}, 0, len(batch)*len(columns))
		for j, row := range batch {
			valuePlaceholders[j] = placeholders
			args = append(args, row...)
		}

		query := baseQuery + strings.Join(valuePlaceholders, ", ")
		result, err := tx.Exec(query, args...)
		if err != nil {
			tx.Rollback()
			return stats, fmt.Errorf("inserting batch into %s: %w", table, err)
		}

		rowsAffected, _ := result.RowsAffected()
		stats.Inserted += rowsAffected
		stats.Skipped += int64(len(batch)) - rowsAffected

		if err := tx.Commit(); err != nil {
			return stats, fmt.Errorf("committing transaction: %w", err)
		}
	}

	return stats, nil
}
