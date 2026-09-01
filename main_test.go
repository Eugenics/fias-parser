package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadVersionFileCompactDate(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, versionFileName), []byte("20260831\nv.254\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	versionID, textVersion, dateText, ok := readVersionFile(dir)
	if !ok || versionID != 20260831 || textVersion != "v.254" || dateText != "20260831" {
		t.Fatalf("readVersionFile() = (%d, %q, %q, %t)", versionID, textVersion, dateText, ok)
	}
}
