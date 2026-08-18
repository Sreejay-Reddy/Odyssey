package config

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)


type Config struct {
    Services map[string]string `yaml:"services"`
    Registry map[string]TargetConfig `yaml:"registry"`
}

type TargetConfig struct {
    Retry     RetryConfig    `yaml:"retry"`
    OnFailure *FailureConfig `yaml:"on_failure,omitempty"`
}

type RetryConfig struct {
    Policy   string `yaml:"policy"`
    Attempts int    `yaml:"attempts,omitempty"`
    Delay    string `yaml:"delay"`
}

type FailureConfig struct {
    Notify        string `yaml:"notify"`
    WaitForInput  bool   `yaml:"wait_for_input"`
}

func FindYAML() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		path := filepath.Join(dir, "odyssey.yaml")

		if _, err := os.Stat(path); err == nil {
			return path, nil
		}

		parent := filepath.Dir(dir)

		if parent == dir {
			break
		}

		dir = parent
	}

	return "", errors.New("odyssey.yaml not found")
}

func ReadYAML(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var config Config

	if err := yaml.Unmarshal(data, &config); err != nil {
		return Config{}, err
	}

	return config, nil
}