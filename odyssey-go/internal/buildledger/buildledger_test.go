package buildledger

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/sreejay-reddy/odyssey/odyssey-go/configutil"
	"github.com/sreejay-reddy/odyssey/odyssey-go/internal/registry"
	"github.com/sreejay-reddy/odyssey/odyssey-go/types"

	"github.com/jackc/pgx/v5"
)

func testConn(t *testing.T) *pgx.Conn {
	t.Helper()

	dbURL := os.Getenv("DATABASE_URL")

	if dbURL == "" {
		t.Fatal("DATABASE_URL is not set")
	}

	conn, err := pgx.Connect(context.Background(), dbURL)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	t.Cleanup(func() {
		conn.Close(context.Background())
	})

	return conn
}

func cleanDatabase(t *testing.T, conn *pgx.Conn) {
	t.Helper()

	_, err := conn.Exec(
		context.Background(),
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

func testConfig() configutil.Config {
	return configutil.Config{
		Services: map[string]string{
			"payments": "http://localhost:8001",
			"email":    "http://localhost:8002",
		},
		Registry: map[string]configutil.TargetConfig{
			"payment": {},
			"email":   {},
		},
	}
}

func testRegisterTarget(t *testing.T, cfg configutil.Config, target string) {
	t.Helper()

	registry.Register(
		cfg,
		target,
		func() (any, error) {
			return nil, nil
		},
		10000,
	)
}

func TestBuildLedgerRejectsEmptyKey(t *testing.T) {
	conn := testConn(t)
	cfg := testConfig()

	ok, err := BuildLedger(
		context.Background(),
		conn,
		cfg,
		"",
		[]types.Step{
			{Target: "payment"},
		},
	)

	if ok {
		t.Fatal("expected success to be false")
	}

	if err == nil {
		t.Fatal("expected error")
	}

	if err.Error() != "key cannot be empty" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildLedgerRejectsEmptySteps(t *testing.T) {
	conn := testConn(t)
	cfg := testConfig()

	ok, err := BuildLedger(
		context.Background(),
		conn,
		cfg,
		"order-1",
		nil,
	)

	if ok {
		t.Fatal("expected success to be false")
	}

	if err == nil {
		t.Fatal("expected error")
	}

	if err.Error() != "build ledger requires at least one step" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildLedgerRejectsDuplicateTargets(t *testing.T) {
	conn := testConn(t)
	cfg := testConfig()

	cleanDatabase(t, conn)

	ok, err := BuildLedger(
		context.Background(),
		conn,
		cfg,
		"order-1",
		[]types.Step{
			{Target: "payment"},
			{Target: "payment"},
		},
	)

	if ok {
		t.Fatal("expected success to be false")
	}

	if err == nil {
		t.Fatal("expected error")
	}

	if err.Error() != "step targets must be unique" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildLedgerRejectsEmptyTarget(t *testing.T) {
	conn := testConn(t)
	cfg := testConfig()

	ok, err := BuildLedger(
		context.Background(),
		conn,
		cfg,
		"order-1",
		[]types.Step{
			{Target: "   "},
		},
	)

	if ok {
		t.Fatal("expected success to be false")
	}

	if err == nil {
		t.Fatal("expected error")
	}

	if err.Error() != "target cannot be empty" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildLedgerRejectsUnknownUndelegatedTarget(t *testing.T) {
	conn := testConn(t)
	cfg := testConfig()

	ok, err := BuildLedger(
		context.Background(),
		conn,
		cfg,
		"order-1",
		[]types.Step{
			{Target: "unknown"},
		},
	)

	if ok {
		t.Fatal("expected success to be false")
	}

	if err == nil {
		t.Fatal("expected error")
	}

	if err.Error() !=
		"target doesn't exist in registry and target isn't delegated" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildLedgerAllowsRegisteredTarget(t *testing.T) {
	conn := testConn(t)
	cfg := testConfig()

	cleanDatabase(t, conn)

	testRegisterTarget(t, cfg, "payment")

	ok, err := BuildLedger(
		context.Background(),
		conn,
		cfg,
		"order-1",
		[]types.Step{
			{Target: "payment"},
		},
	)

	if err != nil {
		t.Fatalf("BuildLedger returned error: %v", err)
	}

	if !ok {
		t.Fatal("expected success to be true")
	}
}

func TestBuildLedgerAllowsDelegatedTarget(t *testing.T) {
	conn := testConn(t)
	cfg := testConfig()

	cleanDatabase(t, conn)

	ok, err := BuildLedger(
		context.Background(),
		conn,
		cfg,
		"order-1",
		[]types.Step{
			{
				Target:   "payment",
				Delegate: "payments",
			},
		},
	)

	if err != nil {
		t.Fatalf("BuildLedger returned error: %v", err)
	}

	if !ok {
		t.Fatal("expected success to be true")
	}
}

func TestBuildLedgerRejectsUnknownDelegate(t *testing.T) {
	conn := testConn(t)
	cfg := testConfig()

	ok, err := BuildLedger(
		context.Background(),
		conn,
		cfg,
		"order-1",
		[]types.Step{
			{
				Target:   "payment",
				Delegate: "unknown-service",
			},
		},
	)

	if ok {
		t.Fatal("expected success to be false")
	}

	if err == nil {
		t.Fatal("expected error")
	}

	if err.Error() != "delegate does not exist in configured services" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildLedgerPersistsLocalStep(t *testing.T) {
	conn := testConn(t)
	cfg := testConfig()

	cleanDatabase(t, conn)

	testRegisterTarget(t, cfg, "payment")

	ok, err := BuildLedger(
		context.Background(),
		conn,
		cfg,
		"order-1",
		[]types.Step{
			{
				Target: "payment",
				Input: map[string]any{
					"amount": 100,
				},
			},
		},
	)

	if err != nil {
		t.Fatalf("BuildLedger returned error: %v", err)
	}

	if !ok {
		t.Fatal("expected success to be true")
	}

	var (
		key    string
		target string
		mode   string
		input  []byte
	)

	err = conn.QueryRow(
		context.Background(),
		`SELECT key, target, mode, input
		 FROM odyssey_ledger
		 WHERE key = $1 AND target = $2`,
		"order-1",
		"payment",
	).Scan(
		&key,
		&target,
		&mode,
		&input,
	)

	if err != nil {
		t.Fatalf("failed to query ledger: %v", err)
	}

	if key != "order-1" {
		t.Fatalf("expected key order-1, got %q", key)
	}

	if target != "payment" {
		t.Fatalf("expected target payment, got %q", target)
	}

	if mode != "local" {
		t.Fatalf("expected local mode, got %q", mode)
	}

	var decoded map[string]any

	if err := json.Unmarshal(input, &decoded); err != nil {
		t.Fatalf("failed to decode input: %v", err)
	}

	if decoded["amount"] != float64(100) {
		t.Fatalf(
			"expected amount 100, got %v",
			decoded["amount"],
		)
	}
}

func TestBuildLedgerPersistsDelegatedStep(t *testing.T) {
	conn := testConn(t)
	cfg := testConfig()

	cleanDatabase(t, conn)

	ok, err := BuildLedger(
		context.Background(),
		conn,
		cfg,
		"order-1",
		[]types.Step{
			{
				Target:   "payment",
				Delegate: "payments",
				Input: map[string]any{
					"amount": 500,
				},
			},
		},
	)

	if err != nil {
		t.Fatalf("BuildLedger returned error: %v", err)
	}

	if !ok {
		t.Fatal("expected success to be true")
	}

	var mode string

	err = conn.QueryRow(
		context.Background(),
		`SELECT mode
		 FROM odyssey_ledger
		 WHERE key = $1 AND target = $2`,
		"order-1",
		"payment",
	).Scan(&mode)

	if err != nil {
		t.Fatalf("failed to query ledger: %v", err)
	}

	if mode != "delegated" {
		t.Fatalf(
			"expected delegated mode, got %q",
			mode,
		)
	}

	var (
		key     string
		target  string
		emitTo  string
	)

	err = conn.QueryRow(
		context.Background(),
		`SELECT key, target, emit_to
		 FROM odyssey_deliveries
		 WHERE key = $1 AND target = $2`,
		"order-1",
		"payment",
	).Scan(
		&key,
		&target,
		&emitTo,
	)

	if err != nil {
		t.Fatalf("failed to query delivery: %v", err)
	}

	if key != "order-1" {
		t.Fatalf("expected key order-1, got %q", key)
	}

	if target != "payment" {
		t.Fatalf("expected target payment, got %q", target)
	}

	if emitTo != "payments" {
		t.Fatalf(
			"expected payments delegate, got %q",
			emitTo,
		)
	}
}

func TestBuildLedgerDoesNotCreateDeliveryForLocalStep(t *testing.T) {
	conn := testConn(t)
	cfg := testConfig()

	cleanDatabase(t, conn)

	testRegisterTarget(t, cfg, "payment")

	ok, err := BuildLedger(
		context.Background(),
		conn,
		cfg,
		"order-1",
		[]types.Step{
			{Target: "payment"},
		},
	)

	if err != nil {
		t.Fatalf("BuildLedger returned error: %v", err)
	}

	if !ok {
		t.Fatal("expected success to be true")
	}

	var count int

	err = conn.QueryRow(
		context.Background(),
		`SELECT COUNT(*)
		 FROM odyssey_deliveries
		 WHERE key = $1`,
		"order-1",
	).Scan(&count)

	if err != nil {
		t.Fatalf("failed to query deliveries: %v", err)
	}

	if count != 0 {
		t.Fatalf(
			"expected zero deliveries, got %d",
			count,
		)
	}
}

func TestBuildLedgerPersistsMultipleSteps(t *testing.T) {
	conn := testConn(t)
	cfg := testConfig()

	cleanDatabase(t, conn)

	testRegisterTarget(t, cfg, "payment")
	testRegisterTarget(t, cfg, "email")

	steps := []types.Step{
		{
			Target: "payment",
			Input: map[string]any{
				"amount": 100,
			},
		},
		{
			Target: "email",
			Input: map[string]any{
				"recipient": "test@example.com",
			},
		},
		{
			Target:   "inventory",
			Delegate: "payments",
			Input: map[string]any{
				"sku": "ABC",
			},
		},
	}

	ok, err := BuildLedger(
		context.Background(),
		conn,
		cfg,
		"order-1",
		steps,
	)

	if err != nil {
		t.Fatalf("BuildLedger returned error: %v", err)
	}

	if !ok {
		t.Fatal("expected success to be true")
	}

	var ledgerCount int

	err = conn.QueryRow(
		context.Background(),
		`SELECT COUNT(*)
		 FROM odyssey_ledger
		 WHERE key = $1`,
		"order-1",
	).Scan(&ledgerCount)

	if err != nil {
		t.Fatalf("failed to count ledger rows: %v", err)
	}

	if ledgerCount != 3 {
		t.Fatalf(
			"expected 3 ledger rows, got %d",
			ledgerCount,
		)
	}

	var deliveryCount int

	err = conn.QueryRow(
		context.Background(),
		`SELECT COUNT(*)
		 FROM odyssey_deliveries
		 WHERE key = $1`,
		"order-1",
	).Scan(&deliveryCount)

	if err != nil {
		t.Fatalf("failed to count delivery rows: %v", err)
	}

	if deliveryCount != 1 {
		t.Fatalf(
			"expected 1 delivery row, got %d",
			deliveryCount,
		)
	}
}

func TestBuildLedgerDuplicateIsAtomic(t *testing.T) {
	conn := testConn(t)
	cfg := testConfig()

	cleanDatabase(t, conn)

	_, err := BuildLedger(
		context.Background(),
		conn,
		cfg,
		"order-1",
		[]types.Step{
			{
				Target:   "payment",
				Delegate: "payments",
			},
		},
	)

	if err != nil {
		t.Fatalf("unexpected initial error: %v", err)
	}

	_, err = BuildLedger(
		context.Background(),
		conn,
		cfg,
		"order-1",
		[]types.Step{
			{
				Target:   "payment",
				Delegate: "payments",
			},
		},
	)

	if err == nil {
		t.Fatal("expected duplicate build to fail")
	}

	var ledgerCount int

	err = conn.QueryRow(
		context.Background(),
		`SELECT COUNT(*)
		 FROM odyssey_ledger
		 WHERE key = $1`,
		"order-1",
	).Scan(&ledgerCount)

	if err != nil {
		t.Fatalf("failed to query ledger: %v", err)
	}

	if ledgerCount != 1 {
		t.Fatalf(
			"expected exactly one ledger row, got %d",
			ledgerCount,
		)
	}
}

func TestBuildLedgerTransactionRollback(t *testing.T) {
	conn := testConn(t)
	cfg := testConfig()

	cleanDatabase(t, conn)

	// Force a database error by using a duplicate key if the schema
	// has a uniqueness constraint on (key, target).
	_, err := BuildLedger(
		context.Background(),
		conn,
		cfg,
		"order-1",
		[]types.Step{
			{
				Target:   "payment",
				Delegate: "payments",
			},
		},
	)

	if err != nil {
		t.Fatalf("first BuildLedger failed: %v", err)
	}

	_, err = BuildLedger(
		context.Background(),
		conn,
		cfg,
		"order-1",
		[]types.Step{
			{
				Target:   "payment",
				Delegate: "payments",
			},
			{
				Target: "email",
				Delegate: "payments",
			},
		},
	)

	if err == nil {
		t.Fatal("expected second BuildLedger to fail")
	}

	var count int

	err = conn.QueryRow(
		context.Background(),
		`SELECT COUNT(*)
		 FROM odyssey_ledger
		 WHERE key = $1`,
		"order-1",
	).Scan(&count)

	if err != nil {
		t.Fatalf("failed to query ledger: %v", err)
	}

	if count != 1 {
		t.Fatalf(
			"expected transaction rollback to preserve one existing row, got %d",
			count,
		)
	}
}

func TestBuildLedgerContextCancellation(t *testing.T) {
	conn := testConn(t)
	cfg := testConfig()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := BuildLedger(
		ctx,
		conn,
		cfg,
		"order-1",
		[]types.Step{
			{
				Target: "payment",
			},
		},
	)

	if err == nil {
		t.Fatal("expected context cancellation error")
	}

	if !errors.Is(err, context.Canceled) {
		t.Fatalf(
			"expected context.Canceled, got %v",
			err,
		)
	}
}

func TestBuildLedgerPersistsSequence(t *testing.T) {
	conn := testConn(t)
	cfg := testConfig()

	cleanDatabase(t, conn)

	testRegisterTarget(t, cfg, "payment")
	testRegisterTarget(t, cfg, "email")

	ok, err := BuildLedger(
		context.Background(),
		conn,
		cfg,
		"order-sequence",
		[]types.Step{
			{Target: "payment"},
			{Target: "email"},
		},
	)

	if err != nil {
		t.Fatalf("BuildLedger returned error: %v", err)
	}

	if !ok {
		t.Fatal("expected success to be true")
	}

	var (
		paymentSequence int64
		emailSequence   int64
	)

	err = conn.QueryRow(
		context.Background(),
		`SELECT sequence
		 FROM odyssey_ledger
		 WHERE key = $1
		   AND target = $2`,
		"order-sequence",
		"payment",
	).Scan(&paymentSequence)

	if err != nil {
		t.Fatalf("failed to query payment sequence: %v", err)
	}

	err = conn.QueryRow(
		context.Background(),
		`SELECT sequence
		 FROM odyssey_ledger
		 WHERE key = $1
		   AND target = $2`,
		"order-sequence",
		"email",
	).Scan(&emailSequence)

	if err != nil {
		t.Fatalf("failed to query email sequence: %v", err)
	}

	if paymentSequence != 1 {
		t.Fatalf(
			"expected payment sequence 1, got %d",
			paymentSequence,
		)
	}

	if emailSequence != 2 {
		t.Fatalf(
			"expected email sequence 2, got %d",
			emailSequence,
		)
	}
}

func TestBuildLedgerPersistsInput(t *testing.T) {
	conn := testConn(t)
	cfg := testConfig()

	cleanDatabase(t, conn)

	testRegisterTarget(t, cfg, "payment")

	ok, err := BuildLedger(
		context.Background(),
		conn,
		cfg,
		"order-input",
		[]types.Step{
			{
				Target: "payment",
				Input: map[string]any{
					"amount":   250,
					"currency": "USD",
				},
			},
		},
	)

	if err != nil {
		t.Fatalf("BuildLedger returned error: %v", err)
	}

	if !ok {
		t.Fatal("expected success to be true")
	}

	var input []byte

	err = conn.QueryRow(
		context.Background(),
		`SELECT input
		 FROM odyssey_ledger
		 WHERE key = $1
		   AND target = $2`,
		"order-input",
		"payment",
	).Scan(&input)

	if err != nil {
		t.Fatalf("failed to query ledger input: %v", err)
	}

	var decoded map[string]any

	if err := json.Unmarshal(input, &decoded); err != nil {
		t.Fatalf("failed to decode input: %v", err)
	}

	if decoded["amount"] != float64(250) {
		t.Fatalf(
			"expected amount 250, got %v",
			decoded["amount"],
		)
	}

	if decoded["currency"] != "USD" {
		t.Fatalf(
			"expected currency USD, got %v",
			decoded["currency"],
		)
	}
}