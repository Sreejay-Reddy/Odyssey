package config

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
	"github.com/sreejay-reddy/odyssey/odyssey-go/configutil"
)

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

func ReadYAML(path string) (configutil.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return configutil.Config{}, err
	}

	var config configutil.Config

	if err := yaml.Unmarshal(data, &config); err != nil {
		return configutil.Config{}, err
	}

	return config, nil
}