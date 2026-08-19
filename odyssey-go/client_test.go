package odyssey

import (
	"context"
	"os"
	"testing"

	"odyssey-go/internal/buildledger"
	"odyssey-go/internal/config"
	"odyssey-go/internal/registry"
)

func clientTestConfig() config.Config {
	return config.Config{
		Services: map[string]string{
			"payments": "http://localhost:9000",
		},
		Registry: map[string]config.TargetConfig{
			"payment": {},
			"order":   {},
		},
	}
}

func clientTest(t *testing.T) *Client {
	t.Helper()

	dbURL := os.Getenv("DATABASE_URL")

	if dbURL == "" {
		t.Fatal("DATABASE_URL is not set")
	}

	return NewClient(
		dbURL,
		clientTestConfig(),
	)
}

func cleanClientDatabase(t *testing.T, client *Client) {
	t.Helper()

	ctx := context.Background()

	conn, err := client.connect(ctx)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer conn.Close(ctx)

	_, err = conn.Exec(
		ctx,
		`TRUNCATE
			odyssey_deliveries,
			odyssey_ledger,
			odyssey_journeys
		RESTART IDENTITY CASCADE`,
	)

	if err != nil {
		t.Fatalf("failed to clean database: %v", err)
	}
}

func TestNewClient(t *testing.T) {
	cfg := clientTestConfig()

	client := NewClient(
		"postgres://example",
		cfg,
	)

	if client == nil {
		t.Fatal("expected client")
	}

	if client.dbURL != "postgres://example" {
		t.Fatalf(
			"expected dbURL %q, got %q",
			"postgres://example",
			client.dbURL,
		)
	}

	if len(client.config.Services) != 1 {
		t.Fatalf(
			"expected 1 service, got %d",
			len(client.config.Services),
		)
	}

	if len(client.config.Registry) != 2 {
		t.Fatalf(
			"expected 2 registry entries, got %d",
			len(client.config.Registry),
		)
	}
}

func TestClientInitDB(t *testing.T) {
	client := clientTest(t)

	if err := client.InitDB(context.Background()); err != nil {
		t.Fatalf(
			"InitDB failed: %v",
			err,
		)
	}
}

func TestClientInitDBInvalidConnection(t *testing.T) {
	client := NewClient(
		"postgres://invalid:invalid@localhost:1/invalid",
		clientTestConfig(),
	)

	err := client.InitDB(context.Background())

	if err == nil {
		t.Fatal("expected InitDB to fail")
	}
}

func TestClientRegister(t *testing.T) {
	registry.Reset()

	client := clientTest(t)

	fn := func() (any, error) {
		return "success", nil
	}

	err := client.Register(
		"payment",
		fn,
		10000,
	)

	if err != nil {
		t.Fatalf(
			"Register failed: %v",
			err,
		)
	}

	registered, exists := registry.Get("payment")

	if !exists {
		t.Fatal("expected payment to be registered")
	}

	if registered.Target != "payment" {
		t.Fatalf(
			"expected target payment, got %s",
			registered.Target,
		)
	}

	if registered.TTLMS != 10000 {
		t.Fatalf(
			"expected TTL 10000, got %d",
			registered.TTLMS,
		)
	}
}

func TestClientRegisterInvalidFunction(t *testing.T) {
	registry.Reset()

	client := clientTest(t)

	err := client.Register(
		"payment",
		"not-a-function",
		10000,
	)

	if err == nil {
		t.Fatal("expected Register to fail")
	}
}

func TestClientRegisterInvalidTTL(t *testing.T) {
	registry.Reset()

	client := clientTest(t)

	err := client.Register(
		"payment",
		func() (any, error) {
			return nil, nil
		},
		0,
	)

	if err == nil {
		t.Fatal("expected Register to fail")
	}
}

func TestClientRegisterDuplicateTarget(t *testing.T) {
	registry.Reset()

	client := clientTest(t)

	fn := func() (any, error) {
		return nil, nil
	}

	err := client.Register(
		"payment",
		fn,
		10000,
	)

	if err != nil {
		t.Fatalf(
			"first Register failed: %v",
			err,
		)
	}

	err = client.Register(
		"payment",
		fn,
		10000,
	)

	if err == nil {
		t.Fatal("expected duplicate Register to fail")
	}
}

