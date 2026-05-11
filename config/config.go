package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type CameraConfig struct {
	URL         string `yaml:"url"`
	Interval    int    `yaml:"interval"`
	Timeout     int    `yaml:"timeout"`
	Screenshots bool   `yaml:"screenshots"`
}

type Settings struct {
	MetricPrefix  string `yaml:"metric_prefix"`
	LogLevel      string `yaml:"log_level"`
	ScreenshotDir string `yaml:"screenshots_dir"`
	Port          int    `yaml:"port"`
}

type Config struct {
	Cameras  map[string]*CameraConfig `yaml:"cameras"`
	Settings Settings                 `yaml:"settings"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	applyDefaults(cfg)

	if err := validate(cfg); err != nil {
		return nil, fmt.Errorf("config validation: %w", err)
	}

	return cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.Settings.MetricPrefix == "" {
		cfg.Settings.MetricPrefix = "rtsp"
	}
	if cfg.Settings.LogLevel == "" {
		cfg.Settings.LogLevel = "error"
	}
	if cfg.Settings.Port == 0 {
		cfg.Settings.Port = 2112
	}

	for _, cam := range cfg.Cameras {
		if cam.Interval == 0 {
			cam.Interval = 60
		}
		if cam.Timeout == 0 {
			cam.Timeout = 30
		}
	}
}

func validate(cfg *Config) error {
	if len(cfg.Cameras) == 0 {
		return fmt.Errorf("no cameras defined")
	}

	screenshotNeeded := false
	for name, cam := range cfg.Cameras {
		if cam.URL == "" {
			return fmt.Errorf("camera %q: url is required", name)
		}
		if !strings.HasPrefix(cam.URL, "rtsp://") && !strings.HasPrefix(cam.URL, "rtsps://") &&
			!strings.HasPrefix(cam.URL, "http://") && !strings.HasPrefix(cam.URL, "https://") {
			return fmt.Errorf("camera %q: url must start with rtsp://, rtsps://, http://, or https://", name)
		}
		if cam.Screenshots {
			screenshotNeeded = true
		}
	}

	if screenshotNeeded && cfg.Settings.ScreenshotDir == "" {
		return fmt.Errorf("screenshots_dir must be set when any camera has screenshots enabled")
	}

	return nil
}
