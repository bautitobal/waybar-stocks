package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Asset struct {
	Symbol string `yaml:"symbol"`
	Name   string `yaml:"name"`
	// optional timeframe for percent change (e.g. "1D", "3D", "1W", "1M", "1Y", "15m")
	Timeframe string `yaml:"timeframe,omitempty"`
}

type Colors struct {
	Up      string `yaml:"up"`
	Down    string `yaml:"down"`
	Neutral string `yaml:"neutral"`
}

type Config struct {
	RefreshInterval  int     `yaml:"refresh_interval"`
	RotationInterval int     `yaml:"rotation_interval"`
	Format           string  `yaml:"format"`
	Assets           []Asset `yaml:"assets"`
	Colors           Colors  `yaml:"colors"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
