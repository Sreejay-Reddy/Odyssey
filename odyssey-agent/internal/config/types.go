package config

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