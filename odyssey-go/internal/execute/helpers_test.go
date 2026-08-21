package execute

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

func helperTestConn(t *testing.T) *pgx.Conn {
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

func cleanHelperDatabase(t *testing.T, conn *pgx.Conn) {
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

func TestDecodeInput(t *testing.T) {
	type Input struct {
		Name   string `json:"name"`
		Amount int    `json:"amount"`
	}

	input := []byte(`{
		"name": "payment",
		"amount": 100
	}`)

	value, err := decodeInput(
		input,
		reflect.TypeOf(Input{}),
	)

	if err != nil {
		t.Fatalf(
			"decodeInput failed: %v",
			err,
		)
	}

	result, ok := value.Interface().(Input)

	if !ok {
		t.Fatalf(
			"expected Input, got %T",
			value.Interface(),
		)
	}

	if result.Name != "payment" {
		t.Fatalf(
			"expected name payment, got %s",
			result.Name,
		)
	}

	if result.Amount != 100 {
		t.Fatalf(
			"expected amount 100, got %d",
			result.Amount,
		)
	}
}

func TestDecodeInputMalformedJSON(t *testing.T) {
	type Input struct {
		Name string `json:"name"`
	}

	_, err := decodeInput(
		[]byte(`{invalid json`),
		reflect.TypeOf(Input{}),
	)

	if err == nil {
		t.Fatal("expected decodeInput to fail")
	}
}

func TestDecodeInputPrimitive(t *testing.T) {
	value, err := decodeInput(
		[]byte(`"hello"`),
		reflect.TypeOf(""),
	)

	if err != nil {
		t.Fatalf(
			"decodeInput failed: %v",
			err,
		)
	}

	result, ok := value.Interface().(string)

	if !ok {
		t.Fatalf(
			"expected string, got %T",
			value.Interface(),
		)
	}

	if result != "hello" {
		t.Fatalf(
			"expected hello, got %s",
			result,
		)
	}
}

func TestDecodeInputArray(t *testing.T) {
	value, err := decodeInput(
		[]byte(`[1,2,3]`),
		reflect.TypeOf([]int{}),
	)

	if err != nil {
		t.Fatalf(
			"decodeInput failed: %v",
			err,
		)
	}

	result, ok := value.Interface().([]int)

	if !ok {
		t.Fatalf(
			"expected []int, got %T",
			value.Interface(),
		)
	}

	if len(result) != 3 {
		t.Fatalf(
			"expected 3 values, got %d",
			len(result),
		)
	}
}

func TestGetOwnerID(t *testing.T) {
	ownerID := getOwnerID()

	if ownerID == "" {
		t.Fatal("expected owner ID")
	}

	if !strings.Contains(
		ownerID,
		"-",
	) {
		t.Fatalf(
			"expected owner ID to contain separator, got %q",
			ownerID,
		)
	}
}

func TestFetchInput(t *testing.T) {
	conn := helperTestConn(t)
	cleanHelperDatabase(t, conn)

	ctx := context.Background()

	_, err := conn.Exec(
		ctx,
		`INSERT INTO odyssey_ledger (
			key,
			target,
			mode,
			input
		)
		VALUES ($1, $2, $3, $4)`,
		"input-key",
		"payment",
		"local",
		[]byte(`{"amount":100}`),
	)

	if err != nil {
		t.Fatalf(
			"failed to insert ledger row: %v",
			err,
		)
	}

	e := execution{
		key:    "input-key",
		target: "payment",
		conn:   conn,
	}

	found, err := e.fetchInput(ctx)

	if err != nil {
		t.Fatalf(
			"fetchInput failed: %v",
			err,
		)
	}

	if !found {
		t.Fatal("expected input to be found")
	}

	var input map[string]any

	if err := json.Unmarshal(
		e.input,
		&input,
	); err != nil {
		t.Fatalf(
			"failed to decode fetched input: %v",
			err,
		)
	}

	if input["amount"] != float64(100) {
		t.Fatalf(
			"expected amount 100, got %v",
			input["amount"],
		)
	}
}

func TestFetchInputMissing(t *testing.T) {
	conn := helperTestConn(t)
	cleanHelperDatabase(t, conn)

	e := execution{
		key:    "missing-key",
		target: "payment",
		conn:   conn,
	}

	found, err := e.fetchInput(
		context.Background(),
	)

	if err != nil {
		t.Fatalf(
			"fetchInput failed: %v",
			err,
		)
	}

	if found {
		t.Fatal("expected input not to be found")
	}
}

func TestFetchResponse(t *testing.T) {
	conn := helperTestConn(t)
	cleanHelperDatabase(t, conn)

	ctx := context.Background()

	_, err := conn.Exec(
		ctx,
		`INSERT INTO odyssey_ledger (
			key,
			target,
			mode,
			status
		)
		VALUES ($1, $2, $3, $4)`,
		"response-key",
		"payment",
		"local",
		"completed",
	)

	if err != nil {
		t.Fatalf(
			"failed to insert ledger row: %v",
			err,
		)
	}

	_, err = conn.Exec(
		ctx,
		`INSERT INTO odyssey_journeys (
			key,
			target,
			owner_id,
			expires_at,
			updated_at,
			fencing_token,
			status,
			execution_result
		)
		VALUES (
			$1,
			$2,
			$3,
			NOW() + INTERVAL '10 seconds',
			NOW(),
			nextval('odyssey_token_seq'),
			'completed',
			$4
		)`,
		"response-key",
		"payment",
		"owner-1",
		[]byte(`{"status":"paid"}`),
	)

	if err != nil {
		t.Fatalf(
			"failed to insert journey: %v",
			err,
		)
	}

	e := execution{
		key:    "response-key",
		target: "payment",
		conn:   conn,
	}

	found, err := e.fetchResponse(ctx)

	if err != nil {
		t.Fatalf(
			"fetchResponse failed: %v",
			err,
		)
	}

	if !found {
		t.Fatal("expected response to be found")
	}

	if e.metadata.status != "completed" {
		t.Fatalf(
			"expected completed status, got %s",
			e.metadata.status,
		)
	}
}

func TestFetchResponseMissing(t *testing.T) {
	conn := helperTestConn(t)
	cleanHelperDatabase(t, conn)

	e := execution{
		key:    "missing-response",
		target: "payment",
		conn:   conn,
	}

	found, err := e.fetchResponse(
		context.Background(),
	)

	if err != nil {
		t.Fatalf(
			"fetchResponse failed: %v",
			err,
		)
	}

	if found {
		t.Fatal("expected response not to be found")
	}
}

func TestFetchResponseIgnoresNonCompletedJourney(t *testing.T) {
	conn := helperTestConn(t)
	cleanHelperDatabase(t, conn)

	ctx := context.Background()

	_, err := conn.Exec(
		ctx,
		`INSERT INTO odyssey_ledger (
			key,
			target,
			mode,
			status
		)
		VALUES ($1, $2, $3, $4)`,
		"executing-key",
		"payment",
		"local",
		"executing",
	)

	if err != nil {
		t.Fatalf(
			"failed to insert ledger row: %v",
			err,
		)
	}

	_, err = conn.Exec(
		ctx,
		`INSERT INTO odyssey_journeys (
			key,
			target,
			owner_id,
			expires_at,
			updated_at,
			fencing_token,
			status,
			execution_result
		)
		VALUES (
			$1,
			$2,
			$3,
			NOW() + INTERVAL '10 seconds',
			NOW(),
			nextval('odyssey_token_seq'),
			'executing',
			$4
		)`,
		"executing-key",
		"payment",
		"owner-1",
		[]byte(`{"status":"running"}`),
	)

	if err != nil {
		t.Fatalf(
			"failed to insert journey: %v",
			err,
		)
	}

	e := execution{
		key:    "executing-key",
		target: "payment",
		conn:   conn,
	}

	found, err := e.fetchResponse(ctx)

	if err != nil {
		t.Fatalf(
			"fetchResponse failed: %v",
			err,
		)
	}

	if found {
		t.Fatal(
			"expected executing journey to have no fetched response",
		)
	}
}

func TestFetchInputPreservesJSON(t *testing.T) {
	conn := helperTestConn(t)
	cleanHelperDatabase(t, conn)

	ctx := context.Background()

	input := []byte(`{
		"user": "alice",
		"amount": 250,
		"items": ["a", "b"]
	}`)

	_, err := conn.Exec(
		ctx,
		`INSERT INTO odyssey_ledger (
			key,
			target,
			mode,
			input
		)
		VALUES ($1, $2, $3, $4)`,
		"json-key",
		"payment",
		"local",
		input,
	)

	if err != nil {
		t.Fatalf(
			"failed to insert ledger row: %v",
			err,
		)
	}

	e := execution{
		key:    "json-key",
		target: "payment",
		conn:   conn,
	}

	found, err := e.fetchInput(ctx)

	if err != nil {
		t.Fatalf(
			"fetchInput failed: %v",
			err,
		)
	}

	if !found {
		t.Fatal("expected input")
	}

	var expected any
	var actual any

	if err := json.Unmarshal(input, &expected); err != nil {
		t.Fatalf(
			"failed to decode expected input: %v",
			err,
		)
	}

	if err := json.Unmarshal(e.input, &actual); err != nil {
		t.Fatalf(
			"failed to decode actual input: %v",
			err,
		)
	}

	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf(
			"input mismatch: expected %v, got %v",
			expected,
			actual,
		)
	}
}