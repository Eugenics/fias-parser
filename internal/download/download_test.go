package download

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLatestArchiveSelectsHighestVersionOfRequestedKind(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{
		"20260801_full.zip",
		"20260803_full.zip",
		"20260804_delta.zip",
		"20260805_full.zip.part",
		"gar_xml.zip",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	path, version, err := latestArchive(dir, "full")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, "20260803_full.zip"); path != want {
		t.Fatalf("archive path = %q, want %q", path, want)
	}
	if version != 20260803 {
		t.Fatalf("archive version = %d, want 20260803", version)
	}
}

func TestLatestArchiveReportsMissingKind(t *testing.T) {
	dir := t.TempDir()
	if _, _, err := latestArchive(dir, "delta"); err == nil {
		t.Fatal("expected an error when delta archive is absent")
	}
}
