package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/BRO3886/healthsync/internal/parser"
	"github.com/BRO3886/healthsync/internal/storage"
)

type handlers struct {
	db  *storage.DB
	job *parseJob
}

// parseJob tracks the state of an async parse operation.
type parseJob struct {
	mu        sync.RWMutex
	running   bool
	records   atomic.Int64
	workouts  atomic.Int64
	startedAt time.Time
	result    *parser.ParseResult
	err       error
}

func (j *parseJob) status() map[string]any {
	j.mu.RLock()
	defer j.mu.RUnlock()

	s := map[string]any{
		"running":  j.running,
		"records":  j.records.Load(),
		"workouts": j.workouts.Load(),
	}

	if !j.startedAt.IsZero() {
		s["started_at"] = j.startedAt.Format(time.RFC3339)
		s["elapsed"] = time.Since(j.startedAt).Round(time.Millisecond).String()
	}

	if j.result != nil {
		s["status"] = "completed"
		s["total_records"] = j.result.Total
		s["total_workouts"] = j.result.Workouts
	} else if j.err != nil {
		s["status"] = "failed"
		s["error"] = j.err.Error()
	} else if j.running {
		s["status"] = "running"
	} else {
		s["status"] = "idle"
	}

	return s
}

func (h *handlers) handleUpload(w http.ResponseWriter, r *http.Request) {
	// Check if a parse is already running
	h.job.mu.RLock()
	running := h.job.running
	h.job.mu.RUnlock()
	if running {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]any{
			"error":  "a parse job is already running",
			"status": h.job.status(),
		})
		return
	}

	// Limit upload to 2GB
	r.Body = http.MaxBytesReader(w, r.Body, 2<<30)

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, fmt.Sprintf("reading upload: %v", err), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Write to temp file (zip needs random access)
	tmpFile, err := os.CreateTemp("", "healthsync-upload-*.zip")
	if err != nil {
		http.Error(w, "creating temp file", http.StatusInternalServerError)
		return
	}
	tmpPath := tmpFile.Name()

	if _, err := io.Copy(tmpFile, file); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		http.Error(w, "saving upload", http.StatusInternalServerError)
		return
	}
	tmpFile.Close()

	ext := filepath.Ext(header.Filename)
	if ext != ".zip" && ext != ".xml" {
		os.Remove(tmpPath)
		http.Error(w, "unsupported file type (expected .zip or .xml)", http.StatusBadRequest)
		return
	}

	log.Printf("Upload received: %s (%d bytes), starting async parse", header.Filename, header.Size)

	// Reset job state and start async
	h.job.mu.Lock()
	h.job.running = true
	h.job.records.Store(0)
	h.job.workouts.Store(0)
	h.job.startedAt = time.Now()
	h.job.result = nil
	h.job.err = nil
	h.job.mu.Unlock()

	go func() {
		defer os.Remove(tmpPath)
		defer func() {
			h.job.mu.Lock()
			h.job.running = false
			h.job.mu.Unlock()
		}()

		progress := func(records int64, workouts int64) {
			h.job.records.Store(records)
			h.job.workouts.Store(workouts)
		}

		result, err := parser.ParseFile(tmpPath, h.db, progress)

		h.job.mu.Lock()
		h.job.result = result
		h.job.err = err
		h.job.mu.Unlock()

		if err != nil {
			log.Printf("Parse failed: %v", err)
		} else {
			log.Printf("Parse completed: %d records, %d workouts", result.Total, result.Workouts)
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]any{
		"status":  "accepted",
		"message": "file uploaded, parsing in background",
		"poll":    "/api/upload/status",
	})
}

func (h *handlers) handleUploadStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.job.status())
}

func (h *handlers) handleQuery(w http.ResponseWriter, r *http.Request) {
	table := chi.URLParam(r, "table")

	if _, ok := storage.TableNameMap[table]; !ok {
		http.Error(w, fmt.Sprintf("unknown table: %q", table), http.StatusBadRequest)
		return
	}

	params := storage.QueryParams{
		Table: table,
		From:  r.URL.Query().Get("from"),
		To:    r.URL.Query().Get("to"),
		Limit: 50,
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 {
			params.Limit = v
		}
	}

	rows, err := h.db.QueryRows(params)
	if err != nil {
		http.Error(w, fmt.Sprintf("query error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rows)
}
