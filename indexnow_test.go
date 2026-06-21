package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestIndexNowKeyFile guards that the IndexNow key file exists, that its
// contents equal the filename stem, and that scripts/indexnow.sh references
// the same key. If either drifts the submission will 403 in production.
func TestIndexNowKeyFile(t *testing.T) {
	const key = "09d76431580e356eafd8d91aeecc0906"
	keyFile := filepath.Join("website", "static", key+".txt")

	data, err := os.ReadFile(keyFile)
	if err != nil {
		t.Fatalf("key file %s not found: %v", keyFile, err)
	}
	got := strings.TrimRight(string(data), "\r\n")
	if got != key {
		t.Errorf("key file contents %q != filename stem %q", got, key)
	}

	script := filepath.Join("scripts", "indexnow.sh")
	f, err := os.Open(script)
	if err != nil {
		t.Fatalf("indexnow.sh not found: %v", err)
	}
	defer f.Close()

	found := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), key) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("scripts/indexnow.sh does not reference key %q", key)
	}
}
