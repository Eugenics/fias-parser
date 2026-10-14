package main

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"gar_converter/internal/download"
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

func TestVersionArchivePathPreservesEventPath(t *testing.T) {
	result := download.ExtractResult{
		ArchivePath: "/archives/20261013_delta.zip",
		SourceArchivePath: "/app/data/downloads/20261013_delta.zip",
	}
	if got := versionArchivePath(result); got != result.SourceArchivePath {
		t.Fatalf("versionArchivePath() = %q, want %q", got, result.SourceArchivePath)
	}
}

func TestVersionArchivePathUsesLocalPathWithoutEvent(t *testing.T) {
	result := download.ExtractResult{ArchivePath: "/archives/20261013_delta.zip"}
	if got := versionArchivePath(result); got != result.ArchivePath {
		t.Fatalf("versionArchivePath() = %q, want %q", got, result.ArchivePath)
	}
}

func TestProcessDeltaArchivesSkipsImportedAndProcessesRemainingInOrder(t *testing.T) {
	archives := []download.ExtractResult{
		{VersionID: 20260901, ArchivePath: "20260901_delta.zip"},
		{VersionID: 20260904, ArchivePath: "20260904_delta.zip"},
		{VersionID: 20260908, ArchivePath: "20260908_delta.zip"},
	}
	imported := map[int64]bool{20260904: true}
	var processed []int64

	err := processDeltaArchives(
		context.Background(),
		archives,
		func(_ context.Context, versionID int64) (bool, error) {
			return imported[versionID], nil
		},
		func(_ context.Context, archive download.ExtractResult) error {
			processed = append(processed, archive.VersionID)
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	want := []int64{20260901, 20260908}
	if !reflect.DeepEqual(processed, want) {
		t.Fatalf("processed versions = %v, want %v", processed, want)
	}
}
