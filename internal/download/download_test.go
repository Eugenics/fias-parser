package download

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"gar_converter/internal/config"
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

func TestExtractZipExtractsOnlyImportedFilesAndVersion(t *testing.T) {
	importedFilePrefixes := []string{
		"AS_ADDR_OBJ_",
		"AS_ADDR_OBJ_PARAMS_",
		"AS_ADM_HIERARCHY_",
		"AS_HOUSE_TYPES_",
	}
	var archive bytes.Buffer
	zw := zip.NewWriter(&archive)
	files := map[string]string{
		"gar_xml/01/AS_ADDR_OBJ_20260831_abc.XML":          "addresses",
		"gar_xml/01/AS_ADDR_OBJ_PARAMS_20260831_abc.XML":   "params",
		"gar_xml/01/AS_ADM_HIERARCHY_20260831_abc.XML":     "hierarchy",
		"gar_xml/AS_HOUSE_TYPES_20260831_abc.XML":          "types",
		"gar_xml/version.txt":                              "20260831",
		"gar_xml/01/AS_ADDR_OBJ_DIVISION_20260831_abc.XML": "skip",
		"gar_xml/01/AS_HOUSES_20260831_abc.XML":            "skip",
		"gar_xml/readme.txt":                               "skip",
	}
	for name, contents := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(contents)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	zr, err := zip.NewReader(bytes.NewReader(archive.Bytes()), int64(archive.Len()))
	if err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()
	if err := extractZip(zr, dest, importedFilePrefixes); err != nil {
		t.Fatal(err)
	}

	for name, contents := range files {
		path := filepath.Join(dest, filepath.FromSlash(name))
		got, err := os.ReadFile(path)
		wantExtracted := contents != "skip"
		if wantExtracted && err != nil {
			t.Errorf("expected %s to be extracted: %v", name, err)
		}
		if !wantExtracted && !os.IsNotExist(err) {
			t.Errorf("expected %s to be skipped, read error = %v", name, err)
		}
		if wantExtracted && string(got) != contents {
			t.Errorf("contents of %s = %q, want %q", name, got, contents)
		}
	}
}

func TestExtractKeepsPreviousDirectoryWhenVersionIsInvalid(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "full")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, "previous.xml"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(root, "20260831_full.zip")
	writeTestArchive(t, archivePath, "20260830")

	d := New(config.FiasConfig{ImportedFilePrefixes: []string{"AS_ADDR_OBJ_"}})
	err := d.Extract(ExtractResult{VersionID: 20260831, ArchivePath: archivePath}, dest)
	if err == nil {
		t.Fatal("expected version mismatch error")
	}
	if data, readErr := os.ReadFile(filepath.Join(dest, "previous.xml")); readErr != nil || string(data) != "old" {
		t.Fatalf("previous directory was changed: data=%q err=%v", data, readErr)
	}
}

func TestExtractReplacesDirectoryAfterValidation(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "full")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, "previous.xml"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(root, "20260831_full.zip")
	writeTestArchive(t, archivePath, "20260831")

	d := New(config.FiasConfig{ImportedFilePrefixes: []string{"AS_ADDR_OBJ_"}})
	if err := d.Extract(ExtractResult{VersionID: 20260831, ArchivePath: archivePath}, dest); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, "previous.xml")); !os.IsNotExist(err) {
		t.Fatalf("previous file still exists: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "01", "AS_ADDR_OBJ_20260831_test.XML")); err != nil {
		t.Fatal(err)
	}
}

func writeTestArchive(t *testing.T, path, version string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(file)
	for name, contents := range map[string]string{
		"gar_xml/version.txt":                      version,
		"gar_xml/01/AS_ADDR_OBJ_20260831_test.XML": "<ROOT/>",
	} {
		writer, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write([]byte(contents)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}
