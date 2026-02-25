package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func tempDB(t *testing.T) *DB {
	t.Helper()
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("opening test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestDefaultDBPath(t *testing.T) {
	p := DefaultDBPath()
	if p == "" {
		t.Fatal("DefaultDBPath returned empty string")
	}
	if filepath.Base(p) != "healthsync.db" {
		t.Errorf("expected healthsync.db, got %s", filepath.Base(p))
	}
	if !filepath.IsAbs(p) {
		t.Errorf("expected absolute path, got %s", p)
	}
}

func TestOpen_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "sub", "nested", "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	db.Close()

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("database file was not created")
	}
}

func TestOpen_InvalidPath(t *testing.T) {
	_, err := Open("/dev/null/impossible/test.db")
	if err == nil {
		t.Fatal("expected error for invalid path")
	}
}

func TestOpen_IdempotentMigration(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	// Open and close twice — migration should be idempotent
	db1, err := Open(dbPath)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	db1.Close()

	db2, err := Open(dbPath)
	if err != nil {
		t.Fatalf("second open: %v", err)
	}
	db2.Close()
}

func TestConn_ReturnsUnderlyingDB(t *testing.T) {
	db := tempDB(t)
	if db.Conn() == nil {
		t.Fatal("Conn() returned nil")
	}
}

// --- BatchInsertRecords ---

