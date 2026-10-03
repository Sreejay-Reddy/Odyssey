package config

import "time"

type Config struct {
	Services map[string]string       `yaml:"services"`
	Registry map[string]TargetConfig `yaml:"registry"`
	Agent    AgentConfig             `yaml:"agent"`
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
	Notify       string `yaml:"notify"`
	WaitForInput bool   `yaml:"wait_for_input"`
}

type AgentConfig struct {
	SDK      SDKConfig      `yaml:"sdk"`
	Postgres PostgresConfig `yaml:"postgres"`
	WorkPool WorkPool       `yaml:"workpool"`
}

type SDKConfig struct {
	Workers   int `yaml:"workers"`
	BatchSize int `yaml:"batchsize"`
}

type WorkPool struct {
	Maxsize  int `yaml:"maxsize"`
	Interval time.Duration `yaml:"interval"`
}

type PostgresConfig struct {
	PoolSize int32 `yaml:"pool_size"`
}

