package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTestYAML(t *testing.T, content string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "odyssey.yaml")

	if err := os.WriteFile(
		path,
		[]byte(content),
		0600,
	); err != nil {
		t.Fatalf("failed to write test YAML: %v", err)
	}

	return path
}

func TestReadEnvironmentExists(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/odyssey")

	value, exists := ReadEnvironment()

	if !exists {
		t.Fatal("expected DATABASE_URL to exist")
	}

	if value != "postgres://localhost/odyssey" {
		t.Fatalf(
			"expected DATABASE_URL to be postgres://localhost/odyssey, got %q",
			value,
		)
	}
}

func TestReadEnvironmentMissing(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	os.Unsetenv("DATABASE_URL")

	value, exists := ReadEnvironment()

	if exists {
		t.Fatal("expected DATABASE_URL to not exist")
	}

	if value != "" {
		t.Fatalf(
			"expected empty value, got %q",
			value,
		)
	}
}

func TestReadYAML(t *testing.T) {
	path := writeTestYAML(t, `
services:
  payments: http://localhost:8001
  notifications: http://localhost:8002

registry:
  payment:
    retry:
      policy: exponential
      attempts: 3
      delay: 1s
    on_failure:
      notify: notifications
      wait_for_input: true

  email:
    retry:
      policy: fixed
      attempts: 2
      delay: 500ms
`)

	cfg, err := ReadYAML(path)

	if err != nil {
		t.Fatalf("ReadYAML returned error: %v", err)
	}

	if len(cfg.Services) != 2 {
		t.Fatalf(
			"expected 2 services, got %d",
			len(cfg.Services),
		)
	}

	if cfg.Services["payments"] != "http://localhost:8001" {
		t.Fatalf("unexpected payments service")
	}

	if cfg.Services["notifications"] != "http://localhost:8002" {
		t.Fatalf("unexpected notifications service")
	}

	if len(cfg.Registry) != 2 {
		t.Fatalf(
			"expected 2 registry targets, got %d",
			len(cfg.Registry),
		)
	}

	payment, exists := cfg.Registry["payment"]

	if !exists {
		t.Fatal("expected payment target")
	}

	if payment.Retry.Policy != "exponential" {
		t.Fatalf(
			"expected exponential policy, got %q",
			payment.Retry.Policy,
		)
	}

	if payment.Retry.Attempts != 3 {
		t.Fatalf(
			"expected 3 attempts, got %d",
			payment.Retry.Attempts,
		)
	}

	if payment.Retry.Delay != "1s" {
		t.Fatalf(
			"expected 1s delay, got %q",
			payment.Retry.Delay,
		)
	}

	if payment.OnFailure == nil {
		t.Fatal("expected on_failure configuration")
	}

	if payment.OnFailure.Notify != "notifications" {
		t.Fatalf(
			"expected notifications, got %q",
			payment.OnFailure.Notify,
		)
	}

	if !payment.OnFailure.WaitForInput {
		t.Fatal("expected wait_for_input to be true")
	}
}

func TestReadYAMLWithoutOptionalOnFailure(t *testing.T) {
	path := writeTestYAML(t, `
services:
  payments: http://localhost:8001

registry:
  payment:
    retry:
      policy: fixed
      attempts: 3
      delay: 1s
`)

	cfg, err := ReadYAML(path)

	if err != nil {
		t.Fatalf("ReadYAML returned error: %v", err)
	}

	target, exists := cfg.Registry["payment"]

	if !exists {
		t.Fatal("expected payment target")
	}

	if target.OnFailure != nil {
		t.Fatal("expected OnFailure to be nil")
	}
}

func TestReadYAMLEmptyConfig(t *testing.T) {
	path := writeTestYAML(t, ``)

	cfg, err := ReadYAML(path)

	if err != nil {
		t.Fatalf("ReadYAML returned error: %v", err)
	}

	if cfg.Services != nil && len(cfg.Services) != 0 {
		t.Fatalf("expected no services")
	}

	if cfg.Registry != nil && len(cfg.Registry) != 0 {
		t.Fatalf("expected no registry targets")
	}
}

func TestReadYAMLInvalidYAML(t *testing.T) {
	path := writeTestYAML(t, `
services:
  payments:
    - invalid
  broken: [
`)

	_, err := ReadYAML(path)

	if err == nil {
		t.Fatal("expected ReadYAML to return an error")
	}
}

func TestReadYAMLFileDoesNotExist(t *testing.T) {
	_, err := ReadYAML(
		filepath.Join(
			t.TempDir(),
			"does-not-exist.yaml",
		),
	)

	if err == nil {
		t.Fatal("expected ReadYAML to return an error")
	}
}

func TestFindYAML(t *testing.T) {
	oldDir, err := os.Getwd()

	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	testDir := t.TempDir()

	if err := os.Chdir(testDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	t.Cleanup(func() {
		os.Chdir(oldDir)
	})

	expected := filepath.Join(testDir, "odyssey.yaml")

	if err := os.WriteFile(
		expected,
		[]byte(`
services:
  payments: http://localhost:8001
`),
		0600,
	); err != nil {
		t.Fatalf("failed to create odyssey.yaml: %v", err)
	}

	path, err := FindYAML()

	if err != nil {
		t.Fatalf("FindYAML returned error: %v", err)
	}

	if path != expected {
		t.Fatalf(
			"expected %q, got %q",
			expected,
			path,
		)
	}
}

func TestFindYAMLNotFound(t *testing.T) {
	oldDir, err := os.Getwd()

	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	testDir := t.TempDir()

	if err := os.Chdir(testDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	t.Cleanup(func() {
		os.Chdir(oldDir)
	})

	_, err = FindYAML()

	if err == nil {
		t.Fatal("expected FindYAML to return an error")
	}

	if err.Error() != "odyssey.yaml not found" {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}
}