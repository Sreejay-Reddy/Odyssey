package config

import (
	"os"
	"encoding/json"

	"gopkg.in/yaml.v3"
	"github.com/google/uuid"
	"github.com/sreejay-reddy/odyssey/odyssey-go/configutil"
)

const defaultConfig = 
`
version : 1

agent:
    sdk:
        workers: 4
        batchsize: 64
    postgres:
        pool_size: 15

registry:
  default:
    retry:
      policy: forever
      delay: 2s

    on_failure:
      notify: slack
      wait_for_input: true
`

func CreateConfig(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	return os.WriteFile(
		path,
		[]byte(defaultConfig),
		0644,
	)
}

func LoadConfig() (configutil.Config, error) {
	data, err := os.ReadFile("odyssey/odyssey.yaml")
	if err != nil {
		return configutil.Config{}, err
	}

	var config configutil.Config

	if err := yaml.Unmarshal(data, &config); err != nil {
		return configutil.Config{}, err
	}

	return config, nil
}

func LoadState() (configutil.State, error) {
    state := configutil.State{}

    sdkPath := "odyssey/sdk.json"

    data, err := os.ReadFile(sdkPath)
    if os.IsNotExist(err) {
        state.SDKID = uuid.New()

        data, err := json.Marshal(state.SDKID)
        if err != nil {
            return configutil.State{}, err
        }

        if err := os.WriteFile(sdkPath, data, 0644); err != nil {
            return configutil.State{}, err
        }
    } else if err != nil {
        return configutil.State{}, err
    } else {
        if err := json.Unmarshal(data, &state.SDKID); err != nil {
            return configutil.State{}, err
        }
    }

    state.SessionID = uuid.New()

    return state, nil
}