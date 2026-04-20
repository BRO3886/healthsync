package parser

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BRO3886/healthsync/internal/storage"
)

// ParseResult holds the parsing outcome.
type ParseResult struct {
	Stats    []*storage.InsertStats
	Total    int64
	Workouts int64
}

// ProgressFunc is called periodically during parsing with the count of records processed.
type ProgressFunc func(records int64, workouts int64)

// ParseFile parses an Apple Health export file (.zip or .xml) and inserts data into the DB.
func ParseFile(path string, db *storage.DB, progress ProgressFunc) (*ParseResult, error) {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".zip":
		return parseZip(path, db, progress)
	case ".xml":
		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("opening xml file: %w", err)
		}
		defer f.Close()
		return parseXML(f, db, progress)
	default:
		return nil, fmt.Errorf("unsupported file type: %s (expected .zip or .xml)", ext)
	}
}

func parseZip(path string, db *storage.DB, progress ProgressFunc) (*ParseResult, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("opening zip: %w", err)
	}
	defer r.Close()

	entry, err := findHealthExport(r.File)
	if err != nil {
		return nil, err
	}

	rc, err := entry.Open()
	if err != nil {
		return nil, fmt.Errorf("opening %s in zip: %w", entry.Name, err)
	}
	defer rc.Close()
	return parseXML(rc, db, progress)
}

// findHealthExport locates the HealthKit export XML inside an Apple Health zip.
//
// The filename is not reliable: Apple localizes it per device language
// ("export.xml" on English, "导出.xml" on Chinese, and so on), users can
// rename the file, and re-zipped archives may include stray xml files. The
// zip also always contains a sibling "export_cda.xml" in a completely
// different schema (CDA / ClinicalDocument) which must never be parsed as
// HealthKit data.
//
// We therefore identify the export by content, not by name: any .xml entry
// whose head contains "<HealthData" is the HealthKit export. The CDA file
// starts with "<ClinicalDocument" so it is naturally rejected.
func findHealthExport(files []*zip.File) (*zip.File, error) {
	var candidates []*zip.File
	for _, f := range files {
		if f.FileInfo().IsDir() {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(f.Name), ".xml") {
			continue
		}
		candidates = append(candidates, f)
	}

	for _, f := range candidates {
		ok, err := looksLikeHealthKitXML(f)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", f.Name, err)
		}
		if ok {
			return f, nil
		}
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no .xml entries in zip archive")
	}
	return nil, fmt.Errorf("no HealthKit export XML found in zip archive (found %d .xml entries but none contained a <HealthData> root — is this an Apple Health export?)", len(candidates))
}

// looksLikeHealthKitXML reads the head of a zip entry and checks for the
// HealthKit root element. 4 KiB is enough to skip the XML prolog, optional
// DOCTYPE, and whitespace before <HealthData> even on exports with long
// DTDs.
func looksLikeHealthKitXML(f *zip.File) (bool, error) {
	rc, err := f.Open()
	if err != nil {
		return false, err
	}
	defer rc.Close()

	head := make([]byte, 4096)
	n, err := io.ReadFull(rc, head)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return false, err
	}
	return bytes.Contains(head[:n], []byte("<HealthData")), nil
}

// stripDTD pipes the input through a goroutine that removes the DOCTYPE section.
// Uses io.Pipe so no bytes are lost to buffering.
func stripDTD(r io.Reader) io.Reader {
	pr, pw := io.Pipe()

	go func() {
		scanner := bufio.NewScanner(r)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
		inDTD := false

		for scanner.Scan() {
			line := scanner.Text()

			if strings.Contains(line, "<!DOCTYPE") {
				inDTD = true
				continue
			}
			if inDTD {
				if strings.Contains(line, "]>") {
					inDTD = false
				}
				continue
			}

			if _, err := pw.Write([]byte(line + "\n")); err != nil {
				pw.CloseWithError(err)
				return
			}
		}

		if err := scanner.Err(); err != nil {
			pw.CloseWithError(err)
		} else {
			pw.Close()
		}
	}()

	return pr
}

