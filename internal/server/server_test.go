package server

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"archive/zip"

	"github.com/go-chi/chi/v5"

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

func newTestHandlers(t *testing.T) *handlers {
	t.Helper()
	return &handlers{db: tempDB(t), job: &parseJob{}}
}

func testRouter(h *handlers) *chi.Mux {
	r := chi.NewRouter()
	r.Post("/api/upload", h.handleUpload)
	r.Get("/api/upload/status", h.handleUploadStatus)
	r.Get("/api/health/{table}", h.handleQuery)
	return r
}

func makeTestZip(t *testing.T, xmlContent string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	zf, err := w.Create("apple_health_export/export.xml")
	if err != nil {
		t.Fatalf("creating zip entry: %v", err)
	}
	zf.Write([]byte(xmlContent))
	w.Close()
	return buf.Bytes()
}

func uploadFile(t *testing.T, router http.Handler, filename string, content []byte) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("creating form file: %v", err)
	}
	part.Write(content)
	mw.Close()

	req := httptest.NewRequest("POST", "/api/upload", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

// --- handleUpload ---

func TestHandleUpload_ValidZip(t *testing.T) {
	h := newTestHandlers(t)
	router := testRouter(h)

	xml := `<?xml version="1.0" encoding="UTF-8"?>
<HealthData locale="en_US">
  <Record type="HKQuantityTypeIdentifierHeartRate" sourceName="Watch" unit="count/min" value="72" startDate="2024-01-01 00:00:00 +0000" endDate="2024-01-01 00:01:00 +0000"/>
</HealthData>`
	zipData := makeTestZip(t, xml)

	rr := uploadFile(t, router, "export.zip", zipData)

	if rr.Code != http.StatusAccepted {
		t.Errorf("expected 202 Accepted, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["status"] != "accepted" {
		t.Errorf("expected status accepted, got %v", resp["status"])
	}
	if resp["poll"] != "/api/upload/status" {
		t.Errorf("expected poll URL, got %v", resp["poll"])
	}
}

func TestHandleUpload_MissingFile(t *testing.T) {
	h := newTestHandlers(t)
	router := testRouter(h)

	req := httptest.NewRequest("POST", "/api/upload", strings.NewReader(""))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=xxx")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing file, got %d", rr.Code)
	}
}

