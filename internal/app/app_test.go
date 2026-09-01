package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"gar_converter/internal/config"
)

func TestRunReturnsXMLParseError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "broken.XML"), []byte("<ROOT><OBJECT"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{Importer: config.ImporterConfig{BatchSize: 10, Workers: 1}}
	if err := Run(context.Background(), dir, cfg, nil); err == nil {
		t.Fatal("expected malformed XML error")
	}
}
