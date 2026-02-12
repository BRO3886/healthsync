package parser

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BRO3886/healthsync/internal/storage"
)

func tempDB(t *testing.T) *storage.DB {
	t.Helper()
	dir := t.TempDir()
	db, err := storage.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("opening test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// --- RecordColumns ---

func TestRecordColumns_Sleep(t *testing.T) {
	cols := RecordColumns("sleep")
	for _, c := range cols {
		if c == "unit" {
			t.Error("sleep table should not have unit column")
		}
	}
	if len(cols) != 4 {
		t.Errorf("expected 4 columns for sleep, got %d", len(cols))
	}
}

func TestRecordColumns_NonSleep(t *testing.T) {
	for _, table := range []string{"heart_rate", "steps", "spo2", "vo2_max"} {
		cols := RecordColumns(table)
		hasUnit := false
		for _, c := range cols {
			if c == "unit" {
				hasUnit = true
			}
		}
		if !hasUnit {
			t.Errorf("table %s should have unit column", table)
		}
		if len(cols) != 5 {
			t.Errorf("expected 5 columns for %s, got %d", table, len(cols))
		}
	}
}

// --- WorkoutColumns ---

func TestWorkoutColumns(t *testing.T) {
	cols := WorkoutColumns()
	if len(cols) != 10 {
		t.Errorf("expected 10 workout columns, got %d", len(cols))
	}
	expected := []string{
		"activity_type", "source_name", "start_date", "end_date",
		"duration", "duration_unit",
		"total_distance", "total_distance_unit",
		"total_energy_burned", "total_energy_burned_unit",
	}
	for i, e := range expected {
		if cols[i] != e {
			t.Errorf("column %d: expected %s, got %s", i, e, cols[i])
		}
	}
}

// --- TargetRecordTypes ---

func TestTargetRecordTypes(t *testing.T) {
	expected := map[string]string{
		"HKQuantityTypeIdentifierHeartRate":        "heart_rate",
		"HKQuantityTypeIdentifierStepCount":        "steps",
		"HKQuantityTypeIdentifierOxygenSaturation": "spo2",
		"HKQuantityTypeIdentifierVO2Max":           "vo2_max",
		"HKCategoryTypeIdentifierSleepAnalysis":    "sleep",
	}
	for k, v := range expected {
		if got, ok := TargetRecordTypes[k]; !ok {
			t.Errorf("missing key %s", k)
		} else if got != v {
			t.Errorf("key %s: expected %s, got %s", k, v, got)
		}
	}
}

// --- parseFloat ---

func TestParseFloat_Empty(t *testing.T) {
	if v := parseFloat(""); v != nil {
		t.Errorf("expected nil for empty, got %v", v)
	}
}

func TestParseFloat_Valid(t *testing.T) {
	v := parseFloat("72.5")
	if v == nil {
		t.Fatal("expected non-nil for valid float")
	}
	if f, ok := v.(float64); !ok || f != 72.5 {
		t.Errorf("expected 72.5, got %v", v)
	}
}

func TestParseFloat_Integer(t *testing.T) {
	v := parseFloat("100")
	if f, ok := v.(float64); !ok || f != 100.0 {
		t.Errorf("expected 100.0, got %v", v)
	}
}

func TestParseFloat_Invalid(t *testing.T) {
	if v := parseFloat("not-a-number"); v != nil {
		t.Errorf("expected nil for invalid, got %v", v)
	}
}

func TestParseFloat_Negative(t *testing.T) {
	v := parseFloat("-3.14")
	if f, ok := v.(float64); !ok || f != -3.14 {
		t.Errorf("expected -3.14, got %v", v)
	}
}

// --- nilIfEmpty ---

func TestNilIfEmpty_Empty(t *testing.T) {
	if v := nilIfEmpty(""); v != nil {
		t.Errorf("expected nil for empty, got %v", v)
	}
}

func TestNilIfEmpty_NonEmpty(t *testing.T) {
	v := nilIfEmpty("hello")
	if s, ok := v.(string); !ok || s != "hello" {
		t.Errorf("expected 'hello', got %v", v)
	}
}

// --- stripDTD ---

func TestStripDTD_WithDTD(t *testing.T) {
	input := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE HealthData [
<!ELEMENT HealthData (Record*)>
<!ATTLIST Record type CDATA #REQUIRED>
]>
<HealthData locale="en_US">
<Record type="test"/>
</HealthData>`

	cleaned := stripDTD(strings.NewReader(input))
	out, err := io.ReadAll(cleaned)
	if err != nil {
		t.Fatalf("reading stripped: %v", err)
	}

	s := string(out)
	if strings.Contains(s, "<!DOCTYPE") {
		t.Error("DTD was not stripped")
	}
	if !strings.Contains(s, "<HealthData") {
		t.Error("HealthData element was stripped")
	}
	if !strings.Contains(s, "<Record") {
		t.Error("Record element was stripped")
	}
	if !strings.Contains(s, `<?xml`) {
		t.Error("XML declaration was stripped")
	}
}

func TestStripDTD_NoDTD(t *testing.T) {
	input := `<?xml version="1.0"?>
<HealthData>
<Record type="test"/>
</HealthData>`

	cleaned := stripDTD(strings.NewReader(input))
	out, err := io.ReadAll(cleaned)
	if err != nil {
		t.Fatalf("reading stripped: %v", err)
	}

	if !strings.Contains(string(out), "<Record") {
		t.Error("Record element missing from output")
	}
}

func TestStripDTD_EmptyInput(t *testing.T) {
	cleaned := stripDTD(strings.NewReader(""))
	out, err := io.ReadAll(cleaned)
	if err != nil {
		t.Fatalf("reading stripped: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("expected empty output, got %q", string(out))
	}
}

// --- ParseFile ---

func TestParseFile_UnsupportedExtension(t *testing.T) {
	f := filepath.Join(t.TempDir(), "data.csv")
	os.WriteFile(f, []byte("data"), 0644)

	db := tempDB(t)
	_, err := ParseFile(f, db, nil)
	if err == nil {
		t.Fatal("expected error for .csv file")
	}
	if !strings.Contains(err.Error(), "unsupported file type") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestParseFile_FileNotFound(t *testing.T) {
	db := tempDB(t)
	_, err := ParseFile("/nonexistent/file.xml", db, nil)
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func makeTestXML(records string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<HealthData locale="en_US">
` + records + `
</HealthData>`
}

func writeTestXML(t *testing.T, content string) string {
	t.Helper()
	f := filepath.Join(t.TempDir(), "export.xml")
	if err := os.WriteFile(f, []byte(content), 0644); err != nil {
		t.Fatalf("writing test xml: %v", err)
	}
	return f
}

func TestParseFile_AllRecordTypes(t *testing.T) {
	xml := makeTestXML(`
  <Record type="HKQuantityTypeIdentifierHeartRate" sourceName="Watch" unit="count/min" value="72" startDate="2024-01-01 00:00:00 +0000" endDate="2024-01-01 00:01:00 +0000"/>
  <Record type="HKQuantityTypeIdentifierStepCount" sourceName="Watch" unit="count" value="100" startDate="2024-01-01 00:00:00 +0000" endDate="2024-01-01 00:01:00 +0000"/>
  <Record type="HKQuantityTypeIdentifierOxygenSaturation" sourceName="Watch" unit="%" value="0.98" startDate="2024-01-01 00:00:00 +0000" endDate="2024-01-01 00:01:00 +0000"/>
  <Record type="HKQuantityTypeIdentifierVO2Max" sourceName="Watch" unit="mL/min·kg" value="42" startDate="2024-01-01 00:00:00 +0000" endDate="2024-01-01 00:01:00 +0000"/>
  <Record type="HKCategoryTypeIdentifierSleepAnalysis" sourceName="Watch" value="HKCategoryValueSleepAnalysisAsleepCore" startDate="2024-01-01 22:00:00 +0000" endDate="2024-01-02 06:00:00 +0000"/>
`)
	f := writeTestXML(t, xml)
	db := tempDB(t)

	result, err := ParseFile(f, db, nil)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if result.Total != 5 {
		t.Errorf("expected 5 records, got %d", result.Total)
	}

	// Verify each table has 1 row
	tables := map[string]int64{
		"heart-rate": 1, "steps": 1, "spo2": 1, "vo2max": 1, "sleep": 1,
	}
	for table, expected := range tables {
		count, err := db.CountRows(table)
		if err != nil {
			t.Errorf("counting %s: %v", table, err)
			continue
		}
		if count != expected {
			t.Errorf("table %s: expected %d, got %d", table, expected, count)
		}
	}
}

func TestParseFile_SkipsIrrelevantRecords(t *testing.T) {
	xml := makeTestXML(`
  <Record type="HKQuantityTypeIdentifierDietaryWater" sourceName="App" unit="mL" value="200" startDate="2024-01-01 00:00:00 +0000" endDate="2024-01-01 00:01:00 +0000"/>
  <Record type="HKQuantityTypeIdentifierBodyMass" sourceName="App" unit="kg" value="70" startDate="2024-01-01 00:00:00 +0000" endDate="2024-01-01 00:01:00 +0000"/>
  <Record type="HKQuantityTypeIdentifierHeartRate" sourceName="Watch" unit="count/min" value="72" startDate="2024-01-01 00:00:00 +0000" endDate="2024-01-01 00:01:00 +0000"/>
`)
	f := writeTestXML(t, xml)
	db := tempDB(t)

	result, err := ParseFile(f, db, nil)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("expected 1 matching record, got %d", result.Total)
	}
}

func TestParseFile_ZeroMatchingRecords(t *testing.T) {
	xml := makeTestXML(`
  <Record type="HKQuantityTypeIdentifierDietaryWater" sourceName="App" unit="mL" value="200" startDate="2024-01-01 00:00:00 +0000" endDate="2024-01-01 00:01:00 +0000"/>
`)
	f := writeTestXML(t, xml)
	db := tempDB(t)

	result, err := ParseFile(f, db, nil)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if result.Total != 0 {
		t.Errorf("expected 0 records, got %d", result.Total)
	}
}

func TestParseFile_Workouts(t *testing.T) {
	xml := makeTestXML(`
  <Workout workoutActivityType="HKWorkoutActivityTypeRunning" duration="30.5" durationUnit="min" totalDistance="5.2" totalDistanceUnit="km" totalEnergyBurned="300" totalEnergyBurnedUnit="kcal" sourceName="Watch" startDate="2024-01-01 08:00:00 +0000" endDate="2024-01-01 08:30:00 +0000"/>
`)
	f := writeTestXML(t, xml)
	db := tempDB(t)

	result, err := ParseFile(f, db, nil)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if result.Workouts != 1 {
		t.Errorf("expected 1 workout, got %d", result.Workouts)
	}

	count, _ := db.CountRows("workouts")
	if count != 1 {
		t.Errorf("expected 1 workout in db, got %d", count)
	}
}

func TestParseFile_WorkoutMissingOptionalFields(t *testing.T) {
	xml := makeTestXML(`
  <Workout workoutActivityType="HKWorkoutActivityTypeYoga" duration="60" durationUnit="min" sourceName="Watch" startDate="2024-01-01 08:00:00 +0000" endDate="2024-01-01 09:00:00 +0000"/>
`)
	f := writeTestXML(t, xml)
	db := tempDB(t)

	result, err := ParseFile(f, db, nil)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if result.Workouts != 1 {
		t.Errorf("expected 1 workout, got %d", result.Workouts)
	}
}

func TestParseFile_ProgressCallback(t *testing.T) {
	// Create XML with >10k records to trigger progress
	var b strings.Builder
	for i := 0; i < 10001; i++ {
		b.WriteString(`  <Record type="HKQuantityTypeIdentifierHeartRate" sourceName="Watch" unit="count/min" value="72" startDate="2024-01-01 00:00:00 +0000" endDate="2024-01-01 00:01:00 +0000"/>` + "\n")
	}
	xml := makeTestXML(b.String())
	f := writeTestXML(t, xml)
	db := tempDB(t)

	called := false
	progress := func(records int64, workouts int64) {
		called = true
		if records < 0 {
			t.Error("negative record count in progress")
		}
	}

	_, err := ParseFile(f, db, progress)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if !called {
		t.Error("progress callback was never called")
	}
}

func TestParseFile_NilProgressCallback(t *testing.T) {
	xml := makeTestXML(`
  <Record type="HKQuantityTypeIdentifierHeartRate" sourceName="Watch" unit="count/min" value="72" startDate="2024-01-01 00:00:00 +0000" endDate="2024-01-01 00:01:00 +0000"/>
`)
	f := writeTestXML(t, xml)
	db := tempDB(t)

	// Should not panic with nil progress
	_, err := ParseFile(f, db, nil)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
}

func TestParseFile_XMLWithDTD(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE HealthData [
<!ELEMENT HealthData (Record*)>
<!ATTLIST Record type CDATA #REQUIRED>
]>
<HealthData locale="en_US">
  <Record type="HKQuantityTypeIdentifierHeartRate" sourceName="Watch" unit="count/min" value="72" startDate="2024-01-01 00:00:00 +0000" endDate="2024-01-01 00:01:00 +0000"/>
</HealthData>`

	f := writeTestXML(t, xml)
	db := tempDB(t)

	result, err := ParseFile(f, db, nil)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("expected 1 record from DTD XML, got %d", result.Total)
	}
}

// --- Zip support ---

func makeTestZip(t *testing.T, xmlContent string, xmlPath string) string {
	t.Helper()
	zipPath := filepath.Join(t.TempDir(), "export.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("creating zip: %v", err)
	}
	w := zip.NewWriter(f)
	zf, err := w.Create(xmlPath)
	if err != nil {
		t.Fatalf("creating zip entry: %v", err)
	}
	if _, err := io.Copy(zf, bytes.NewReader([]byte(xmlContent))); err != nil {
		t.Fatalf("writing zip entry: %v", err)
	}
	w.Close()
	f.Close()
	return zipPath
}

func TestParseFile_ZipWithExportXML(t *testing.T) {
	xml := makeTestXML(`
  <Record type="HKQuantityTypeIdentifierHeartRate" sourceName="Watch" unit="count/min" value="72" startDate="2024-01-01 00:00:00 +0000" endDate="2024-01-01 00:01:00 +0000"/>
`)
	zipPath := makeTestZip(t, xml, "export.xml")
	db := tempDB(t)

	result, err := ParseFile(zipPath, db, nil)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("expected 1 record from zip, got %d", result.Total)
	}
}

func TestParseFile_ZipWithNestedExportXML(t *testing.T) {
	xml := makeTestXML(`
  <Record type="HKQuantityTypeIdentifierStepCount" sourceName="iPhone" unit="count" value="500" startDate="2024-01-01 00:00:00 +0000" endDate="2024-01-01 00:01:00 +0000"/>
`)
	zipPath := makeTestZip(t, xml, "apple_health_export/export.xml")
	db := tempDB(t)

	result, err := ParseFile(zipPath, db, nil)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("expected 1 record from nested zip, got %d", result.Total)
	}
}

func TestParseFile_ZipWithoutExportXML(t *testing.T) {
	zipPath := makeTestZip(t, "some data", "other_file.txt")
	db := tempDB(t)

	_, err := ParseFile(zipPath, db, nil)
	if err == nil {
		t.Fatal("expected error for zip without export.xml")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseFile_InvalidZip(t *testing.T) {
	f := filepath.Join(t.TempDir(), "bad.zip")
	os.WriteFile(f, []byte("not a zip"), 0644)
	db := tempDB(t)

	_, err := ParseFile(f, db, nil)
	if err == nil {
		t.Fatal("expected error for invalid zip")
	}
}

// --- Records with child elements ---

func TestParseFile_RecordWithMetadata(t *testing.T) {
	xml := makeTestXML(`
  <Record type="HKQuantityTypeIdentifierHeartRate" sourceName="Watch" unit="count/min" value="72" startDate="2024-01-01 00:00:00 +0000" endDate="2024-01-01 00:01:00 +0000">
    <MetadataEntry key="HKMetadataKeySyncVersion" value="1"/>
    <MetadataEntry key="HKMetadataKeySyncIdentifier" value="ABC-123"/>
  </Record>
`)
	f := writeTestXML(t, xml)
	db := tempDB(t)

	result, err := ParseFile(f, db, nil)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("expected 1 record, got %d", result.Total)
	}
}

func TestParseFile_WorkoutWithChildElements(t *testing.T) {
	xml := makeTestXML(`
  <Workout workoutActivityType="HKWorkoutActivityTypeRunning" duration="30" durationUnit="min" totalDistance="5" totalDistanceUnit="km" totalEnergyBurned="300" totalEnergyBurnedUnit="kcal" sourceName="Watch" startDate="2024-01-01 08:00:00 +0000" endDate="2024-01-01 08:30:00 +0000">
    <MetadataEntry key="HKIndoorWorkout" value="0"/>
    <WorkoutEvent type="HKWorkoutEventTypePause" date="2024-01-01 08:15:00 +0000"/>
    <WorkoutStatistics type="HKQuantityTypeIdentifierHeartRate" startDate="2024-01-01 08:00:00 +0000" endDate="2024-01-01 08:30:00 +0000" average="150" minimum="120" maximum="180" unit="count/min"/>
  </Workout>
`)
	f := writeTestXML(t, xml)
	db := tempDB(t)

	result, err := ParseFile(f, db, nil)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if result.Workouts != 1 {
		t.Errorf("expected 1 workout, got %d", result.Workouts)
	}
}

// --- Mixed records and workouts ---

func TestParseFile_MixedRecordsAndWorkouts(t *testing.T) {
	xml := makeTestXML(`
  <Record type="HKQuantityTypeIdentifierHeartRate" sourceName="Watch" unit="count/min" value="72" startDate="2024-01-01 00:00:00 +0000" endDate="2024-01-01 00:01:00 +0000"/>
  <Record type="HKQuantityTypeIdentifierStepCount" sourceName="Watch" unit="count" value="100" startDate="2024-01-01 00:00:00 +0000" endDate="2024-01-01 00:01:00 +0000"/>
  <Workout workoutActivityType="HKWorkoutActivityTypeRunning" duration="30" durationUnit="min" sourceName="Watch" startDate="2024-01-01 08:00:00 +0000" endDate="2024-01-01 08:30:00 +0000"/>
  <Record type="HKQuantityTypeIdentifierHeartRate" sourceName="Watch" unit="count/min" value="75" startDate="2024-01-01 01:00:00 +0000" endDate="2024-01-01 01:01:00 +0000"/>
  <Workout workoutActivityType="HKWorkoutActivityTypeYoga" duration="60" durationUnit="min" sourceName="Watch" startDate="2024-01-02 08:00:00 +0000" endDate="2024-01-02 09:00:00 +0000"/>
`)
	f := writeTestXML(t, xml)
	db := tempDB(t)

	result, err := ParseFile(f, db, nil)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if result.Total != 3 {
		t.Errorf("expected 3 records, got %d", result.Total)
	}
	if result.Workouts != 2 {
		t.Errorf("expected 2 workouts, got %d", result.Workouts)
	}
}
