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
		// Original 5
		"HKQuantityTypeIdentifierHeartRate":        "heart_rate",
		"HKQuantityTypeIdentifierStepCount":        "steps",
		"HKQuantityTypeIdentifierOxygenSaturation": "spo2",
		"HKQuantityTypeIdentifierVO2Max":           "vo2_max",
		"HKCategoryTypeIdentifierSleepAnalysis":    "sleep",
		// Spot-check new entries
		"HKQuantityTypeIdentifierRestingHeartRate":        "resting_heart_rate",
		"HKQuantityTypeIdentifierActiveEnergyBurned":      "active_energy",
		"HKQuantityTypeIdentifierBodyMass":                "body_mass",
		"HKCategoryTypeIdentifierMindfulSession":          "mindful_sessions",
		"HKQuantityTypeIdentifierBloodPressureSystolic":   "blood_pressure_systolic",
		"HKQuantityTypeIdentifierBloodPressureDiastolic":  "blood_pressure_diastolic",
	}
	for k, v := range expected {
		if got, ok := TargetRecordTypes[k]; !ok {
			t.Errorf("missing key %s", k)
		} else if got != v {
			t.Errorf("key %s: expected %s, got %s", k, v, got)
		}
	}
	if len(TargetRecordTypes) < 39 {
		t.Errorf("expected at least 39 entries, got %d", len(TargetRecordTypes))
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
	// DietaryCaffeine and UVExposure are not in TargetRecordTypes — should be skipped
	xml := makeTestXML(`
  <Record type="HKQuantityTypeIdentifierDietaryCaffeine" sourceName="App" unit="mg" value="80" startDate="2024-01-01 00:00:00 +0000" endDate="2024-01-01 00:01:00 +0000"/>
  <Record type="HKQuantityTypeIdentifierUVExposure" sourceName="App" unit="count" value="3" startDate="2024-01-01 00:00:00 +0000" endDate="2024-01-01 00:01:00 +0000"/>
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
	// DietaryCaffeine is not in TargetRecordTypes — should produce 0 records
	xml := makeTestXML(`
  <Record type="HKQuantityTypeIdentifierDietaryCaffeine" sourceName="App" unit="mg" value="80" startDate="2024-01-01 00:00:00 +0000" endDate="2024-01-01 00:01:00 +0000"/>
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
	return makeTestZipEntries(t, map[string]string{xmlPath: xmlContent})
}

func makeTestZipEntries(t *testing.T, entries map[string]string) string {
	t.Helper()
	zipPath := filepath.Join(t.TempDir(), "export.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("creating zip: %v", err)
	}
	w := zip.NewWriter(f)
	for path, content := range entries {
		zf, err := w.Create(path)
		if err != nil {
			t.Fatalf("creating zip entry %q: %v", path, err)
		}
		if _, err := io.Copy(zf, bytes.NewReader([]byte(content))); err != nil {
			t.Fatalf("writing zip entry %q: %v", path, err)
		}
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
	if !strings.Contains(err.Error(), "no .xml entries") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseFile_ZipWithLocalizedExport(t *testing.T) {
	xml := makeTestXML(`
  <Record type="HKQuantityTypeIdentifierStepCount" sourceName="iPhone" unit="count" value="500" startDate="2024-01-01 00:00:00 +0000" endDate="2024-01-01 00:01:00 +0000"/>
`)
	zipPath := makeTestZipEntries(t, map[string]string{
		"apple_health_export/导出.xml":       xml,
		"apple_health_export/export_cda.xml": "<ClinicalDocument></ClinicalDocument>",
	})
	db := tempDB(t)

	result, err := ParseFile(zipPath, db, nil)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("expected 1 record from localized zip, got %d", result.Total)
	}
}

// Stray xml files (e.g. from re-zipped archives) must not be picked — we
// identify the export by content, not by filename.
func TestParseFile_ZipWithStrayXML(t *testing.T) {
	healthXML := makeTestXML(`
  <Record type="HKQuantityTypeIdentifierStepCount" sourceName="iPhone" unit="count" value="100" startDate="2024-01-01 00:00:00 +0000" endDate="2024-01-01 00:01:00 +0000"/>
`)
	zipPath := makeTestZipEntries(t, map[string]string{
		"apple_health_export/notes.xml":      `<?xml version="1.0"?><notes><note>hi</note></notes>`,
		"apple_health_export/导出.xml":         healthXML,
		"apple_health_export/export_cda.xml": `<?xml version="1.0"?><ClinicalDocument/>`,
	})
	db := tempDB(t)

	result, err := ParseFile(zipPath, db, nil)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("expected 1 record (HealthKit xml picked by content), got %d", result.Total)
	}
}

// export_cda.xml must never be picked, even if it is the only .xml present.
func TestParseFile_ZipIgnoresCDAOnly(t *testing.T) {
	zipPath := makeTestZipEntries(t, map[string]string{
		"apple_health_export/export_cda.xml": "<ClinicalDocument></ClinicalDocument>",
	})
	db := tempDB(t)

	_, err := ParseFile(zipPath, db, nil)
	if err == nil {
		t.Fatal("expected error when only export_cda.xml is present")
	}
	if !strings.Contains(err.Error(), "no HealthKit export XML") {
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

func TestRecordColumns_CategoryTables(t *testing.T) {
	for _, tbl := range []string{"mindful_sessions", "stand_hours"} {
		cols := RecordColumns(tbl)
		if len(cols) != 4 {
			t.Errorf("%s: expected 4 columns, got %d", tbl, len(cols))
		}
		for _, c := range cols {
			if c == "unit" {
				t.Errorf("%s: should not have unit column", tbl)
			}
		}
	}
}

func TestBloodPressureColumns(t *testing.T) {
	cols := BloodPressureColumns()
	if len(cols) != 6 {
		t.Errorf("expected 6 blood pressure columns, got %d", len(cols))
	}
	expected := []string{"source_name", "start_date", "end_date", "systolic", "diastolic", "unit"}
	for i, e := range expected {
		if cols[i] != e {
			t.Errorf("column %d: expected %s, got %s", i, e, cols[i])
		}
	}
}

func TestParseFile_NewQuantityTypes(t *testing.T) {
	xmlContent := makeTestXML(`
  <Record type="HKQuantityTypeIdentifierRestingHeartRate" sourceName="Watch" unit="count/min" value="58" startDate="2024-01-01 08:00:00 +0000" endDate="2024-01-01 08:01:00 +0000"/>
  <Record type="HKQuantityTypeIdentifierActiveEnergyBurned" sourceName="Watch" unit="kcal" value="350.5" startDate="2024-01-01 00:00:00 +0000" endDate="2024-01-01 23:59:59 +0000"/>
  <Record type="HKQuantityTypeIdentifierBodyMass" sourceName="Withings" unit="kg" value="72.3" startDate="2024-01-01 07:00:00 +0000" endDate="2024-01-01 07:00:00 +0000"/>
`)
	f := writeTestXML(t, xmlContent)
	db := tempDB(t)

	result, err := ParseFile(f, db, nil)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if result.Total != 3 {
		t.Errorf("expected 3 records, got %d", result.Total)
	}

	tables := map[string]int64{
		"resting-heart-rate": 1,
		"active-energy":      1,
		"body-mass":          1,
	}
	for tbl, expected := range tables {
		count, err := db.CountRows(tbl)
		if err != nil {
			t.Errorf("counting %s: %v", tbl, err)
			continue
		}
		if count != expected {
			t.Errorf("table %s: expected %d rows, got %d", tbl, expected, count)
		}
	}
}

func TestParseFile_CategoryTypes(t *testing.T) {
	xmlContent := makeTestXML(`
  <Record type="HKCategoryTypeIdentifierMindfulSession" sourceName="Headspace" value="HKCategoryValueNotApplicable" startDate="2024-01-01 07:00:00 +0000" endDate="2024-01-01 07:10:00 +0000"/>
  <Record type="HKCategoryTypeIdentifierAppleStandHour" sourceName="Watch" value="HKCategoryValueAppleStandHourStood" startDate="2024-01-01 10:00:00 +0000" endDate="2024-01-01 11:00:00 +0000"/>
`)
	f := writeTestXML(t, xmlContent)
	db := tempDB(t)

	result, err := ParseFile(f, db, nil)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("expected 2 records, got %d", result.Total)
	}

	for _, tbl := range []string{"mindful-sessions", "stand-hours"} {
		count, err := db.CountRows(tbl)
		if err != nil {
			t.Errorf("counting %s: %v", tbl, err)
			continue
		}
		if count != 1 {
			t.Errorf("table %s: expected 1 row, got %d", tbl, count)
		}
	}
}

func TestParseFile_BloodPressurePairing(t *testing.T) {
	// Same source + start_date → should produce 1 blood_pressure row
	xmlContent := makeTestXML(`
  <Record type="HKQuantityTypeIdentifierBloodPressureSystolic" sourceName="Watch" unit="mmHg" value="120" startDate="2024-01-01 08:00:00 +0000" endDate="2024-01-01 08:01:00 +0000"/>
  <Record type="HKQuantityTypeIdentifierBloodPressureDiastolic" sourceName="Watch" unit="mmHg" value="80" startDate="2024-01-01 08:00:00 +0000" endDate="2024-01-01 08:01:00 +0000"/>
`)
	f := writeTestXML(t, xmlContent)
	db := tempDB(t)

	result, err := ParseFile(f, db, nil)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("expected 1 paired record, got %d", result.Total)
	}

	count, err := db.CountRows("blood-pressure")
	if err != nil {
		t.Fatalf("counting blood_pressure: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 blood_pressure row, got %d", count)
	}
}

func TestParseFile_BloodPressureUnpairedDropped(t *testing.T) {
	// Only systolic, no diastolic → should produce 0 rows
	xmlContent := makeTestXML(`
  <Record type="HKQuantityTypeIdentifierBloodPressureSystolic" sourceName="Watch" unit="mmHg" value="120" startDate="2024-01-01 08:00:00 +0000" endDate="2024-01-01 08:01:00 +0000"/>
`)
	f := writeTestXML(t, xmlContent)
	db := tempDB(t)

	result, err := ParseFile(f, db, nil)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if result.Total != 0 {
		t.Errorf("expected 0 complete pairs, got %d", result.Total)
	}

	count, _ := db.CountRows("blood-pressure")
	if count != 0 {
		t.Errorf("expected 0 blood_pressure rows, got %d", count)
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

	// Attributes take priority — energy/distance should be from attributes, not WorkoutStatistics
	rows, err := db.QueryRows(storage.QueryParams{Table: "workouts", Limit: 10})
	if err != nil {
		t.Fatalf("query error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	row := rows[0]
	if row["total_energy_burned"] != float64(300) {
		t.Errorf("expected total_energy_burned=300, got %v", row["total_energy_burned"])
	}
	if row["total_distance"] != float64(5) {
		t.Errorf("expected total_distance=5, got %v", row["total_distance"])
	}
}

func TestParseFile_WorkoutStatistics_PopulatesEnergyAndDistance(t *testing.T) {
	// No totalEnergyBurned / totalDistance attributes — values come from WorkoutStatistics children
	xml := makeTestXML(`
  <Workout workoutActivityType="HKWorkoutActivityTypeCycling" duration="47.4" durationUnit="min" sourceName="Watch" startDate="2024-06-01 09:00:00 +0000" endDate="2024-06-01 09:47:00 +0000">
    <WorkoutStatistics type="HKQuantityTypeIdentifierActiveEnergyBurned" startDate="2024-06-01 09:00:00 +0000" endDate="2024-06-01 09:47:00 +0000" sum="230.44" unit="Cal"/>
    <WorkoutStatistics type="HKQuantityTypeIdentifierDistanceCycling" startDate="2024-06-01 09:00:00 +0000" endDate="2024-06-01 09:47:00 +0000" sum="10.43" unit="km"/>
  </Workout>
`)
	f := writeTestXML(t, xml)
	db := tempDB(t)

	result, err := ParseFile(f, db, nil)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if result.Workouts != 1 {
		t.Fatalf("expected 1 workout, got %d", result.Workouts)
	}

	rows, err := db.QueryRows(storage.QueryParams{Table: "workouts", Limit: 10})
	if err != nil {
		t.Fatalf("query error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	row := rows[0]
	if row["total_energy_burned"] == nil {
		t.Error("expected total_energy_burned to be populated from WorkoutStatistics, got nil")
	}
	if row["total_energy_burned"] != 230.44 {
		t.Errorf("expected total_energy_burned=230.44, got %v", row["total_energy_burned"])
	}
	if row["total_energy_burned_unit"] != "Cal" {
		t.Errorf("expected total_energy_burned_unit=Cal, got %v", row["total_energy_burned_unit"])
	}
	if row["total_distance"] == nil {
		t.Error("expected total_distance to be populated from WorkoutStatistics, got nil")
	}
	if row["total_distance"] != 10.43 {
		t.Errorf("expected total_distance=10.43, got %v", row["total_distance"])
	}
	if row["total_distance_unit"] != "km" {
		t.Errorf("expected total_distance_unit=km, got %v", row["total_distance_unit"])
	}
}

func TestParseFile_WorkoutStatistics_AttributeTakesPriority(t *testing.T) {
	// Attribute value (100) should win over WorkoutStatistics value (200)
	xml := makeTestXML(`
  <Workout workoutActivityType="HKWorkoutActivityTypeRunning" duration="30" durationUnit="min" totalEnergyBurned="100" totalEnergyBurnedUnit="kcal" sourceName="Watch" startDate="2024-01-01 08:00:00 +0000" endDate="2024-01-01 08:30:00 +0000">
    <WorkoutStatistics type="HKQuantityTypeIdentifierActiveEnergyBurned" sum="200" unit="kcal"/>
    <WorkoutStatistics type="HKQuantityTypeIdentifierDistanceWalkingRunning" sum="5.0" unit="km"/>
  </Workout>
`)
	f := writeTestXML(t, xml)
	db := tempDB(t)

	result, err := ParseFile(f, db, nil)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if result.Workouts != 1 {
		t.Fatalf("expected 1 workout, got %d", result.Workouts)
	}

	rows, err := db.QueryRows(storage.QueryParams{Table: "workouts", Limit: 10})
	if err != nil {
		t.Fatalf("query error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	row := rows[0]
	// Attribute (100) should take priority over WorkoutStatistics (200)
	if row["total_energy_burned"] != float64(100) {
		t.Errorf("expected total_energy_burned=100 (attribute priority), got %v", row["total_energy_burned"])
	}
	// Distance was not in attributes — should be populated from WorkoutStatistics
	if row["total_distance"] != 5.0 {
		t.Errorf("expected total_distance=5.0 from WorkoutStatistics, got %v", row["total_distance"])
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

func TestParseFile_BloodPressure_DiastolicBeforeSystolic(t *testing.T) {
	// Diastolic arrives first in XML — staging should still pair correctly
	xmlContent := makeTestXML(`
  <Record type="HKQuantityTypeIdentifierBloodPressureDiastolic" sourceName="Watch" unit="mmHg" value="80" startDate="2024-01-01 08:00:00 +0000" endDate="2024-01-01 08:01:00 +0000"/>
  <Record type="HKQuantityTypeIdentifierBloodPressureSystolic" sourceName="Watch" unit="mmHg" value="120" startDate="2024-01-01 08:00:00 +0000" endDate="2024-01-01 08:01:00 +0000"/>
`)
	f := writeTestXML(t, xmlContent)
	db := tempDB(t)

	result, err := ParseFile(f, db, nil)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("expected 1 paired record, got %d", result.Total)
	}
	count, err := db.CountRows("blood-pressure")
	if err != nil {
		t.Fatalf("counting blood_pressure: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 blood_pressure row, got %d", count)
	}
}

func TestParseFile_BloodPressure_MultipleTimestamps(t *testing.T) {
	xmlContent := makeTestXML(`
  <Record type="HKQuantityTypeIdentifierBloodPressureSystolic" sourceName="Watch" unit="mmHg" value="120" startDate="2024-01-01 08:00:00 +0000" endDate="2024-01-01 08:01:00 +0000"/>
  <Record type="HKQuantityTypeIdentifierBloodPressureDiastolic" sourceName="Watch" unit="mmHg" value="80" startDate="2024-01-01 08:00:00 +0000" endDate="2024-01-01 08:01:00 +0000"/>
  <Record type="HKQuantityTypeIdentifierBloodPressureSystolic" sourceName="Watch" unit="mmHg" value="118" startDate="2024-01-02 08:00:00 +0000" endDate="2024-01-02 08:01:00 +0000"/>
  <Record type="HKQuantityTypeIdentifierBloodPressureDiastolic" sourceName="Watch" unit="mmHg" value="76" startDate="2024-01-02 08:00:00 +0000" endDate="2024-01-02 08:01:00 +0000"/>
`)
	f := writeTestXML(t, xmlContent)
	db := tempDB(t)

	result, err := ParseFile(f, db, nil)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("expected 2 paired records, got %d", result.Total)
	}
	count, err := db.CountRows("blood-pressure")
	if err != nil {
		t.Fatalf("counting blood_pressure: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 blood_pressure rows, got %d", count)
	}
}

func TestParseFile_BloodPressure_DifferentSources(t *testing.T) {
	// Watch and iPhone both measure BP at the same time — should produce 2 rows
	xmlContent := makeTestXML(`
  <Record type="HKQuantityTypeIdentifierBloodPressureSystolic" sourceName="Watch" unit="mmHg" value="120" startDate="2024-01-01 08:00:00 +0000" endDate="2024-01-01 08:01:00 +0000"/>
  <Record type="HKQuantityTypeIdentifierBloodPressureDiastolic" sourceName="Watch" unit="mmHg" value="80" startDate="2024-01-01 08:00:00 +0000" endDate="2024-01-01 08:01:00 +0000"/>
  <Record type="HKQuantityTypeIdentifierBloodPressureSystolic" sourceName="iPhone" unit="mmHg" value="122" startDate="2024-01-01 08:00:00 +0000" endDate="2024-01-01 08:01:00 +0000"/>
  <Record type="HKQuantityTypeIdentifierBloodPressureDiastolic" sourceName="iPhone" unit="mmHg" value="82" startDate="2024-01-01 08:00:00 +0000" endDate="2024-01-01 08:01:00 +0000"/>
`)
	f := writeTestXML(t, xmlContent)
	db := tempDB(t)

	result, err := ParseFile(f, db, nil)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("expected 2 paired records (one per source), got %d", result.Total)
	}
	count, err := db.CountRows("blood-pressure")
	if err != nil {
		t.Fatalf("counting blood_pressure: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 blood_pressure rows, got %d", count)
	}
}

func TestParseFile_BloodPressure_OnlyBPInFile(t *testing.T) {
	xmlContent := makeTestXML(`
  <Record type="HKQuantityTypeIdentifierBloodPressureSystolic" sourceName="Watch" unit="mmHg" value="115" startDate="2024-01-01 09:00:00 +0000" endDate="2024-01-01 09:01:00 +0000"/>
  <Record type="HKQuantityTypeIdentifierBloodPressureDiastolic" sourceName="Watch" unit="mmHg" value="75" startDate="2024-01-01 09:00:00 +0000" endDate="2024-01-01 09:01:00 +0000"/>
`)
	f := writeTestXML(t, xmlContent)
	db := tempDB(t)

	result, err := ParseFile(f, db, nil)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("expected 1 record, got %d", result.Total)
	}
	count, _ := db.CountRows("blood-pressure")
	if count != 1 {
		t.Errorf("expected 1 blood_pressure row, got %d", count)
	}
}

func TestParseFile_BloodPressure_UnpairedBothKinds(t *testing.T) {
	// Unpaired systolic + unpaired diastolic at different timestamps — 0 rows
	xmlContent := makeTestXML(`
  <Record type="HKQuantityTypeIdentifierBloodPressureSystolic" sourceName="Watch" unit="mmHg" value="120" startDate="2024-01-01 08:00:00 +0000" endDate="2024-01-01 08:01:00 +0000"/>
  <Record type="HKQuantityTypeIdentifierBloodPressureDiastolic" sourceName="Watch" unit="mmHg" value="80" startDate="2024-01-01 09:00:00 +0000" endDate="2024-01-01 09:01:00 +0000"/>
`)
	f := writeTestXML(t, xmlContent)
	db := tempDB(t)

	result, err := ParseFile(f, db, nil)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if result.Total != 0 {
		t.Errorf("expected 0 complete pairs, got %d", result.Total)
	}
	count, _ := db.CountRows("blood-pressure")
	if count != 0 {
		t.Errorf("expected 0 blood_pressure rows, got %d", count)
	}
}

// --- normalizeTimestamp ---

func TestNormalizeTimestamp(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"positive offset", "2024-01-01 22:00:00 +0530", "2024-01-01 22:00:00"},
		{"negative offset", "2024-03-15 08:30:00 -0800", "2024-03-15 08:30:00"},
		{"UTC offset", "2024-06-01 12:00:00 +0000", "2024-06-01 12:00:00"},
		{"no offset", "2024-01-01 22:00:00", "2024-01-01 22:00:00"},
		{"empty string", "", ""},
		{"short string", "2024-01-01", "2024-01-01"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeTimestamp(tt.input)
			if got != tt.want {
				t.Errorf("normalizeTimestamp(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseXML_TimestampsNormalized(t *testing.T) {
	xmlData := `<HealthData>
		<Record type="HKCategoryTypeIdentifierSleepAnalysis"
			sourceName="Watch" value="HKCategoryValueSleepAnalysisAsleepCore"
			startDate="2024-01-01 23:00:00 +0530" endDate="2024-01-02 07:00:00 +0530"/>
		<Record type="HKQuantityTypeIdentifierStepCount"
			sourceName="iPhone" value="500" unit="count"
			startDate="2024-01-01 08:00:00 -0800" endDate="2024-01-01 08:30:00 -0800"/>
	</HealthData>`

	db := tempDB(t)
	_, err := parseXML(strings.NewReader(xmlData), db, nil)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	// Verify sleep timestamps have no offset
	sleepRows, err := db.QueryRows(storage.QueryParams{Table: "sleep", Limit: 10})
	if err != nil {
		t.Fatalf("query sleep: %v", err)
	}
	if len(sleepRows) != 1 {
		t.Fatalf("expected 1 sleep row, got %d", len(sleepRows))
	}
	startDate := sleepRows[0]["start_date"].(string)
	if startDate != "2024-01-01 23:00:00" {
		t.Errorf("expected '2024-01-01 23:00:00', got %q", startDate)
	}

	// Verify steps timestamps have no offset
	stepRows, err := db.QueryRows(storage.QueryParams{Table: "steps", Limit: 10})
	if err != nil {
		t.Fatalf("query steps: %v", err)
	}
	if len(stepRows) != 1 {
		t.Fatalf("expected 1 step row, got %d", len(stepRows))
	}
	stepStart := stepRows[0]["start_date"].(string)
	if stepStart != "2024-01-01 08:00:00" {
		t.Errorf("expected '2024-01-01 08:00:00', got %q", stepStart)
	}
}

func TestParseXML_WorkoutTimestampsNormalized(t *testing.T) {
	xmlData := `<HealthData>
		<Workout workoutActivityType="HKWorkoutActivityTypeRunning"
			duration="30" durationUnit="min"
			totalDistance="5" totalDistanceUnit="km"
			totalEnergyBurned="300" totalEnergyBurnedUnit="kcal"
			sourceName="Watch"
			startDate="2024-01-01 08:00:00 +0530" endDate="2024-01-01 08:30:00 +0530"/>
	</HealthData>`

	db := tempDB(t)
	_, err := parseXML(strings.NewReader(xmlData), db, nil)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	rows, err := db.QueryRows(storage.QueryParams{Table: "workouts", Limit: 10})
	if err != nil {
		t.Fatalf("query workouts: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 workout row, got %d", len(rows))
	}
	if got := rows[0]["start_date"].(string); got != "2024-01-01 08:00:00" {
		t.Errorf("start_date: expected '2024-01-01 08:00:00', got %q", got)
	}
	if got := rows[0]["end_date"].(string); got != "2024-01-01 08:30:00" {
		t.Errorf("end_date: expected '2024-01-01 08:30:00', got %q", got)
	}
}
