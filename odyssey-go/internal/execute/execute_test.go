package execute

import (
	"context"
	"errors"
	"testing"
	"reflect"

	"github.com/sreejay-reddy/odyssey/odyssey-go/configutil"
	"github.com/sreejay-reddy/odyssey/odyssey-go/internal/registry"
	"github.com/jackc/pgx/v5"
)

func setupExecuteTest(t *testing.T) *pgx.Conn {
	t.Helper()

	registry.Reset()

	conn := testConn(t)

	t.Cleanup(func() {
		registry.Reset()
	})

	return conn
}

func registerExecuteTarget(
	t *testing.T,
	target string,
	fn any,
) {
	t.Helper()

	cfg := configutil.Config{
		Services: map[string]string{},
		Registry: map[string]configutil.TargetConfig{
			target: {},
		},
	}

	err := registry.Register(
		cfg,
		target,
		fn,
		10_000,
	)

	if err != nil {
		t.Fatalf(
			"failed to register target: %v",
			err,
		)
	}
}

func insertInputLedger(
	t *testing.T,
	ctx context.Context,
	conn *pgx.Conn,
	key string,
	target string,
	input []byte,
) {
	t.Helper()

	_, err := conn.Exec(
		ctx,
		`INSERT INTO odyssey_ledger (
			key,
			target,
			sequence,
			mode,
			status,
			input
		)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		key,
		target,
		1,
		"local",
		"claimed",
		input,
	)

	if err != nil {
		t.Fatalf(
			"failed to insert ledger row: %v",
			err,
		)
	}
}

func TestExecuteWithInput(t *testing.T) {
	ctx := context.Background()

	conn := setupExecuteTest(t)

	type Input struct {
		Name string `json:"name"`
	}

	type Output struct {
		Message string `json:"message"`
	}

	key := "test-execute-input"
	target := "hello"

	insertInputLedger(
		t,
		ctx,
		conn,
		key,
		target,
		[]byte(`{"name":"Summer"}`),
	)

	registerExecuteTarget(
		t,
		target,
		func(
			ctx context.Context,
			input Input,
		) (Output, error) {
			return Output{
				Message: "Hello, " + input.Name + "!",
			}, nil
		},
	)

	result, acquired, err := Execute(
		ctx,
		conn,
		key,
		target,
	)

	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if !acquired {
		t.Fatal("expected execution to be acquired")
	}

	output, ok := result.(Output)

	if !ok {
		t.Fatalf(
			"expected Output, got %T",
			result,
		)
	}

	if output.Message != "Hello, Summer!" {
		t.Fatalf(
			"expected Hello, Summer!, got %s",
			output.Message,
		)
	}
}


func TestExecuteWithoutInput(t *testing.T) {
	ctx := context.Background()

	conn := setupExecuteTest(t)

	type Output struct {
		Message string `json:"message"`
	}

	key := "test-execute-no-input"
	target := "hello"

	insertClaimedLedger(
		t,
		ctx,
		conn,
		key,
		target,
	)

	registerExecuteTarget(
		t,
		target,
		func(
			ctx context.Context,
		) (Output, error) {
			return Output{
				Message: "hello",
			}, nil
		},
	)

	result, acquired, err := Execute(
		ctx,
		conn,
		key,
		target,
	)

	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if !acquired {
		t.Fatal("expected execution to be acquired")
	}

	output, ok := result.(Output)

	if !ok {
		t.Fatalf(
			"expected Output, got %T",
			result,
		)
	}

	if output.Message != "hello" {
		t.Fatalf(
			"expected hello, got %s",
			output.Message,
		)
	}
}


func TestExecuteRejectsInputForFunctionWithoutInput(t *testing.T) {
	ctx := context.Background()

	conn := setupExecuteTest(t)

	type Output struct {
		Message string `json:"message"`
	}

	key := "test-input-without-arg"
	target := "hello"

	insertInputLedger(
		t,
		ctx,
		conn,
		key,
		target,
		[]byte(`{"name":"Summer"}`),
	)

	registerExecuteTarget(
		t,
		target,
		func(
			ctx context.Context,
		) (Output, error) {
			return Output{
				Message: "hello",
			}, nil
		},
	)

	_, _, err := Execute(
		ctx,
		conn,
		key,
		target,
	)

	if err == nil {
		t.Fatal("expected Execute to return an error")
	}

	if err.Error() != "input exists but registered function does not accept input" {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}
}


func TestExecuteRejectsMissingInput(t *testing.T) {
	ctx := context.Background()

	conn := setupExecuteTest(t)

	type Input struct {
		Name string `json:"name"`
	}

	type Output struct {
		Message string `json:"message"`
	}

	key := "test-missing-input"
	target := "hello"

	insertClaimedLedger(
		t,
		ctx,
		conn,
		key,
		target,
	)

	registerExecuteTarget(
		t,
		target,
		func(
			ctx context.Context,
			input Input,
		) (Output, error) {
			return Output{
				Message: "hello",
			}, nil
		},
	)

	_, _, err := Execute(
		ctx,
		conn,
		key,
		target,
	)

	if err == nil {
		t.Fatal("expected Execute to return an error")
	}

	if err.Error() != "no input exists but registered function requires input" {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}
}


func TestExecuteFunctionFailureAbandons(t *testing.T) {
	ctx := context.Background()

	conn := setupExecuteTest(t)

	type Output struct {
		Message string `json:"message"`
	}

	key := "test-execute-failure"
	target := "failing"

	insertClaimedLedger(
		t,
		ctx,
		conn,
		key,
		target,
	)

	registerExecuteTarget(
		t,
		target,
		func(
			ctx context.Context,
		) (Output, error) {
			return Output{}, errors.New("boom")
		},
	)

	_, _, err := Execute(
		ctx,
		conn,
		key,
		target,
	)

	if err == nil {
		t.Fatal("expected Execute to return an error")
	}

	var expired bool

	err = conn.QueryRow(
		ctx,
		`SELECT expires_at <= NOW()
		FROM odyssey_journeys
		WHERE key = $1
		  AND target = $2`,
		key,
		target,
	).Scan(&expired)

	if err != nil {
		t.Fatalf(
			"failed to query journey: %v",
			err,
		)
	}

	if !expired {
		t.Fatal("expected failed execution to abandon journey")
	}
}


func TestDecodeInputInvalidJSON(t *testing.T) {
	type testInput struct {
		Name string `json:"name"`
	}

	_, err := decodeInput(
		[]byte(`not-json`),
		reflect.TypeOf(testInput{}),
	)

	if err == nil {
		t.Fatal("expected decodeInput to return an error")
	}
}

func TestFetchResponseCachedResult(t *testing.T) {
	ctx := context.Background()

	conn := testConn(t)

	key := "test-cached-result"
	target := "target-a"

	insertClaimedLedger(t, ctx, conn, key, target)
	insertCompletedJourney(t, ctx, conn, key, target)

	e := testExecution(t, conn, key, target)

	found, err := e.fetchResponse(ctx)

	if err != nil {
		t.Fatalf("fetchResponse returned error: %v", err)
	}

	if !found {
		t.Fatal("expected cached response to be found")
	}

	if e.metadata.status != "completed" {
		t.Fatalf(
			"expected completed status, got %q",
			e.metadata.status,
		)
	}

	if e.metadata.response == nil {
		t.Fatal("expected response")
	}
}