func TestBatchInsert_EmptyRecords(t *testing.T) {
	db := tempDB(t)
	stats, err := db.BatchInsertRecords("heart_rate", []string{"source_name", "start_date", "end_date", "value", "unit"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.Inserted != 0 || stats.Skipped != 0 {
		t.Errorf("expected 0/0, got %d/%d", stats.Inserted, stats.Skipped)
	}
}

func TestBatchInsert_SingleRecord(t *testing.T) {
	db := tempDB(t)
	cols := []string{"source_name", "start_date", "end_date", "value", "unit"}
	records := [][]any{
		{"Watch", "2024-01-01 00:00:00", "2024-01-01 00:01:00", 72.0, "count/min"},
	}
	stats, err := db.BatchInsertRecords("heart_rate", cols, records)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.Inserted != 1 {
		t.Errorf("expected 1 inserted, got %d", stats.Inserted)
	}
}

func TestBatchInsert_ExactBatchBoundary(t *testing.T) {
	db := tempDB(t)
	cols := []string{"source_name", "start_date", "end_date", "value", "unit"}
	records := make([][]any, 1000)
	for i := range records {
		records[i] = []any{"Watch", "2024-01-01 00:00:00", "2024-01-01 00:01:00", float64(i), "count/min"}
	}
	stats, err := db.BatchInsertRecords("heart_rate", cols, records)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.Inserted != 1000 {
		t.Errorf("expected 1000 inserted, got %d", stats.Inserted)
	}
}

func TestBatchInsert_CrossesBatchBoundary(t *testing.T) {
	db := tempDB(t)
	cols := []string{"source_name", "start_date", "end_date", "value", "unit"}
	records := make([][]any, 1001)
	for i := range records {
		records[i] = []any{"Watch", "2024-01-01 00:00:00", "2024-01-01 00:01:00", float64(i), "count/min"}
	}
	stats, err := db.BatchInsertRecords("heart_rate", cols, records)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.Inserted != 1001 {
		t.Errorf("expected 1001 inserted, got %d", stats.Inserted)
	}
}

func TestBatchInsert_Dedup(t *testing.T) {
	db := tempDB(t)
	cols := []string{"source_name", "start_date", "end_date", "value", "unit"}
	row := []any{"Watch", "2024-01-01 00:00:00", "2024-01-01 00:01:00", 72.0, "count/min"}

	// Insert once
	stats1, err := db.BatchInsertRecords("heart_rate", cols, [][]any{row})
	if err != nil {
		t.Fatalf("first insert: %v", err)
	}
	if stats1.Inserted != 1 {
		t.Errorf("first: expected 1 inserted, got %d", stats1.Inserted)
	}

	// Insert same row again — should be ignored
	stats2, err := db.BatchInsertRecords("heart_rate", cols, [][]any{row})
	if err != nil {
		t.Fatalf("second insert: %v", err)
	}
	if stats2.Inserted != 0 {
		t.Errorf("second: expected 0 inserted, got %d", stats2.Inserted)
	}
	if stats2.Skipped != 1 {
		t.Errorf("second: expected 1 skipped, got %d", stats2.Skipped)
	}
}

func TestBatchInsert_MixNewAndDuplicate(t *testing.T) {
	db := tempDB(t)
	cols := []string{"source_name", "start_date", "end_date", "value", "unit"}

	existing := []any{"Watch", "2024-01-01 00:00:00", "2024-01-01 00:01:00", 72.0, "count/min"}
	db.BatchInsertRecords("heart_rate", cols, [][]any{existing})

	mixed := [][]any{
		existing,                                                                              // duplicate
		{"Watch", "2024-01-01 00:02:00", "2024-01-01 00:03:00", 75.0, "count/min"},           // new
		{"Watch", "2024-01-01 00:04:00", "2024-01-01 00:05:00", 80.0, "count/min"},           // new
	}
	stats, err := db.BatchInsertRecords("heart_rate", cols, mixed)
	if err != nil {
		t.Fatalf("mixed insert: %v", err)
	}
	if stats.Inserted != 2 {
		t.Errorf("expected 2 inserted, got %d", stats.Inserted)
	}
	if stats.Skipped != 1 {
		t.Errorf("expected 1 skipped, got %d", stats.Skipped)
	}
}

func TestBatchInsert_InvalidTable(t *testing.T) {
	db := tempDB(t)
	_, err := db.BatchInsertRecords("nonexistent_table", []string{"col"}, [][]any{{"val"}})
	if err == nil {
		t.Fatal("expected error for invalid table")
	}
}

func TestBatchInsert_SleepTable(t *testing.T) {
	db := tempDB(t)
	cols := []string{"source_name", "start_date", "end_date", "value"}
	records := [][]any{
		{"Watch", "2024-01-01 22:00:00", "2024-01-02 06:00:00", "HKCategoryValueSleepAnalysisAsleepCore"},
	}
	stats, err := db.BatchInsertRecords("sleep", cols, records)
	if err != nil {
		t.Fatalf("sleep insert: %v", err)
	}
	if stats.Inserted != 1 {
		t.Errorf("expected 1 inserted, got %d", stats.Inserted)
	}
}

func TestBatchInsert_WorkoutsTable(t *testing.T) {
	db := tempDB(t)
	cols := []string{
		"activity_type", "source_name", "start_date", "end_date",
		"duration", "duration_unit",
		"total_distance", "total_distance_unit",
		"total_energy_burned", "total_energy_burned_unit",
	}
	records := [][]any{
		{"HKWorkoutActivityTypeRunning", "Watch", "2024-01-01 08:00:00", "2024-01-01 08:30:00",
			30.0, "min", 5.0, "km", 300.0, "kcal"},
	}
	stats, err := db.BatchInsertRecords("workouts", cols, records)
	if err != nil {
		t.Fatalf("workout insert: %v", err)
	}
	if stats.Inserted != 1 {
		t.Errorf("expected 1 inserted, got %d", stats.Inserted)
	}
}

func TestBatchInsert_WorkoutsNullOptionalFields(t *testing.T) {
	db := tempDB(t)
	cols := []string{
		"activity_type", "source_name", "start_date", "end_date",
		"duration", "duration_unit",
		"total_distance", "total_distance_unit",
		"total_energy_burned", "total_energy_burned_unit",
	}
	records := [][]any{
		{"HKWorkoutActivityTypeYoga", "Watch", "2024-01-01 08:00:00", "2024-01-01 09:00:00",
			60.0, "min", nil, nil, nil, nil},
	}
	stats, err := db.BatchInsertRecords("workouts", cols, records)
	if err != nil {
		t.Fatalf("workout insert: %v", err)
	}
	if stats.Inserted != 1 {
		t.Errorf("expected 1 inserted, got %d", stats.Inserted)
	}
}

// --- QueryRows ---

func TestQueryRows_UnknownTable(t *testing.T) {
	db := tempDB(t)
	_, err := db.QueryRows(QueryParams{Table: "bogus"})
	if err == nil {
		t.Fatal("expected error for unknown table")
	}
}

func TestQueryRows_EmptyTable(t *testing.T) {
	db := tempDB(t)
	rows, err := db.QueryRows(QueryParams{Table: "heart-rate", Limit: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("expected 0 rows, got %d", len(rows))
	}
}

func TestQueryRows_WithData(t *testing.T) {
	db := tempDB(t)
	cols := []string{"source_name", "start_date", "end_date", "value", "unit"}
	records := [][]any{
		{"Watch", "2024-01-01 00:00:00", "2024-01-01 00:01:00", 72.0, "count/min"},
		{"Watch", "2024-01-02 00:00:00", "2024-01-02 00:01:00", 75.0, "count/min"},
		{"Watch", "2024-01-03 00:00:00", "2024-01-03 00:01:00", 80.0, "count/min"},
	}
	db.BatchInsertRecords("heart_rate", cols, records)

	rows, err := db.QueryRows(QueryParams{Table: "heart-rate", Limit: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 3 {
		t.Errorf("expected 3 rows, got %d", len(rows))
	}
}

func TestQueryRows_FromFilter(t *testing.T) {
	db := tempDB(t)
	cols := []string{"source_name", "start_date", "end_date", "value", "unit"}
	records := [][]any{
		{"Watch", "2024-01-01 00:00:00", "2024-01-01 00:01:00", 72.0, "count/min"},
		{"Watch", "2024-06-01 00:00:00", "2024-06-01 00:01:00", 75.0, "count/min"},
		{"Watch", "2024-12-01 00:00:00", "2024-12-01 00:01:00", 80.0, "count/min"},
	}
	db.BatchInsertRecords("heart_rate", cols, records)

	rows, err := db.QueryRows(QueryParams{Table: "heart-rate", From: "2024-06-01", Limit: 50})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("expected 2 rows with from filter, got %d", len(rows))
	}
}

func TestQueryRows_ToFilter(t *testing.T) {
	db := tempDB(t)
	cols := []string{"source_name", "start_date", "end_date", "value", "unit"}
	records := [][]any{
		{"Watch", "2024-01-01 00:00:00", "2024-01-01 00:01:00", 72.0, "count/min"},
		{"Watch", "2024-06-01 00:00:00", "2024-06-01 00:01:00", 75.0, "count/min"},
		{"Watch", "2024-12-01 00:00:00", "2024-12-01 00:01:00", 80.0, "count/min"},
	}
	db.BatchInsertRecords("heart_rate", cols, records)

	// Note: "2024-06-02" ensures the 2024-06-01 record is included
	// since start_date "2024-06-01 00:00:00" <= "2024-06-02"
	rows, err := db.QueryRows(QueryParams{Table: "heart-rate", To: "2024-06-02", Limit: 50})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("expected 2 rows with to filter, got %d", len(rows))
	}
}

func TestQueryRows_FromAndToFilter(t *testing.T) {
	db := tempDB(t)
	cols := []string{"source_name", "start_date", "end_date", "value", "unit"}
	records := [][]any{
		{"Watch", "2024-01-01 00:00:00", "2024-01-01 00:01:00", 72.0, "count/min"},
		{"Watch", "2024-06-01 00:00:00", "2024-06-01 00:01:00", 75.0, "count/min"},
		{"Watch", "2024-12-01 00:00:00", "2024-12-01 00:01:00", 80.0, "count/min"},
	}
	db.BatchInsertRecords("heart_rate", cols, records)

	rows, err := db.QueryRows(QueryParams{Table: "heart-rate", From: "2024-03-01", To: "2024-09-01", Limit: 50})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("expected 1 row with from+to filter, got %d", len(rows))
	}
}

func TestQueryRows_LimitZero(t *testing.T) {
	db := tempDB(t)
	cols := []string{"source_name", "start_date", "end_date", "value", "unit"}
	records := make([][]any, 5)
	for i := range records {
		records[i] = []any{"Watch", "2024-01-01 00:00:00", "2024-01-01 00:01:00", float64(60 + i), "count/min"}
	}
	db.BatchInsertRecords("heart_rate", cols, records)

	// Limit 0 means no LIMIT clause
	rows, err := db.QueryRows(QueryParams{Table: "heart-rate", Limit: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 5 {
		t.Errorf("expected 5 rows with no limit, got %d", len(rows))
	}
}

func TestQueryRows_CLIFriendlyNames(t *testing.T) {
	db := tempDB(t)

	// Both heart-rate and heart_rate should work
	for _, name := range []string{"heart-rate", "heart_rate"} {
		_, err := db.QueryRows(QueryParams{Table: name, Limit: 1})
		if err != nil {
			t.Errorf("table name %q should be valid, got error: %v", name, err)
		}
	}
	for _, name := range []string{"vo2max", "vo2_max"} {
		_, err := db.QueryRows(QueryParams{Table: name, Limit: 1})
		if err != nil {
			t.Errorf("table name %q should be valid, got error: %v", name, err)
		}
	}
}

func TestQueryRows_OrderDescending(t *testing.T) {
	db := tempDB(t)
	cols := []string{"source_name", "start_date", "end_date", "value", "unit"}
	records := [][]any{
		{"Watch", "2024-01-01 00:00:00", "2024-01-01 00:01:00", 72.0, "count/min"},
		{"Watch", "2024-01-03 00:00:00", "2024-01-03 00:01:00", 80.0, "count/min"},
		{"Watch", "2024-01-02 00:00:00", "2024-01-02 00:01:00", 75.0, "count/min"},
	}
	db.BatchInsertRecords("heart_rate", cols, records)

	rows, err := db.QueryRows(QueryParams{Table: "heart-rate", Limit: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be ordered by start_date DESC
	first := rows[0]["start_date"].(string)
	if first != "2024-01-03 00:00:00" {
		t.Errorf("expected first row to be 2024-01-03, got %s", first)
	}
}

// --- CountRows ---

func TestCountRows_EmptyTable(t *testing.T) {
	db := tempDB(t)
	count, err := db.CountRows("heart-rate")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}
}

func TestCountRows_UnknownTable(t *testing.T) {
	db := tempDB(t)
	_, err := db.CountRows("bogus")
	if err == nil {
		t.Fatal("expected error for unknown table")
	}
}

func TestCountRows_WithData(t *testing.T) {
	db := tempDB(t)
	cols := []string{"source_name", "start_date", "end_date", "value", "unit"}
	records := [][]any{
		{"Watch", "2024-01-01 00:00:00", "2024-01-01 00:01:00", 72.0, "count/min"},
		{"Watch", "2024-01-02 00:00:00", "2024-01-02 00:01:00", 75.0, "count/min"},
	}
	db.BatchInsertRecords("heart_rate", cols, records)

	count, err := db.CountRows("heart-rate")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}
}

// --- ValidTableNames ---

func TestValidTableNames(t *testing.T) {
	names := ValidTableNames()
	// All original names must still be present
	required := []string{"heart-rate", "steps", "spo2", "vo2max", "sleep", "workouts"}
	nameSet := make(map[string]bool, len(names))
	for _, n := range names {
		nameSet[n] = true
	}
	for _, r := range required {
		if !nameSet[r] {
			t.Errorf("required table name %q missing from ValidTableNames", r)
		}
	}
	if len(names) < 41 {
		t.Errorf("expected at least 41 table names, got %d", len(names))
	}
}

// --- TableNameMap ---

// --- Steps Dedup ---

func TestDeduplicateSteps_NoOverlap(t *testing.T) {
	records := []stepsRecord{
		{source: "iPhone", startDate: "2024-01-01 08:00:00", endDate: "2024-01-01 08:15:00", value: 100},
		{source: "iPhone", startDate: "2024-01-01 08:15:00", endDate: "2024-01-01 08:30:00", value: 200},
	}
	result := deduplicateSteps(records)
	if len(result) != 2 {
		t.Errorf("expected 2 records, got %d", len(result))
	}
	total := 0.0
	for _, r := range result {
		total += r.value
	}
	if total != 300 {
		t.Errorf("expected total 300, got %.0f", total)
	}
}

func TestDeduplicateSteps_OverlapSameSource(t *testing.T) {
	// Two records from the same source overlap — first wins
	records := []stepsRecord{
		{source: "iPhone", startDate: "2024-01-01 08:00:00", endDate: "2024-01-01 08:20:00", value: 150},
		{source: "iPhone", startDate: "2024-01-01 08:10:00", endDate: "2024-01-01 08:30:00", value: 200},
	}
	result := deduplicateSteps(records)
	if len(result) != 1 {
		t.Errorf("expected 1 record, got %d", len(result))
	}
	if result[0].value != 150 {
		t.Errorf("expected first-wins value 150, got %.0f", result[0].value)
	}
}

func TestDeduplicateSteps_OverlapWatchBeatsIPhone(t *testing.T) {
	// Watch and iPhone overlap — Watch should win
	records := []stepsRecord{
		{source: "Sid's iPhone", startDate: "2024-01-01 08:00:00", endDate: "2024-01-01 08:30:00", value: 500},
		{source: "Sid's Apple Watch", startDate: "2024-01-01 08:05:00", endDate: "2024-01-01 08:25:00", value: 300},
	}
	result := deduplicateSteps(records)
	if len(result) != 1 {
		t.Errorf("expected 1 record, got %d", len(result))
	}
	if result[0].value != 300 {
		t.Errorf("expected Watch value 300, got %.0f", result[0].value)
	}
}

func TestDeduplicateSteps_OverlapIPhoneBeatsThirdParty(t *testing.T) {
	records := []stepsRecord{
		{source: "Pedometer++", startDate: "2024-01-01 08:00:00", endDate: "2024-01-01 08:30:00", value: 600},
		{source: "Sid's iPhone", startDate: "2024-01-01 08:05:00", endDate: "2024-01-01 08:25:00", value: 400},
	}
	result := deduplicateSteps(records)
	if len(result) != 1 {
		t.Errorf("expected 1 record, got %d", len(result))
	}
	if result[0].value != 400 {
		t.Errorf("expected iPhone value 400, got %.0f", result[0].value)
	}
}

func TestDeduplicateSteps_MixedOverlapAndNon(t *testing.T) {
	records := []stepsRecord{
		{source: "Sid's Apple Watch", startDate: "2024-01-01 08:00:00", endDate: "2024-01-01 08:15:00", value: 200},
		{source: "Sid's iPhone", startDate: "2024-01-01 08:05:00", endDate: "2024-01-01 08:20:00", value: 300},
		{source: "Sid's Apple Watch", startDate: "2024-01-01 09:00:00", endDate: "2024-01-01 09:15:00", value: 150},
	}
	result := deduplicateSteps(records)
	if len(result) != 2 {
		t.Errorf("expected 2 records, got %d", len(result))
	}
	total := 0.0
	for _, r := range result {
		total += r.value
	}
	// Watch 200 (beats iPhone 300) + Watch 150 = 350
	if total != 350 {
		t.Errorf("expected total 350, got %.0f", total)
	}
}

func TestDeduplicateSteps_AdjacentNotOverlapping(t *testing.T) {
	// end_date == start_date means no overlap (intervals are [start, end))
	records := []stepsRecord{
		{source: "iPhone", startDate: "2024-01-01 08:00:00", endDate: "2024-01-01 08:15:00", value: 100},
		{source: "Apple Watch", startDate: "2024-01-01 08:15:00", endDate: "2024-01-01 08:30:00", value: 200},
	}
	result := deduplicateSteps(records)
	if len(result) != 2 {
		t.Errorf("expected 2 records (adjacent, not overlapping), got %d", len(result))
	}
}

func TestQueryStepsDailyTotal_Integration(t *testing.T) {
	db := tempDB(t)
	cols := []string{"source_name", "start_date", "end_date", "value", "unit"}

	// Day 1: Watch and iPhone overlap
	records := [][]any{
		{"Sid's Apple Watch", "2024-01-01 08:00:00", "2024-01-01 08:15:00", 200.0, "count"},
		{"Sid's iPhone", "2024-01-01 08:05:00", "2024-01-01 08:20:00", 350.0, "count"},
		{"Sid's Apple Watch", "2024-01-01 09:00:00", "2024-01-01 09:15:00", 150.0, "count"},
		// Day 2: single source
		{"Sid's Apple Watch", "2024-01-02 10:00:00", "2024-01-02 10:15:00", 500.0, "count"},
	}
	_, err := db.BatchInsertRecords("steps", cols, records)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	results, err := db.QueryStepsDailyTotal(QueryParams{Table: "steps"})
	if err != nil {
		t.Fatalf("QueryStepsDailyTotal: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 days, got %d", len(results))
	}

	// Day 1: Watch 200 wins over iPhone 350, plus Watch 150 = 350
	if results[0]["date"] != "2024-01-01" {
		t.Errorf("expected date 2024-01-01, got %v", results[0]["date"])
	}
	if results[0]["total"] != "350" {
		t.Errorf("expected day 1 total 350, got %v", results[0]["total"])
	}

	// Day 2: 500
	if results[1]["date"] != "2024-01-02" {
		t.Errorf("expected date 2024-01-02, got %v", results[1]["date"])
	}
	if results[1]["total"] != "500" {
		t.Errorf("expected day 2 total 500, got %v", results[1]["total"])
	}
}

func TestQueryStepsDailyTotal_WithDateFilters(t *testing.T) {
	db := tempDB(t)
	cols := []string{"source_name", "start_date", "end_date", "value", "unit"}
	records := [][]any{
		{"Watch", "2024-01-01 08:00:00", "2024-01-01 08:15:00", 100.0, "count"},
		{"Watch", "2024-01-02 08:00:00", "2024-01-02 08:15:00", 200.0, "count"},
		{"Watch", "2024-01-03 08:00:00", "2024-01-03 08:15:00", 300.0, "count"},
	}
	db.BatchInsertRecords("steps", cols, records)

	results, err := db.QueryStepsDailyTotal(QueryParams{
		Table: "steps",
		From:  "2024-01-02",
		To:    "2024-01-02 23:59:59",
	})
	if err != nil {
		t.Fatalf("QueryStepsDailyTotal: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 day, got %d", len(results))
	}
	if results[0]["total"] != "200" {
		t.Errorf("expected 200, got %v", results[0]["total"])
	}
}

func TestQueryStepsDailyTotal_Empty(t *testing.T) {
	db := tempDB(t)
	results, err := db.QueryStepsDailyTotal(QueryParams{Table: "steps"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestSourcePriority(t *testing.T) {
	tests := []struct {
		source   string
		expected int
	}{
		{"Sid's Apple Watch", 2},
		{"Apple Watch Series 9", 2},
		{"Sid's iPhone", 1},
		{"iPhone 15 Pro", 1},
		{"Pedometer++", 0},
		{"MyFitnessPal", 0},
	}
	for _, tt := range tests {
		got := sourcePriority(tt.source)
		if got != tt.expected {
			t.Errorf("sourcePriority(%q) = %d, want %d", tt.source, got, tt.expected)
		}
	}
}

// --- TableNameMap ---

func TestBatchInsert_NewQuantityTable(t *testing.T) {
	db := tempDB(t)
	cols := []string{"source_name", "start_date", "end_date", "value", "unit"}
	records := [][]any{
		{"Watch", "2024-01-01 00:00:00", "2024-01-01 00:01:00", 58.0, "count/min"},
	}
	stats, err := db.BatchInsertRecords("resting_heart_rate", cols, records)
	if err != nil {
		t.Fatalf("resting_heart_rate insert: %v", err)
	}
	if stats.Inserted != 1 {
		t.Errorf("expected 1 inserted, got %d", stats.Inserted)
	}
}

func TestBatchInsert_BloodPressureTable(t *testing.T) {
	db := tempDB(t)
	cols := []string{"source_name", "start_date", "end_date", "systolic", "diastolic", "unit"}
	records := [][]any{
		{"Watch", "2024-01-01 08:00:00", "2024-01-01 08:01:00", 120.0, 80.0, "mmHg"},
	}
	stats, err := db.BatchInsertRecords("blood_pressure", cols, records)
	if err != nil {
		t.Fatalf("blood_pressure insert: %v", err)
	}
	if stats.Inserted != 1 {
		t.Errorf("expected 1 inserted, got %d", stats.Inserted)
	}
}

func TestBatchInsert_MindfulSessionsTable(t *testing.T) {
	db := tempDB(t)
	cols := []string{"source_name", "start_date", "end_date", "value"}
	records := [][]any{
		{"Headspace", "2024-01-01 07:00:00", "2024-01-01 07:10:00", "HKCategoryValueNotApplicable"},
	}
	stats, err := db.BatchInsertRecords("mindful_sessions", cols, records)
	if err != nil {
		t.Fatalf("mindful_sessions insert: %v", err)
	}
	if stats.Inserted != 1 {
		t.Errorf("expected 1 inserted, got %d", stats.Inserted)
	}
}

func TestQueryActiveEnergyDailyTotal_Integration(t *testing.T) {
	db := tempDB(t)
	cols := []string{"source_name", "start_date", "end_date", "value", "unit"}
	records := [][]any{
		{"Watch", "2024-01-01 08:00:00", "2024-01-01 08:30:00", 200.0, "kcal"},
		{"Watch", "2024-01-01 09:00:00", "2024-01-01 09:30:00", 150.0, "kcal"},
		{"Watch", "2024-01-02 10:00:00", "2024-01-02 10:30:00", 400.0, "kcal"},
	}
	_, err := db.BatchInsertRecords("active_energy", cols, records)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	results, err := db.QueryActiveEnergyDailyTotal(QueryParams{Table: "active-energy"})
	if err != nil {
		t.Fatalf("QueryActiveEnergyDailyTotal: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 days, got %d", len(results))
	}
	if results[0]["date"] != "2024-01-01" {
		t.Errorf("expected 2024-01-01, got %v", results[0]["date"])
	}
	if results[0]["total"] != "350.00" {
		t.Errorf("expected 350.00, got %v", results[0]["total"])
	}
	if results[1]["total"] != "400.00" {
		t.Errorf("expected 400.00, got %v", results[1]["total"])
	}
}

func TestQueryBasalEnergyDailyTotal_Integration(t *testing.T) {
	db := tempDB(t)
	cols := []string{"source_name", "start_date", "end_date", "value", "unit"}
	records := [][]any{
		{"Watch", "2024-01-01 00:00:00", "2024-01-01 01:00:00", 70.0, "kcal"},
		{"Watch", "2024-01-01 01:00:00", "2024-01-01 02:00:00", 68.0, "kcal"},
	}
	_, err := db.BatchInsertRecords("basal_energy", cols, records)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	results, err := db.QueryBasalEnergyDailyTotal(QueryParams{Table: "basal-energy"})
	if err != nil {
		t.Fatalf("QueryBasalEnergyDailyTotal: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 day, got %d", len(results))
	}
	if results[0]["total"] != "138.00" {
		t.Errorf("expected 138.00, got %v", results[0]["total"])
	}
}

func TestQueryRows_NewTablesAccessible(t *testing.T) {
	db := tempDB(t)
	newTables := []string{
		"resting-heart-rate", "hrv", "heart-rate-recovery", "respiratory-rate",
		"blood-pressure", "active-energy", "basal-energy",
		"mindful-sessions", "stand-hours", "body-mass",
		"walking-speed", "running-speed", "wrist-temperature",
	}
	for _, tbl := range newTables {
		_, err := db.QueryRows(QueryParams{Table: tbl, Limit: 1})
		if err != nil {
			t.Errorf("table %q returned error: %v", tbl, err)
		}
	}
}

func TestTableNameMap_AllCLINamesResolve(t *testing.T) {
	for _, name := range ValidTableNames() {
		if _, ok := TableNameMap[name]; !ok {
			t.Errorf("CLI name %q not in TableNameMap", name)
		}
	}
}

func TestQueryActiveEnergyDailyTotal_WithDateFilters(t *testing.T) {
	db := tempDB(t)
	cols := []string{"source_name", "start_date", "end_date", "value", "unit"}
	records := [][]any{
		{"Watch", "2024-01-01 08:00:00", "2024-01-01 08:30:00", 300.0, "kcal"},
		{"Watch", "2024-01-02 08:00:00", "2024-01-02 08:30:00", 400.0, "kcal"},
		{"Watch", "2024-01-03 08:00:00", "2024-01-03 08:30:00", 500.0, "kcal"},
	}
	db.BatchInsertRecords("active_energy", cols, records)

	results, err := db.QueryActiveEnergyDailyTotal(QueryParams{
		Table: "active-energy",
		From:  "2024-01-02",
		To:    "2024-01-02 23:59:59",
	})
	if err != nil {
		t.Fatalf("QueryActiveEnergyDailyTotal: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 day, got %d", len(results))
	}
	if results[0]["date"] != "2024-01-02" {
		t.Errorf("expected 2024-01-02, got %v", results[0]["date"])
	}
	if results[0]["total"] != "400.00" {
		t.Errorf("expected 400.00, got %v", results[0]["total"])
	}
}

func TestQueryBasalEnergyDailyTotal_WithDateFilters(t *testing.T) {
	db := tempDB(t)
	cols := []string{"source_name", "start_date", "end_date", "value", "unit"}
	records := [][]any{
		{"Watch", "2024-01-01 00:00:00", "2024-01-01 01:00:00", 70.0, "kcal"},
		{"Watch", "2024-01-02 00:00:00", "2024-01-02 01:00:00", 68.0, "kcal"},
		{"Watch", "2024-01-03 00:00:00", "2024-01-03 01:00:00", 72.0, "kcal"},
	}
	db.BatchInsertRecords("basal_energy", cols, records)

	results, err := db.QueryBasalEnergyDailyTotal(QueryParams{
		Table: "basal-energy",
		From:  "2024-01-02",
		To:    "2024-01-02 23:59:59",
	})
	if err != nil {
		t.Fatalf("QueryBasalEnergyDailyTotal: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 day, got %d", len(results))
	}
	if results[0]["total"] != "68.00" {
		t.Errorf("expected 68.00, got %v", results[0]["total"])
	}
}

func TestQueryActiveEnergyDailyTotal_DeduplicationWatchBeatsIPhone(t *testing.T) {
	db := tempDB(t)
	cols := []string{"source_name", "start_date", "end_date", "value", "unit"}
	// Watch and iPhone overlap — Watch should win
	records := [][]any{
		{"Sid's iPhone", "2024-01-01 08:00:00", "2024-01-01 08:30:00", 500.0, "kcal"},
		{"Sid's Apple Watch", "2024-01-01 08:05:00", "2024-01-01 08:25:00", 300.0, "kcal"},
		// Non-overlapping Watch record
		{"Sid's Apple Watch", "2024-01-01 09:00:00", "2024-01-01 09:30:00", 200.0, "kcal"},
	}
	db.BatchInsertRecords("active_energy", cols, records)

	results, err := db.QueryActiveEnergyDailyTotal(QueryParams{Table: "active-energy"})
	if err != nil {
		t.Fatalf("QueryActiveEnergyDailyTotal: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 day, got %d", len(results))
	}
	// Watch 300 (beats iPhone 500) + Watch 200 = 500.00
	if results[0]["total"] != "500.00" {
		t.Errorf("expected 500.00 (Watch wins dedup), got %v", results[0]["total"])
	}
}

func TestQueryActiveEnergyDailyTotal_Empty(t *testing.T) {
	db := tempDB(t)
	results, err := db.QueryActiveEnergyDailyTotal(QueryParams{Table: "active-energy"})
	if err != nil {
		t.Fatalf("unexpected error on empty table: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestQueryRows_BloodPressureColumns(t *testing.T) {
	db := tempDB(t)
	cols := []string{"source_name", "start_date", "end_date", "systolic", "diastolic", "unit"}
	records := [][]any{
		{"Watch", "2024-01-01 08:00:00", "2024-01-01 08:01:00", 120.0, 80.0, "mmHg"},
	}
	db.BatchInsertRecords("blood_pressure", cols, records)

	rows, err := db.QueryRows(QueryParams{Table: "blood-pressure", Limit: 10})
	if err != nil {
		t.Fatalf("QueryRows blood-pressure: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	row := rows[0]
	if _, ok := row["systolic"]; !ok {
		t.Error("expected systolic column in result")
	}
	if _, ok := row["diastolic"]; !ok {
		t.Error("expected diastolic column in result")
	}
	// Verify values are numeric
	systolic, ok := row["systolic"].(float64)
	if !ok {
		t.Errorf("expected systolic to be float64, got %T", row["systolic"])
	} else if systolic != 120.0 {
		t.Errorf("expected systolic=120, got %v", systolic)
	}
	diastolic, ok := row["diastolic"].(float64)
	if !ok {
		t.Errorf("expected diastolic to be float64, got %T", row["diastolic"])
	} else if diastolic != 80.0 {
		t.Errorf("expected diastolic=80, got %v", diastolic)
	}
}

func TestQueryRows_AllValidTablesAccessible(t *testing.T) {
	db := tempDB(t)
	for _, name := range ValidTableNames() {
		_, err := db.QueryRows(QueryParams{Table: name, Limit: 1})
		if err != nil {
			t.Errorf("table %q returned error: %v", name, err)
		}
	}
}
