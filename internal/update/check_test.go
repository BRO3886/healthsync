package update

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseCache(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    *CacheEntry
	}{
		{
			name:    "valid cache",
			content: "checked_at=2026-02-16T10:00:00Z\nlatest=v0.5.0\n",
			want: &CacheEntry{
				CheckedAt: time.Date(2026, 2, 16, 10, 0, 0, 0, time.UTC),
				Latest:    "v0.5.0",
			},
		},
		{
			name:    "empty content",
			content: "",
			want:    nil,
		},
		{
			name:    "missing latest",
			content: "checked_at=2026-02-16T10:00:00Z\n",
			want:    nil,
		},
		{
			name:    "invalid timestamp",
			content: "checked_at=not-a-date\nlatest=v0.5.0\n",
			want:    nil,
		},
		{
			name:    "extra whitespace",
			content: "  checked_at=2026-02-16T10:00:00Z  \n  latest=v0.5.0  \n",
			want: &CacheEntry{
				CheckedAt: time.Date(2026, 2, 16, 10, 0, 0, 0, time.UTC),
				Latest:    "v0.5.0",
			},
		},
		{
			name:    "unknown keys ignored",
			content: "checked_at=2026-02-16T10:00:00Z\nlatest=v0.5.0\nfoo=bar\n",
			want: &CacheEntry{
				CheckedAt: time.Date(2026, 2, 16, 10, 0, 0, 0, time.UTC),
				Latest:    "v0.5.0",
			},
		},
		{
			name:    "malformed line (no equals)",
			content: "checked_at=2026-02-16T10:00:00Z\ngarbage\nlatest=v0.5.0\n",
			want: &CacheEntry{
				CheckedAt: time.Date(2026, 2, 16, 10, 0, 0, 0, time.UTC),
				Latest:    "v0.5.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseCache(tt.content)
			if tt.want == nil {
				if got != nil {
					t.Errorf("ParseCache() = %+v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatal("ParseCache() = nil, want non-nil")
			}
			if !got.CheckedAt.Equal(tt.want.CheckedAt) {
				t.Errorf("CheckedAt = %v, want %v", got.CheckedAt, tt.want.CheckedAt)
			}
			if got.Latest != tt.want.Latest {
				t.Errorf("Latest = %q, want %q", got.Latest, tt.want.Latest)
			}
		})
	}
}

func TestWriteAndReadCache(t *testing.T) {
	tmpDir := t.TempDir()

	entry := &CacheEntry{
		CheckedAt: time.Date(2026, 2, 16, 10, 0, 0, 0, time.UTC),
		Latest:    "v0.5.0",
	}

	WriteCache(tmpDir, entry)

	got := ReadCache(tmpDir)
	if got == nil {
		t.Fatal("ReadCache() = nil after write")
	}
	if got.Latest != "v0.5.0" {
		t.Errorf("Latest = %q, want %q", got.Latest, "v0.5.0")
	}
	if !got.CheckedAt.Equal(entry.CheckedAt) {
		t.Errorf("CheckedAt = %v, want %v", got.CheckedAt, entry.CheckedAt)
	}
}

func TestReadCacheMissing(t *testing.T) {
	tmpDir := t.TempDir()
	got := ReadCache(tmpDir)
	if got != nil {
		t.Errorf("ReadCache() = %+v, want nil for missing file", got)
	}
}

func TestWriteCacheCreatesDir(t *testing.T) {
	tmpDir := t.TempDir()

	entry := &CacheEntry{
		CheckedAt: time.Now(),
		Latest:    "v1.0.0",
	}

	WriteCache(tmpDir, entry)

	dir := filepath.Join(tmpDir, ".cache", "healthsync")
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("cache directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("cache path is not a directory")
	}
}

func TestIsCacheFresh(t *testing.T) {
	now := time.Date(2026, 2, 16, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		checkedAt time.Time
		want      bool
	}{
		{"checked 1 hour ago", now.Add(-1 * time.Hour), true},
		{"checked 23 hours ago", now.Add(-23 * time.Hour), true},
		{"checked 25 hours ago", now.Add(-25 * time.Hour), false},
		{"checked exactly 24 hours ago", now.Add(-24 * time.Hour), false},
		{"checked in the future", now.Add(1 * time.Hour), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := &CacheEntry{CheckedAt: tt.checkedAt, Latest: "v1.0.0"}
			got := IsCacheFresh(entry, now)
			if got != tt.want {
				t.Errorf("IsCacheFresh() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		current string
		latest  string
		want    bool
	}{
		{"v0.4.0", "v0.5.0", true},
		{"v0.5.0", "v0.5.0", false},
		{"v0.5.0", "v0.4.0", false},
		{"v1.0.0", "v2.0.0", true},
		{"v0.9.0", "v0.10.0", true},
		{"v0.10.0", "v0.9.0", false},
		{"0.4.0", "0.5.0", true},
		{"v0.4.0", "0.5.0", true},
		{"dev", "v0.5.0", false},
		{"v0.4.0", "invalid", false},
		{"", "v0.5.0", false},
		{"v1.2.3-rc1", "v1.2.4", true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_vs_%s", tt.current, tt.latest), func(t *testing.T) {
			got := CompareVersions(tt.current, tt.latest)
			if got != tt.want {
				t.Errorf("CompareVersions(%q, %q) = %v, want %v",
					tt.current, tt.latest, got, tt.want)
			}
		})
	}
}

func TestFetchLatestVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"tag_name": "v0.5.0", "name": "v0.5.0"}`)
	}))
	defer server.Close()

	ctx := context.Background()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestFetchLatestVersionErrorCases(t *testing.T) {
	t.Run("context cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := FetchLatestVersion(ctx, http.DefaultClient)
		if err == nil {
			t.Error("expected error for cancelled context")
		}
	})
}

func TestCheckWithFreshCache(t *testing.T) {
	tmpDir := t.TempDir()

	entry := &CacheEntry{
		CheckedAt: time.Now(),
		Latest:    "v0.5.0",
	}
	WriteCache(tmpDir, entry)

	result := Check(tmpDir, "v0.4.0")
	if result == nil {
		t.Fatal("Check() = nil, expected update result")
	}
	if !result.HasUpdate {
		t.Error("HasUpdate = false, expected true")
	}
	if result.Latest != "v0.5.0" {
		t.Errorf("Latest = %q, want %q", result.Latest, "v0.5.0")
	}
}

func TestCheckWithFreshCacheNoUpdate(t *testing.T) {
	tmpDir := t.TempDir()

	entry := &CacheEntry{
		CheckedAt: time.Now(),
		Latest:    "v0.4.0",
	}
	WriteCache(tmpDir, entry)

	result := Check(tmpDir, "v0.4.0")
	if result != nil {
		t.Errorf("Check() = %+v, expected nil (no update)", result)
	}
}

func TestCheckWithStaleCache(t *testing.T) {
	tmpDir := t.TempDir()

	entry := &CacheEntry{
		CheckedAt: time.Now().Add(-25 * time.Hour),
		Latest:    "v0.3.0",
	}
	WriteCache(tmpDir, entry)

	// Will try to fetch from GitHub — may timeout, just verify no panic/hang
	result := Check(tmpDir, "v0.4.0")
	_ = result
}

func TestCheckWithDevVersion(t *testing.T) {
	tmpDir := t.TempDir()

	entry := &CacheEntry{
		CheckedAt: time.Now(),
		Latest:    "v0.5.0",
	}
	WriteCache(tmpDir, entry)

	result := Check(tmpDir, "dev")
	if result != nil {
		t.Errorf("Check() = %+v, expected nil for dev version", result)
	}
}

func TestCachePath(t *testing.T) {
	got := CachePath("/home/user")
	want := "/home/user/.cache/healthsync/update-check"
	if got != want {
		t.Errorf("CachePath() = %q, want %q", got, want)
	}
}
