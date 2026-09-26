package cli

import (
	"os"
	"path/filepath"

	"github.com/sreejay-reddy/odyssey/odyssey-go/configutil"
	"github.com/sreejay-reddy/odyssey/odyssey-go/internal/config"
)

func Init() (configutil.Config, configutil.State, error) {
	const dir = "odyssey"

	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return configutil.Config{}, configutil.State{}, err
	}

	err = config.CreateConfig(filepath.Join(dir, "odyssey.yaml"))
	if err != nil {
		return configutil.Config{}, configutil.State{}, err
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return configutil.Config{}, configutil.State{}, err
	}

	state, err := config.LoadState()
	if err != nil {
		return configutil.Config{}, configutil.State{}, err
	}

	return cfg, state, err
}