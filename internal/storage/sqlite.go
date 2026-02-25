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

		// Cardiac vitals
		`CREATE TABLE IF NOT EXISTS resting_heart_rate (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_name TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			value REAL NOT NULL,
			unit TEXT NOT NULL,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source_name, start_date, end_date, value)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_resting_heart_rate_start_date ON resting_heart_rate(start_date)`,

		`CREATE TABLE IF NOT EXISTS hrv (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_name TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			value REAL NOT NULL,
			unit TEXT NOT NULL,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source_name, start_date, end_date, value)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_hrv_start_date ON hrv(start_date)`,

		`CREATE TABLE IF NOT EXISTS heart_rate_recovery (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_name TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			value REAL NOT NULL,
			unit TEXT NOT NULL,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source_name, start_date, end_date, value)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_heart_rate_recovery_start_date ON heart_rate_recovery(start_date)`,

		`CREATE TABLE IF NOT EXISTS respiratory_rate (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_name TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			value REAL NOT NULL,
			unit TEXT NOT NULL,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source_name, start_date, end_date, value)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_respiratory_rate_start_date ON respiratory_rate(start_date)`,

		`CREATE TABLE IF NOT EXISTS blood_pressure (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_name TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			systolic REAL NOT NULL,
			diastolic REAL NOT NULL,
			unit TEXT NOT NULL,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source_name, start_date, end_date, systolic, diastolic)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_blood_pressure_start_date ON blood_pressure(start_date)`,

		// Activity / Energy
		`CREATE TABLE IF NOT EXISTS active_energy (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_name TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			value REAL NOT NULL,
			unit TEXT NOT NULL,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source_name, start_date, end_date, value)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_active_energy_start_date ON active_energy(start_date)`,

		`CREATE TABLE IF NOT EXISTS basal_energy (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_name TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			value REAL NOT NULL,
			unit TEXT NOT NULL,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source_name, start_date, end_date, value)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_basal_energy_start_date ON basal_energy(start_date)`,

		`CREATE TABLE IF NOT EXISTS exercise_time (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_name TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			value REAL NOT NULL,
			unit TEXT NOT NULL,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source_name, start_date, end_date, value)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_exercise_time_start_date ON exercise_time(start_date)`,

		`CREATE TABLE IF NOT EXISTS stand_time (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_name TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			value REAL NOT NULL,
			unit TEXT NOT NULL,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source_name, start_date, end_date, value)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_stand_time_start_date ON stand_time(start_date)`,

		`CREATE TABLE IF NOT EXISTS flights_climbed (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_name TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			value REAL NOT NULL,
			unit TEXT NOT NULL,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source_name, start_date, end_date, value)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_flights_climbed_start_date ON flights_climbed(start_date)`,

		// Distance
		`CREATE TABLE IF NOT EXISTS distance_walking_running (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_name TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			value REAL NOT NULL,
			unit TEXT NOT NULL,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source_name, start_date, end_date, value)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_distance_walking_running_start_date ON distance_walking_running(start_date)`,

		`CREATE TABLE IF NOT EXISTS distance_cycling (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_name TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			value REAL NOT NULL,
			unit TEXT NOT NULL,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source_name, start_date, end_date, value)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_distance_cycling_start_date ON distance_cycling(start_date)`,

		// Body composition
		`CREATE TABLE IF NOT EXISTS body_mass (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_name TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			value REAL NOT NULL,
			unit TEXT NOT NULL,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source_name, start_date, end_date, value)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_body_mass_start_date ON body_mass(start_date)`,

		`CREATE TABLE IF NOT EXISTS body_mass_index (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_name TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			value REAL NOT NULL,
			unit TEXT NOT NULL,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source_name, start_date, end_date, value)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_body_mass_index_start_date ON body_mass_index(start_date)`,

		`CREATE TABLE IF NOT EXISTS height (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_name TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			value REAL NOT NULL,
			unit TEXT NOT NULL,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source_name, start_date, end_date, value)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_height_start_date ON height(start_date)`,

		// Mobility / Walking
		`CREATE TABLE IF NOT EXISTS walking_speed (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_name TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			value REAL NOT NULL,
			unit TEXT NOT NULL,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source_name, start_date, end_date, value)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_walking_speed_start_date ON walking_speed(start_date)`,

		`CREATE TABLE IF NOT EXISTS walking_step_length (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_name TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			value REAL NOT NULL,
			unit TEXT NOT NULL,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source_name, start_date, end_date, value)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_walking_step_length_start_date ON walking_step_length(start_date)`,

		`CREATE TABLE IF NOT EXISTS walking_asymmetry (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_name TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			value REAL NOT NULL,
			unit TEXT NOT NULL,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source_name, start_date, end_date, value)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_walking_asymmetry_start_date ON walking_asymmetry(start_date)`,

		`CREATE TABLE IF NOT EXISTS walking_double_support (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_name TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			value REAL NOT NULL,
			unit TEXT NOT NULL,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source_name, start_date, end_date, value)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_walking_double_support_start_date ON walking_double_support(start_date)`,

		`CREATE TABLE IF NOT EXISTS walking_steadiness (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_name TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			value REAL NOT NULL,
			unit TEXT NOT NULL,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source_name, start_date, end_date, value)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_walking_steadiness_start_date ON walking_steadiness(start_date)`,

		`CREATE TABLE IF NOT EXISTS stair_ascent_speed (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_name TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			value REAL NOT NULL,
			unit TEXT NOT NULL,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source_name, start_date, end_date, value)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_stair_ascent_speed_start_date ON stair_ascent_speed(start_date)`,

		`CREATE TABLE IF NOT EXISTS stair_descent_speed (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_name TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			value REAL NOT NULL,
			unit TEXT NOT NULL,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source_name, start_date, end_date, value)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_stair_descent_speed_start_date ON stair_descent_speed(start_date)`,

		`CREATE TABLE IF NOT EXISTS six_minute_walk (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_name TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			value REAL NOT NULL,
			unit TEXT NOT NULL,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source_name, start_date, end_date, value)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_six_minute_walk_start_date ON six_minute_walk(start_date)`,

		// Running metrics
		`CREATE TABLE IF NOT EXISTS running_speed (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_name TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			value REAL NOT NULL,
			unit TEXT NOT NULL,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source_name, start_date, end_date, value)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_running_speed_start_date ON running_speed(start_date)`,

		`CREATE TABLE IF NOT EXISTS running_power (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_name TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			value REAL NOT NULL,
			unit TEXT NOT NULL,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source_name, start_date, end_date, value)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_running_power_start_date ON running_power(start_date)`,

		`CREATE TABLE IF NOT EXISTS running_stride_length (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_name TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			value REAL NOT NULL,
			unit TEXT NOT NULL,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source_name, start_date, end_date, value)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_running_stride_length_start_date ON running_stride_length(start_date)`,

		`CREATE TABLE IF NOT EXISTS running_ground_contact_time (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_name TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			value REAL NOT NULL,
			unit TEXT NOT NULL,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source_name, start_date, end_date, value)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_running_ground_contact_time_start_date ON running_ground_contact_time(start_date)`,

		`CREATE TABLE IF NOT EXISTS running_vertical_oscillation (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_name TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			value REAL NOT NULL,
			unit TEXT NOT NULL,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source_name, start_date, end_date, value)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_running_vertical_oscillation_start_date ON running_vertical_oscillation(start_date)`,

		// Other quantity types
		`CREATE TABLE IF NOT EXISTS wrist_temperature (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_name TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			value REAL NOT NULL,
			unit TEXT NOT NULL,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source_name, start_date, end_date, value)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_wrist_temperature_start_date ON wrist_temperature(start_date)`,

		`CREATE TABLE IF NOT EXISTS time_in_daylight (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_name TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			value REAL NOT NULL,
			unit TEXT NOT NULL,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source_name, start_date, end_date, value)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_time_in_daylight_start_date ON time_in_daylight(start_date)`,

		`CREATE TABLE IF NOT EXISTS dietary_water (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_name TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			value REAL NOT NULL,
			unit TEXT NOT NULL,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source_name, start_date, end_date, value)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_dietary_water_start_date ON dietary_water(start_date)`,

		`CREATE TABLE IF NOT EXISTS physical_effort (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_name TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			value REAL NOT NULL,
			unit TEXT NOT NULL,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source_name, start_date, end_date, value)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_physical_effort_start_date ON physical_effort(start_date)`,

		`CREATE TABLE IF NOT EXISTS walking_heart_rate (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_name TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			value REAL NOT NULL,
			unit TEXT NOT NULL,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source_name, start_date, end_date, value)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_walking_heart_rate_start_date ON walking_heart_rate(start_date)`,

		// Category types (no unit — value TEXT)
		`CREATE TABLE IF NOT EXISTS mindful_sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_name TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			value TEXT NOT NULL,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source_name, start_date, end_date, value)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_mindful_sessions_start_date ON mindful_sessions(start_date)`,

		`CREATE TABLE IF NOT EXISTS stand_hours (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_name TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			value TEXT NOT NULL,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source_name, start_date, end_date, value)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_stand_hours_start_date ON stand_hours(start_date)`,
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
