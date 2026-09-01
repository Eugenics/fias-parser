package config

import (
	"fmt"
	"os"
	"time"

	yaml "gopkg.in/yaml.v3"
)

type Config struct {
	Database DatabaseConfig `yaml:"database"`
	Importer ImporterConfig `yaml:"importer"`
	Fias     FiasConfig     `yaml:"fias"`
}

type DatabaseConfig struct {
	DSN string `yaml:"dsn"`
}

type ImporterConfig struct {
	BatchSize int `yaml:"batch_size"`
	Workers   int `yaml:"workers"`
}

type FiasConfig struct {
	ArchivesDir          string        `yaml:"archives_dir"`
	ImportedFilePrefixes []string      `yaml:"imported_file_prefixes"`
	ExpDelta             time.Duration `yaml:"-"`
	ExpDeltaText         string        `yaml:"exp_delta"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	cfg := &Config{
		Importer: ImporterConfig{BatchSize: 1000, Workers: 4},
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	cfg.Fias.ExpDelta = 7 * 24 * time.Hour
	if cfg.Fias.ExpDeltaText != "" {
		expDelta, err := time.ParseDuration(cfg.Fias.ExpDeltaText)
		if err != nil {
			return nil, fmt.Errorf("parse fias.exp_delta: %w", err)
		}
		cfg.Fias.ExpDelta = expDelta
	}

	if cfg.Database.DSN == "" {
		if dsn := os.Getenv("GAR_DATABASE_DSN"); dsn != "" {
			cfg.Database.DSN = dsn
		}
	}
	if cfg.Database.DSN == "" {
		return nil, fmt.Errorf("database.dsn is required")
	}
	return cfg, nil
}
