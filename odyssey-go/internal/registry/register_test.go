package registry

import (
	"testing"

	"odyssey-go/internal/config"
)

func testConfig(targets ...string) config.Config {
	registryConfig := make(map[string]config.TargetConfig)

	for _, target := range targets {
		registryConfig[target] = config.TargetConfig{}
	}

	return config.Config{
		Registry: registryConfig,
	}
}

func testFn(ctx any) error {
	return nil
}

func TestRegister(t *testing.T) {
	registry = make(map[string]Registered)

	cfg := testConfig("payment")

	fn := func() {}

	err := Register(cfg, "payment", fn, 10000)

	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	registered, exists := Get("payment")

	if !exists {
		t.Fatal("expected target to be registered")
	}

	if registered.Target != "payment" {
		t.Fatalf(
			"expected target payment, got %q",
			registered.Target,
		)
	}

	if registered.Fn == nil {
		t.Fatal("expected function to be stored")
	}

	if registered.TTLMS != 10000 {
		t.Fatalf(
			"expected TTLMS 10000, got %d",
			registered.TTLMS,
		)
	}

	if registered.FunctionName == "" {
		t.Fatal("expected function name to be populated")
	}
}

func TestRegisterRejectsNonFunction(t *testing.T) {
	registry = make(map[string]Registered)

	cfg := testConfig("payment")

	err := Register(
		cfg,
		"payment",
		"not a function",
		10000,
	)

	if err == nil {
		t.Fatal("expected Register to return an error")
	}

	if err.Error() != "fn must be a function" {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}
}

func TestRegisterRejectsNilFunction(t *testing.T) {
	registry = make(map[string]Registered)

	cfg := testConfig("payment")

	err := Register(
		cfg,
		"payment",
		nil,
		10000,
	)

	if err == nil {
		t.Fatal("expected Register to return an error")
	}

	if err.Error() != "fn must be a function" {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}
}

func TestRegisterRejectsZeroTTL(t *testing.T) {
	registry = make(map[string]Registered)

	cfg := testConfig("payment")

	err := Register(
		cfg,
		"payment",
		func() {},
		0,
	)

	if err == nil {
		t.Fatal("expected Register to return an error")
	}

	if err.Error() != "ttlMS must be greater than zero" {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}
}

func TestRegisterRejectsNegativeTTL(t *testing.T) {
	registry = make(map[string]Registered)

	cfg := testConfig("payment")

	err := Register(
		cfg,
		"payment",
		func() {},
		-1,
	)

	if err == nil {
		t.Fatal("expected Register to return an error")
	}
}

func TestRegisterRejectsDuplicateTarget(t *testing.T) {
	registry = make(map[string]Registered)

	cfg := testConfig("payment")

	err := Register(
		cfg,
		"payment",
		func() {},
		10000,
	)

	if err != nil {
		t.Fatalf("first registration failed: %v", err)
	}

	err = Register(
		cfg,
		"payment",
		func() {},
		10000,
	)

	if err == nil {
		t.Fatal("expected duplicate registration to fail")
	}

	if err.Error() != "target already registered" {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}
}

func TestRegisterTargetDefinedInConfig(t *testing.T) {
	registry = make(map[string]Registered)

	cfg := testConfig("payment")

	err := Register(
		cfg,
		"payment",
		func() {},
		10000,
	)

	if err != nil {
		t.Fatalf(
			"expected configured target to register: %v",
			err,
		)
	}
}

func TestRegisterTargetNotDefinedWithoutDefault(t *testing.T) {
	registry = make(map[string]Registered)

	cfg := config.Config{
		Registry: map[string]config.TargetConfig{},
	}

	err := Register(
		cfg,
		"payment",
		func() {},
		10000,
	)

	if err == nil {
		t.Fatal("expected registration to fail")
	}

	if err.Error() !=
		"target does not exist in odyssey.yaml and no default is defined" {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}
}

func TestRegisterAllowsTargetWithDefault(t *testing.T) {
	registry = make(map[string]Registered)

	cfg := config.Config{
		Registry: map[string]config.TargetConfig{
			"default": {},
		},
	}

	err := Register(
		cfg,
		"payment",
		func() {},
		10000,
	)

	if err != nil {
		t.Fatalf(
			"expected default configuration to allow target: %v",
			err,
		)
	}
}

func TestGetRegisteredTarget(t *testing.T) {
	registry = make(map[string]Registered)

	cfg := testConfig("payment")

	fn := func() {}

	err := Register(
		cfg,
		"payment",
		fn,
		5000,
	)

	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	registered, exists := Get("payment")

	if !exists {
		t.Fatal("expected target to exist")
	}

	if registered.Target != "payment" {
		t.Fatalf(
			"expected payment, got %q",
			registered.Target,
		)
	}

	if registered.TTLMS != 5000 {
		t.Fatalf(
			"expected TTLMS 5000, got %d",
			registered.TTLMS,
		)
	}
}

func TestGetUnknownTarget(t *testing.T) {
	registry = make(map[string]Registered)

	_, exists := Get("unknown")

	if exists {
		t.Fatal("expected unknown target to not exist")
	}
}

func TestRegisterMultipleTargets(t *testing.T) {
	registry = make(map[string]Registered)

	cfg := testConfig(
		"payment",
		"email",
		"inventory",
	)

	targets := []string{
		"payment",
		"email",
		"inventory",
	}

	for _, target := range targets {
		err := Register(
			cfg,
			target,
			func() {},
			10000,
		)

		if err != nil {
			t.Fatalf(
				"failed to register %q: %v",
				target,
				err,
			)
		}
	}

	for _, target := range targets {
		if _, exists := Get(target); !exists {
			t.Fatalf(
				"expected target %q to be registered",
				target,
			)
		}
	}
}