func parseXML(r io.Reader, db *storage.DB, progress ProgressFunc) (*ParseResult, error) {
	cleaned := stripDTD(r)
	decoder := xml.NewDecoder(cleaned)
	decoder.Strict = false

	// Buffers per table
	buffers := make(map[string][][]any)
	var workoutBuf [][]any

	var recordCount int64
	var workoutCount int64
	batchSize := 1000

	// bpKey identifies a blood pressure reading by source + start time.
	type bpKey struct{ source, start string }
	type bpEntry struct {
		endDate   string
		systolic  any
		diastolic any
		unit      string
	}
	bpStaging := make(map[bpKey]*bpEntry)

	flushTable := func(table string) error {
		if len(buffers[table]) == 0 {
			return nil
		}
		cols := RecordColumns(table)
		if table == "blood_pressure" {
			cols = BloodPressureColumns()
		}
		_, err := db.BatchInsertRecords(table, cols, buffers[table])
		if err != nil {
			return err
		}
		buffers[table] = nil
		return nil
	}

	flushWorkouts := func() error {
		if len(workoutBuf) == 0 {
			return nil
		}
		_, err := db.BatchInsertRecords("workouts", WorkoutColumns(), workoutBuf)
		if err != nil {
			return err
		}
		workoutBuf = nil
		return nil
	}

	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}

		switch se.Name.Local {
		case "Record":
			var rec Record
			if err := decoder.DecodeElement(&rec, &se); err != nil {
				continue
			}

			table, ok := TargetRecordTypes[rec.Type]
			if !ok {
				continue
			}

			// Blood pressure: stage both values; only emit row when pair is complete.
			if table == "blood_pressure_systolic" || table == "blood_pressure_diastolic" {
				key := bpKey{rec.SourceName, normalizeTimestamp(rec.StartDate)}
				e, exists := bpStaging[key]
				if !exists {
					e = &bpEntry{endDate: normalizeTimestamp(rec.EndDate), unit: rec.Unit}
					bpStaging[key] = e
				}
				v := parseFloat(rec.Value)
				if table == "blood_pressure_systolic" {
					e.systolic = v
				} else {
					e.diastolic = v
				}
				if e.systolic != nil && e.diastolic != nil {
					row := []any{key.source, key.start, e.endDate, e.systolic, e.diastolic, e.unit}
					buffers["blood_pressure"] = append(buffers["blood_pressure"], row)
					delete(bpStaging, key)
					recordCount++
					if len(buffers["blood_pressure"]) >= batchSize {
						if err := flushTable("blood_pressure"); err != nil {
							return nil, fmt.Errorf("flushing blood_pressure: %w", err)
						}
					}
					if recordCount%10000 == 0 && progress != nil {
						progress(recordCount, workoutCount)
					}
				}
				continue
			}

			startDate := normalizeTimestamp(rec.StartDate)
			endDate := normalizeTimestamp(rec.EndDate)
			var row []any
			if noUnitTables[table] {
				row = []any{rec.SourceName, startDate, endDate, rec.Value}
			} else {
				row = []any{rec.SourceName, startDate, endDate, parseFloat(rec.Value), rec.Unit}
			}
			buffers[table] = append(buffers[table], row)
			recordCount++

			if len(buffers[table]) >= batchSize {
				if err := flushTable(table); err != nil {
					return nil, fmt.Errorf("flushing %s: %w", table, err)
				}
			}

			if recordCount%10000 == 0 && progress != nil {
				progress(recordCount, workoutCount)
			}

		case "Workout":
			var w Workout
			if err := decoder.DecodeElement(&w, &se); err != nil {
				continue
			}

			// Populate from WorkoutStatistics children if attributes not present (watchOS 10+)
			for _, s := range w.Statistics {
				switch s.Type {
				case "HKQuantityTypeIdentifierActiveEnergyBurned":
					if w.TotalEnergyBurned == "" {
						w.TotalEnergyBurned = s.Sum
						w.TotalEnergyBurnedUnit = s.Unit
					}
				case "HKQuantityTypeIdentifierDistanceWalkingRunning",
					"HKQuantityTypeIdentifierDistanceCycling",
					"HKQuantityTypeIdentifierDistanceSwimming":
					if w.TotalDistance == "" {
						w.TotalDistance = s.Sum
						w.TotalDistanceUnit = s.Unit
					}
				}
			}

			duration := parseFloat(w.Duration)
			totalDistance := parseFloat(w.TotalDistance)
			totalEnergy := parseFloat(w.TotalEnergyBurned)

			row := []any{
				w.ActivityType, w.SourceName, normalizeTimestamp(w.StartDate), normalizeTimestamp(w.EndDate),
				duration, nilIfEmpty(w.DurationUnit),
				totalDistance, nilIfEmpty(w.TotalDistanceUnit),
				totalEnergy, nilIfEmpty(w.TotalEnergyBurnedUnit),
			}
			workoutBuf = append(workoutBuf, row)
			workoutCount++

			if progress != nil {
				progress(recordCount, workoutCount)
			}

			if len(workoutBuf) >= batchSize {
				if err := flushWorkouts(); err != nil {
					return nil, fmt.Errorf("flushing workouts: %w", err)
				}
			}

		default:
			// Skip leaf elements but NOT container elements like HealthData
			// which hold all the Record/Workout children we need.
			switch se.Name.Local {
			case "HealthData":
				// Don't skip — we need to descend into its children
			default:
				if err := decoder.Skip(); err != nil && err != io.EOF {
					continue
				}
			}
		}
	}

	// Flush remaining buffers
	for table := range buffers {
		if err := flushTable(table); err != nil {
			return nil, fmt.Errorf("final flush %s: %w", table, err)
		}
	}
	if err := flushWorkouts(); err != nil {
		return nil, fmt.Errorf("final flush workouts: %w", err)
	}

	if progress != nil {
		progress(recordCount, workoutCount)
	}

	result := &ParseResult{
		Total:    recordCount,
		Workouts: workoutCount,
	}
	return result, nil
}

func parseFloat(s string) any {
	if s == "" {
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return v
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// normalizeTimestamp strips the trailing timezone offset (e.g. " +0530", " -0800")
// from Apple Health timestamps, leaving plain "YYYY-MM-DD HH:MM:SS" local time.
// This is required because SQLite's julianday()/date() cannot parse the
// space-separated ±HHMM offset format that Apple Health uses.
func normalizeTimestamp(s string) string {
	// Apple Health format: "2024-01-01 22:00:00 +0530"
	// We want: "2024-01-01 22:00:00"
	// The timestamp portion is always 19 chars ("YYYY-MM-DD HH:MM:SS").
	if len(s) > 19 && s[19] == ' ' {
		return s[:19]
	}
	return s
}
