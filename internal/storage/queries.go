package storage

import (
	"database/sql"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// TableInfo holds per-table stats for db info output.
type TableInfo struct {
	Name     string
	RowCount int64
	LastDate string // MAX(start_date)
}

// DBInfo holds overall database statistics.
type DBInfo struct {
	Path       string
	FileSizeMB float64
	Tables     []TableInfo // only non-empty tables, sorted by row count desc
	TotalRows  int64
}

// GetDBInfo returns statistics about the database at dbPath.
func (db *DB) GetDBInfo(dbPath string) (*DBInfo, error) {
	info := &DBInfo{Path: dbPath}

	// File size
	if fi, err := os.Stat(dbPath); err == nil {
		info.FileSizeMB = float64(fi.Size()) / (1024 * 1024)
	}

	// List all user tables
	rows, err := db.conn.Query(`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("listing tables: %w", err)
	}
	defer rows.Close()

	var tableNames []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tableNames = append(tableNames, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, name := range tableNames {
		var count int64
		var lastDate sql.NullString

		err := db.conn.QueryRow(
			fmt.Sprintf(`SELECT COUNT(*), MAX(start_date) FROM %s`, name),
		).Scan(&count, &lastDate)
		if err != nil {
			// Table may not have start_date (shouldn't happen but be safe)
			continue
		}

		if count == 0 {
			continue
		}

		t := TableInfo{
			Name:     name,
			RowCount: count,
		}
		if lastDate.Valid {
			// Trim to YYYY-MM-DD
			if len(lastDate.String) >= 10 {
				t.LastDate = lastDate.String[:10]
			} else {
				t.LastDate = lastDate.String
			}
		}

		info.Tables = append(info.Tables, t)
		info.TotalRows += count
	}

	// Sort by row count descending
	sort.Slice(info.Tables, func(i, j int) bool {
		return info.Tables[i].RowCount > info.Tables[j].RowCount
	})

	return info, nil
}

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
	// Existing
	"heart-rate": "heart_rate",
	"heart_rate": "heart_rate",
	"steps":      "steps",
	"spo2":       "spo2",
	"vo2max":     "vo2_max",
	"vo2_max":    "vo2_max",
	"sleep":      "sleep",
	"workouts":   "workouts",

	// Cardiac vitals
	"resting-heart-rate":  "resting_heart_rate",
	"resting_heart_rate":  "resting_heart_rate",
	"hrv":                 "hrv",
	"heart-rate-recovery": "heart_rate_recovery",
	"heart_rate_recovery": "heart_rate_recovery",
	"respiratory-rate":    "respiratory_rate",
	"respiratory_rate":    "respiratory_rate",
	"blood-pressure":      "blood_pressure",
	"blood_pressure":      "blood_pressure",

	// Activity / Energy
	"active-energy":   "active_energy",
	"active_energy":   "active_energy",
	"basal-energy":    "basal_energy",
	"basal_energy":    "basal_energy",
	"exercise-time":   "exercise_time",
	"exercise_time":   "exercise_time",
	"stand-time":      "stand_time",
	"stand_time":      "stand_time",
	"flights-climbed": "flights_climbed",
	"flights_climbed": "flights_climbed",

	// Distance
	"distance-walking-running": "distance_walking_running",
	"distance_walking_running": "distance_walking_running",
	"distance-cycling":         "distance_cycling",
	"distance_cycling":         "distance_cycling",

	// Body composition
	"body-mass":       "body_mass",
	"body_mass":       "body_mass",
	"bmi":             "body_mass_index",
	"body-mass-index": "body_mass_index",
	"body_mass_index": "body_mass_index",
	"height":          "height",

	// Mobility / Walking
	"walking-speed":          "walking_speed",
	"walking_speed":          "walking_speed",
	"walking-step-length":    "walking_step_length",
	"walking_step_length":    "walking_step_length",
	"walking-asymmetry":      "walking_asymmetry",
	"walking_asymmetry":      "walking_asymmetry",
	"walking-double-support": "walking_double_support",
	"walking_double_support": "walking_double_support",
	"walking-steadiness":     "walking_steadiness",
	"walking_steadiness":     "walking_steadiness",
	"stair-ascent-speed":     "stair_ascent_speed",
	"stair_ascent_speed":     "stair_ascent_speed",
	"stair-descent-speed":    "stair_descent_speed",
	"stair_descent_speed":    "stair_descent_speed",
	"six-minute-walk":        "six_minute_walk",
	"six_minute_walk":        "six_minute_walk",

	// Running metrics
	"running-speed":                "running_speed",
	"running_speed":                "running_speed",
	"running-power":                "running_power",
	"running_power":                "running_power",
	"running-stride-length":        "running_stride_length",
	"running_stride_length":        "running_stride_length",
	"running-ground-contact-time":  "running_ground_contact_time",
	"running_ground_contact_time":  "running_ground_contact_time",
	"running-vertical-oscillation": "running_vertical_oscillation",
	"running_vertical_oscillation": "running_vertical_oscillation",

	// Other quantity types
	"wrist-temperature":  "wrist_temperature",
	"wrist_temperature":  "wrist_temperature",
	"time-in-daylight":   "time_in_daylight",
	"time_in_daylight":   "time_in_daylight",
	"dietary-water":      "dietary_water",
	"dietary_water":      "dietary_water",
	"physical-effort":    "physical_effort",
	"physical_effort":    "physical_effort",
	"walking-heart-rate": "walking_heart_rate",
	"walking_heart_rate": "walking_heart_rate",

	// Category types
	"mindful-sessions": "mindful_sessions",
	"mindful_sessions": "mindful_sessions",
	"stand-hours":      "stand_hours",
	"stand_hours":      "stand_hours",
}

// ValidTableNames returns the list of valid CLI table names.
func ValidTableNames() []string {
	return []string{
		// Existing
		"heart-rate", "steps", "spo2", "vo2max", "sleep", "workouts",
		// Cardiac vitals
		"resting-heart-rate", "hrv", "heart-rate-recovery", "respiratory-rate", "blood-pressure",
		// Activity / Energy
		"active-energy", "basal-energy", "exercise-time", "stand-time", "flights-climbed",
		// Distance
		"distance-walking-running", "distance-cycling",
		// Body composition
		"body-mass", "bmi", "height",
		// Mobility / Walking
		"walking-speed", "walking-step-length", "walking-asymmetry",
		"walking-double-support", "walking-steadiness",
		"stair-ascent-speed", "stair-descent-speed", "six-minute-walk",
		// Running metrics
		"running-speed", "running-power", "running-stride-length",
		"running-ground-contact-time", "running-vertical-oscillation",
		// Other quantity types
		"wrist-temperature", "time-in-daylight", "dietary-water", "physical-effort", "walking-heart-rate",
		// Category types
		"mindful-sessions", "stand-hours",
	}
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

// queryEnergyDailyTotal is the shared implementation for active_energy and basal_energy --total.
// It uses the same deduplication algorithm as steps (overlap detection by source priority).
func (db *DB) queryEnergyDailyTotal(tableName string, params QueryParams) ([]map[string]interface{}, error) {
	query := fmt.Sprintf("SELECT source_name, start_date, end_date, value, unit FROM %s WHERE 1=1", tableName)
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
		return nil, fmt.Errorf("querying %s for dedup: %w", tableName, err)
	}
	defer rows.Close()

	var records []stepsRecord
	for rows.Next() {
		var r stepsRecord
		if err := rows.Scan(&r.source, &r.startDate, &r.endDate, &r.value, &r.unit); err != nil {
			return nil, fmt.Errorf("scanning %s row: %w", tableName, err)
		}
		records = append(records, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	deduped := deduplicateSteps(records)

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
			"total": strconv.FormatFloat(dailyTotals[day], 'f', 2, 64),
		})
	}
	return results, nil
}

const (
	// sleepTimeLayout is the normalized form the parser writes to the sleep table.
	sleepTimeLayout = "2006-01-02 15:04:05"

	// sleepSessionGap is the idle time that separates two distinct sleep sessions.
	// Awake segments inside a single night run to minutes; the gap between waking
	// and a daytime nap runs to hours.
	sleepSessionGap = 2 * time.Hour

	// napOnsetStart and napOnsetEnd bound the hours in which an onset is treated as
	// a nap rather than the start of a night. Onset time, not duration, decides this:
	// a 2-hour night before an early flight is still a night, and a 3-hour Sunday
	// afternoon collapse is still a nap.
	napOnsetStart = 11
	napOnsetEnd   = 20

	// nightShift maps an onset to the night it belongs to. Any onset from noon
	// onwards keeps its own date; any onset before noon rolls back to the previous
	// date. This holds across the full plausible range of human bedtimes.
	nightShift = 12 * time.Hour
)

// sleepSession is one contiguous run of Asleep segments, i.e. a real night or a
// real nap, reconstructed from the gaps in the data rather than from the clock.
type sleepSession struct {
	start  time.Time
	end    time.Time
	asleep time.Duration
}

// isNap reports whether the session began during the hours in which a person is
// awake, which makes it a nap rather than a night.
func (s sleepSession) isNap() bool {
	h := s.start.Hour()
	return h >= napOnsetStart && h < napOnsetEnd
}

// night returns the date of the night this session belongs to. A nap is attributed
// to the night that preceded it, so a night and the naps that follow it during the
// same waking day land on one row.
func (s sleepSession) night() string {
	if s.isNap() {
		return s.start.AddDate(0, 0, -1).Format(time.DateOnly)
	}
	return s.start.Add(-nightShift).Format(time.DateOnly)
}

// buildSleepSessions clusters time-ordered Asleep segments into sessions, breaking
// wherever the data goes quiet for longer than sleepSessionGap. Overlapping segments
// are merged before their durations are summed, so a stage recorded twice by two
// sources is counted once.
func buildSleepSessions(segments []sleepSession) []sleepSession {
	var sessions []sleepSession
	var cur []sleepSession

	flush := func() {
		if len(cur) == 0 {
			return
		}
		merged := []sleepSession{cur[0]}
		for _, seg := range cur[1:] {
			last := &merged[len(merged)-1]
			if !seg.start.After(last.end) {
				if seg.end.After(last.end) {
					last.end = seg.end
				}
				continue
			}
			merged = append(merged, seg)
		}

		var asleep time.Duration
		for _, m := range merged {
			asleep += m.end.Sub(m.start)
		}
		sessions = append(sessions, sleepSession{
			start:  cur[0].start,
			end:    cur[len(cur)-1].end,
			asleep: asleep,
		})
		cur = nil
	}

	for _, seg := range segments {
		if len(cur) > 0 && seg.start.Sub(cur[len(cur)-1].end) > sleepSessionGap {
			flush()
		}
		cur = append(cur, seg)
	}
	flush()

	return sessions
}

// QuerySleepDailyTotal returns nightly sleep totals, one row per night that actually
// has data. Only Asleep stages count (Core, Deep, REM, Unspecified); InBed and Awake
// are excluded.
//
// Segments are clustered into whole sessions before any date is assigned to them, so
// a single night is never cut in half by a clock boundary. Nights with no recorded
// sleep are omitted entirely rather than reported as zero: "did not sleep" and "did
// not wear the watch" are different facts. Naps are reported in their own column and
// are never folded into the night's hours.
func (db *DB) QuerySleepDailyTotal(params QueryParams) ([]map[string]interface{}, error) {
	// A night keyed to date D can hold segments recorded on D (evening onset) or on
	// D+1 (post-midnight onset, and any nap during the following day), so the scan
	// window is widened past To and the results are filtered by night afterwards.
	query := `SELECT start_date, end_date FROM sleep WHERE value LIKE '%Asleep%'`
	var args []interface{}

	if params.From != "" {
		query += " AND start_date >= ?"
		args = append(args, params.From)
	}
	if params.To != "" {
		query += " AND date(start_date) <= date(?, '+2 days')"
		args = append(args, params.To)
	}
	query += " ORDER BY start_date ASC"

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying sleep daily total: %w", err)
	}
	defer rows.Close()

	var segments []sleepSession
	for rows.Next() {
		var startStr, endStr string
		if err := rows.Scan(&startStr, &endStr); err != nil {
			return nil, fmt.Errorf("scanning sleep row: %w", err)
		}
		start, err := time.Parse(sleepTimeLayout, startStr)
		if err != nil {
			continue
		}
		end, err := time.Parse(sleepTimeLayout, endStr)
		if err != nil || !end.After(start) {
			continue
		}
		segments = append(segments, sleepSession{start: start, end: end})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	type nightTotal struct {
		hours float64
		naps  float64
		onset time.Time
		wake  time.Time
	}
	totals := make(map[string]*nightTotal)

	for _, s := range buildSleepSessions(segments) {
		night := s.night()
		if params.From != "" && night < params.From[:min(len(params.From), 10)] {
			continue
		}
		if params.To != "" && night > params.To[:min(len(params.To), 10)] {
			continue
		}

		t, ok := totals[night]
		if !ok {
			t = &nightTotal{}
			totals[night] = t
		}
		if s.isNap() {
			t.naps += s.asleep.Hours()
			continue
		}
		t.hours += s.asleep.Hours()
		if t.onset.IsZero() || s.start.Before(t.onset) {
			t.onset = s.start
		}
		if s.end.After(t.wake) {
			t.wake = s.end
		}
	}

	nights := make([]string, 0, len(totals))
	for night := range totals {
		nights = append(nights, night)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(nights)))

	if params.Limit > 0 && len(nights) > params.Limit {
		nights = nights[:params.Limit]
	}

	results := make([]map[string]interface{}, 0, len(nights))
	for _, night := range nights {
		t := totals[night]
		row := map[string]interface{}{
			"night": night,
			"hours": strconv.FormatFloat(t.hours, 'f', 1, 64),
			"naps":  strconv.FormatFloat(t.naps, 'f', 1, 64),
			"onset": "",
			"wake":  "",
		}
		if !t.onset.IsZero() {
			row["onset"] = t.onset.Format("15:04")
			row["wake"] = t.wake.Format("15:04")
		}
		results = append(results, row)
	}
	return results, nil
}

// QueryActiveEnergyDailyTotal returns deduplicated daily active energy totals aggregated by calendar day.
func (db *DB) QueryActiveEnergyDailyTotal(params QueryParams) ([]map[string]interface{}, error) {
	return db.queryEnergyDailyTotal("active_energy", params)
}

// QueryBasalEnergyDailyTotal returns deduplicated daily basal energy totals aggregated by calendar day.
func (db *DB) QueryBasalEnergyDailyTotal(params QueryParams) ([]map[string]interface{}, error) {
	return db.queryEnergyDailyTotal("basal_energy", params)
}