func TestHandleUpload_UnsupportedExtension(t *testing.T) {
	h := newTestHandlers(t)
	router := testRouter(h)

	rr := uploadFile(t, router, "data.csv", []byte("some,data"))

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for .csv, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleUpload_ConflictWhileRunning(t *testing.T) {
	h := newTestHandlers(t)
	router := testRouter(h)

	// Simulate a running job
	h.job.mu.Lock()
	h.job.running = true
	h.job.mu.Unlock()

	zipData := makeTestZip(t, `<?xml version="1.0"?><HealthData/>`)
	rr := uploadFile(t, router, "export.zip", zipData)

	if rr.Code != http.StatusConflict {
		t.Errorf("expected 409 Conflict while running, got %d", rr.Code)
	}
}

// --- handleUploadStatus ---

func TestHandleUploadStatus_Idle(t *testing.T) {
	h := newTestHandlers(t)
	router := testRouter(h)

	req := httptest.NewRequest("GET", "/api/upload/status", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["status"] != "idle" {
		t.Errorf("expected idle status, got %v", resp["status"])
	}
}

func TestHandleUploadStatus_Running(t *testing.T) {
	h := newTestHandlers(t)
	router := testRouter(h)

	h.job.mu.Lock()
	h.job.running = true
	h.job.startedAt = time.Now()
	h.job.records.Store(5000)
	h.job.workouts.Store(10)
	h.job.mu.Unlock()

	req := httptest.NewRequest("GET", "/api/upload/status", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["status"] != "running" {
		t.Errorf("expected running, got %v", resp["status"])
	}
	if resp["records"].(float64) != 5000 {
		t.Errorf("expected 5000 records, got %v", resp["records"])
	}
}

func TestHandleUploadStatus_Completed(t *testing.T) {
	h := newTestHandlers(t)

	// Upload a small file and wait for completion
	router := testRouter(h)
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<HealthData locale="en_US">
  <Record type="HKQuantityTypeIdentifierHeartRate" sourceName="Watch" unit="count/min" value="72" startDate="2024-01-01 00:00:00 +0000" endDate="2024-01-01 00:01:00 +0000"/>
</HealthData>`
	zipData := makeTestZip(t, xml)
	uploadFile(t, router, "export.zip", zipData)

	// Wait for async parse to complete
	for i := 0; i < 50; i++ {
		time.Sleep(100 * time.Millisecond)
		h.job.mu.RLock()
		done := !h.job.running && h.job.result != nil
		h.job.mu.RUnlock()
		if done {
			break
		}
	}

	req := httptest.NewRequest("GET", "/api/upload/status", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["status"] != "completed" {
		t.Errorf("expected completed, got %v", resp["status"])
	}
}

// --- parseJob.status ---

func TestParseJobStatus_Failed(t *testing.T) {
	j := &parseJob{}
	j.running = false
	j.err = io.EOF
	j.startedAt = time.Now().Add(-5 * time.Second)

	s := j.status()
	if s["status"] != "failed" {
		t.Errorf("expected failed, got %v", s["status"])
	}
	if s["error"] != "EOF" {
		t.Errorf("expected EOF error, got %v", s["error"])
	}
}

// --- handleQuery ---

func TestHandleQuery_ValidTable(t *testing.T) {
	h := newTestHandlers(t)
	router := testRouter(h)

	// Insert some data
	cols := []string{"source_name", "start_date", "end_date", "value", "unit"}
	h.db.BatchInsertRecords("heart_rate", cols, [][]any{
		{"Watch", "2024-01-01 00:00:00", "2024-01-01 00:01:00", 72.0, "count/min"},
	})

	req := httptest.NewRequest("GET", "/api/health/heart-rate?limit=10", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var rows []map[string]any
	json.Unmarshal(rr.Body.Bytes(), &rows)
	if len(rows) != 1 {
		t.Errorf("expected 1 row, got %d", len(rows))
	}
}

func TestHandleQuery_UnknownTable(t *testing.T) {
	h := newTestHandlers(t)
	router := testRouter(h)

	req := httptest.NewRequest("GET", "/api/health/bogus", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for unknown table, got %d", rr.Code)
	}
}

func TestHandleQuery_CustomLimit(t *testing.T) {
	h := newTestHandlers(t)
	router := testRouter(h)

	cols := []string{"source_name", "start_date", "end_date", "value", "unit"}
	records := make([][]any, 10)
	for i := range records {
		records[i] = []any{"Watch", "2024-01-01 00:00:00", "2024-01-01 00:01:00", float64(60 + i), "count/min"}
	}
	h.db.BatchInsertRecords("heart_rate", cols, records)

	req := httptest.NewRequest("GET", "/api/health/heart-rate?limit=3", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	var rows []map[string]any
	json.Unmarshal(rr.Body.Bytes(), &rows)
	if len(rows) != 3 {
		t.Errorf("expected 3 rows with limit=3, got %d", len(rows))
	}
}

func TestHandleQuery_InvalidLimitUsesDefault(t *testing.T) {
	h := newTestHandlers(t)
	router := testRouter(h)

	req := httptest.NewRequest("GET", "/api/health/heart-rate?limit=abc", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Should not error, just use default limit
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestHandleQuery_WithDateFilters(t *testing.T) {
	h := newTestHandlers(t)
	router := testRouter(h)

	cols := []string{"source_name", "start_date", "end_date", "value", "unit"}
	records := [][]any{
		{"Watch", "2024-01-01 00:00:00", "2024-01-01 00:01:00", 72.0, "count/min"},
		{"Watch", "2024-06-01 00:00:00", "2024-06-01 00:01:00", 75.0, "count/min"},
		{"Watch", "2024-12-01 00:00:00", "2024-12-01 00:01:00", 80.0, "count/min"},
	}
	h.db.BatchInsertRecords("heart_rate", cols, records)

	req := httptest.NewRequest("GET", "/api/health/heart-rate?from=2024-03-01&to=2024-09-01", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	var rows []map[string]any
	json.Unmarshal(rr.Body.Bytes(), &rows)
	if len(rows) != 1 {
		t.Errorf("expected 1 row with date filters, got %d", len(rows))
	}
}

func TestHandleQuery_EmptyResult(t *testing.T) {
	h := newTestHandlers(t)
	router := testRouter(h)

	req := httptest.NewRequest("GET", "/api/health/steps", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	// Empty result should be null or empty array in JSON
	body := strings.TrimSpace(rr.Body.String())
	if body != "null" && body != "[]" {
		t.Errorf("expected null or [], got %s", body)
	}
}

// --- handleUpload with XML file ---

func TestHandleUpload_XMLFile(t *testing.T) {
	h := newTestHandlers(t)
	router := testRouter(h)

	// Create a temp XML file to reference in test
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<HealthData locale="en_US">
  <Record type="HKQuantityTypeIdentifierHeartRate" sourceName="Watch" unit="count/min" value="72" startDate="2024-01-01 00:00:00 +0000" endDate="2024-01-01 00:01:00 +0000"/>
</HealthData>`

	// Write to a temp file that the handler can access
	tmpDir := t.TempDir()
	xmlPath := filepath.Join(tmpDir, "export.xml")
	os.WriteFile(xmlPath, []byte(xml), 0644)

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, _ := mw.CreateFormFile("file", "export.xml")
	part.Write([]byte(xml))
	mw.Close()

	req := httptest.NewRequest("POST", "/api/upload", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// XML uploads get written as temp .zip files, but the extension check
	// is on the original filename. .xml should be accepted.
	if rr.Code != http.StatusAccepted {
		t.Errorf("expected 202 for .xml upload, got %d: %s", rr.Code, rr.Body.String())
	}
}