func TestClientBuildLedger(t *testing.T) {
	registry.Reset()

	client := clientTest(t)

	ctx := context.Background()

	if err := client.InitDB(ctx); err != nil {
		t.Fatalf(
			"InitDB failed: %v",
			err,
		)
	}

	cleanClientDatabase(t, client)

	err := client.Register(
		"payment",
		func() (any, error) {
			return "paid", nil
		},
		10000,
	)

	if err != nil {
		t.Fatalf(
			"Register failed: %v",
			err,
		)
	}

	err = client.BuildLedger(
		ctx,
		"order-1",
		[]buildledger.Step{
			{
				Target: "payment",
				Input: map[string]any{
					"amount": 100,
				},
			},
		},
	)

	if err != nil {
		t.Fatalf(
			"BuildLedger failed: %v",
			err,
		)
	}

	conn, err := client.connect(ctx)
	if err != nil {
		t.Fatalf(
			"failed to connect: %v",
			err,
		)
	}
	defer conn.Close(ctx)

	var target string
	var mode string

	err = conn.QueryRow(
		ctx,
		`SELECT target, mode
		 FROM odyssey_ledger
		 WHERE key = $1`,
		"order-1",
	).Scan(
		&target,
		&mode,
	)

	if err != nil {
		t.Fatalf(
			"failed to query ledger: %v",
			err,
		)
	}

	if target != "payment" {
		t.Fatalf(
			"expected target payment, got %s",
			target,
		)
	}

	if mode != "local" {
		t.Fatalf(
			"expected local mode, got %s",
			mode,
		)
	}
}

func TestClientBuildLedgerDelegated(t *testing.T) {
	registry.Reset()

	client := clientTest(t)

	ctx := context.Background()

	if err := client.InitDB(ctx); err != nil {
		t.Fatalf(
			"InitDB failed: %v",
			err,
		)
	}

	cleanClientDatabase(t, client)

	err := client.BuildLedger(
		ctx,
		"order-2",
		[]buildledger.Step{
			{
				Target:   "payment",
				Delegate: "payments",
				Input: map[string]any{
					"amount": 250,
				},
			},
		},
	)

	if err != nil {
		t.Fatalf(
			"BuildLedger failed: %v",
			err,
		)
	}

	conn, err := client.connect(ctx)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close(ctx)

	var dbName string

	err = conn.QueryRow(
		ctx,
		`SELECT current_database()`,
	).Scan(&dbName)

	if err != nil {
		t.Fatalf("failed to determine database: %v", err)
	}

	t.Logf("test connection database: %s", dbName)
	defer conn.Close(ctx)

	var ledgerCount int

	err = conn.QueryRow(
		ctx,
		`SELECT COUNT(*)
		FROM odyssey_ledger
		WHERE key = $1`,
		"order-2",
	).Scan(&ledgerCount)

	if err != nil {
		t.Fatalf("failed to count ledger rows: %v", err)
	}

	t.Logf("ledger rows for order-2: %d", ledgerCount)

	var mode string

	err = conn.QueryRow(
		ctx,
		`SELECT mode
		FROM odyssey_ledger
		WHERE key = $1
		AND target = $2`,
		"order-2",
		"payment",
	).Scan(&mode)
	if err != nil {
		t.Fatalf(
			"failed to query ledger: %v",
			err,
		)
	}

	if mode != "delegated" {
		t.Fatalf(
			"expected delegated mode, got %s",
			mode,
		)
	}

	var delegate string

	err = conn.QueryRow(
		ctx,
		`SELECT emit_to
		 FROM odyssey_deliveries
		 WHERE key = $1
		   AND target = $2`,
		"order-2",
		"payment",
	).Scan(&delegate)

	if err != nil {
		t.Fatalf(
			"failed to query delivery: %v",
			err,
		)
	}

	if delegate != "payments" {
		t.Fatalf(
			"expected payments, got %s",
			delegate,
		)
	}
}

func TestClientBuildLedgerInvalidKey(t *testing.T) {
	registry.Reset()

	client := clientTest(t)

	err := client.BuildLedger(
		context.Background(),
		"",
		[]buildledger.Step{
			{
				Target: "payment",
			},
		},
	)

	if err == nil {
		t.Fatal("expected BuildLedger to fail")
	}
}

func TestClientBuildLedgerEmptySteps(t *testing.T) {
	registry.Reset()

	client := clientTest(t)

	err := client.BuildLedger(
		context.Background(),
		"order-1",
		nil,
	)

	if err == nil {
		t.Fatal("expected BuildLedger to fail")
	}
}

func TestClientBuildLedgerUnknownTarget(t *testing.T) {
	registry.Reset()

	client := clientTest(t)

	err := client.BuildLedger(
		context.Background(),
		"order-1",
		[]buildledger.Step{
			{
				Target: "unknown",
			},
		},
	)

	if err == nil {
		t.Fatal("expected BuildLedger to fail")
	}
}

func TestClientBuildLedgerUnknownDelegate(t *testing.T) {
	registry.Reset()

	client := clientTest(t)

	err := client.BuildLedger(
		context.Background(),
		"order-1",
		[]buildledger.Step{
			{
				Target:   "payment",
				Delegate: "unknown-service",
			},
		},
	)

	if err == nil {
		t.Fatal("expected BuildLedger to fail")
	}
}