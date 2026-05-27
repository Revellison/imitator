package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Duration wraps time.Duration for YAML parsing
type Duration struct {
	time.Duration
}

// UnmarshalYAML implements yaml.Unmarshaler
func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration format: %w", err)
	}
	d.Duration = parsed
	return nil
}

// ServerConfig defines HTTP server parameters
type ServerConfig struct {
	Listen         string   `yaml:"listen"`
	ReadTimeout    Duration `yaml:"read_timeout"`
	WriteTimeout   Duration `yaml:"write_timeout"`
	MaxHeaderBytes int      `yaml:"max_header_bytes"`
}

// LoggingConfig defines logging parameters
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// VideoStreamConfig defines videostream module configuration
type VideoStreamConfig struct {
	Path         string            `yaml:"path"`
	ChunksDir    string            `yaml:"chunks_dir"`
	PreloadToRAM bool              `yaml:"preload_to_ram"`
	DeliveryMode string            `yaml:"delivery_mode"`
	SeqKey       string            `yaml:"seq_key"`
	FallbackMode string            `yaml:"fallback_mode"`
	Headers      map[string]string `yaml:"headers"`
}

type CacheWarmerConfig struct {
	EnabledWorker    bool     `yaml:"enabled_worker"`
	IntervalSeconds  int      `yaml:"interval_seconds"`
	ConcurrencyLimit int      `yaml:"concurrency_limit"`
	LocalProxy       string   `yaml:"local_proxy"`
	Targets          []string `yaml:"targets"`
}

type ModulesConfig struct {
	VideoStream VideoStreamConfig `yaml:"videostream"`
	CacheWarmer CacheWarmerConfig `yaml:"cache_warmer"`
}

type Config struct {
	Server         ServerConfig  `yaml:"server"`
	Logging        LoggingConfig `yaml:"logging"`
	EnabledModules []string      `yaml:"enabled_modules"`
	Modules        ModulesConfig `yaml:"modules"`
}

// LoadConfig parses the YAML configuration file and returns a Configuration object.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
