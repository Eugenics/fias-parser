package config

import (
	"fmt"
	"os"

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
	URL         string `yaml:"url"`
	AllInfoURL  string `yaml:"all_info_url"`
	LastInfoURL string `yaml:"last_info_url"`
	DeltaName   string `yaml:"delta_name"`
	FullName    string `yaml:"full_name"`
}

func (f FiasConfig) FullZipName() string {
	if f.FullName == "" {
		return "gar_xml.zip"
	}
	return f.FullName
}

func (f FiasConfig) DeltaZipName() string {
	if f.DeltaName == "" {
		return "gar_delta_xml.zip"
	}
	return f.DeltaName
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
