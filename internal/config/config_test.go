package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadFiasExpDelta(t *testing.T) {
	for name, tc := range map[string]struct {
		yamlText string
		want     time.Duration
	}{
		"default": {"database:\n  dsn: postgres://test\n", 7 * 24 * time.Hour},
		"custom":  {"database:\n  dsn: postgres://test\nfias:\n  exp_delta: 48h\n", 48 * time.Hour},
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			if err := os.WriteFile(path, []byte(tc.yamlText), 0o644); err != nil {
				t.Fatal(err)
			}
			cfg, err := Load(path)
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Fias.ExpDelta != tc.want {
				t.Fatalf("ExpDelta = %s, want %s", cfg.Fias.ExpDelta, tc.want)
			}
		})
	}
}
