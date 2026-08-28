package odyssey

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/sreejay-reddy/odyssey/odyssey-go/configutil"
	"github.com/sreejay-reddy/odyssey/odyssey-go/internal/registry"
)

func testClient(t *testing.T) *Client {
	t.Helper()

	dbURL := os.Getenv("DATABASE_URL")

	if dbURL == "" {
		t.Fatal("DATABASE_URL is not set")
	}

	return &Client{
		dbURL: dbURL,
		config: configutil.Config{
			Services: map[string]string{},
			Registry: map[string]configutil.TargetConfig{
				"payment": {},
			},
		},
	}
}

func testServerConn(t *testing.T) *pgx.Conn {
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

func cleanServerDatabase(t *testing.T, conn *pgx.Conn) {
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

func registerServerTarget(t *testing.T, cfg configutil.Config, target string) {
	t.Helper()

	registry.Register(
		cfg,
		target,
		func() (any, error) {
			return map[string]any{
				"result": "success",
			}, nil
		},
		10000,
	)
}

func executeRequest(
	t *testing.T,
	server *Server,
	body any,
) *httptest.ResponseRecorder {
	t.Helper()

	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/execute",
		bytes.NewReader(payload),
	)

	rec := httptest.NewRecorder()

	server.handleExecute(rec, req)

	return rec
}

func TestHealth(t *testing.T) {
	server := &Server{}

	req := httptest.NewRequest(
		http.MethodGet,
		"/health",
		nil,
	)

	rec := httptest.NewRecorder()

	server.handleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			rec.Code,
		)
	}

	var response map[string]string

	if err := json.Unmarshal(
		rec.Body.Bytes(),
		&response,
	); err != nil {
		t.Fatalf(
			"failed to decode response: %v",
			err,
		)
	}

	if response["status"] != "ok" {
		t.Fatalf(
			"expected status ok, got %q",
			response["status"],
		)
	}
}

func TestExecuteInvalidJSON(t *testing.T) {
	server := &Server{}

	req := httptest.NewRequest(
		http.MethodPost,
		"/execute",
		bytes.NewBufferString(`{invalid json`),
	)

	rec := httptest.NewRecorder()

	server.handleExecute(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}
}

func TestExecuteMissingKey(t *testing.T) {
	server := &Server{}

	body := map[string]string{
		"target": "payment",
	}

	rec := executeRequest(t, server, body)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}
}

func TestExecuteMissingTarget(t *testing.T) {
	server := &Server{}

	body := map[string]string{
		"key": "order-1",
	}

	rec := executeRequest(t, server, body)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}
}

func TestExecuteEmptyKey(t *testing.T) {
	server := &Server{}

	body := map[string]string{
		"key":    "",
		"target": "payment",
	}

	rec := executeRequest(t, server, body)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}
}

func TestExecuteEmptyTarget(t *testing.T) {
	server := &Server{}

	body := map[string]string{
		"key":    "order-1",
		"target": "",
	}

	rec := executeRequest(t, server, body)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}
}

func TestExecuteUnknownTarget(t *testing.T) {
	registry.Reset()

	client := testClient(t)

	server := New(client)

	conn := testServerConn(t)
	cleanServerDatabase(t, conn)

	body := map[string]string{
		"key":    "order-1",
		"target": "unknown",
	}

	rec := executeRequest(t, server, body)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			rec.Code,
		)
	}
}

func TestExecuteSuccessfulExecution(t *testing.T) {
	registry.Reset()

	client := testClient(t)

	type PaymentInput struct {
		Amount float64 `json:"amount"`
	}

	type PaymentResponse struct {
		Status string  `json:"status"`
		Amount float64 `json:"amount"`
	}

	err := client.Register(
		"payment",
		func(ctx context.Context, input PaymentInput) (PaymentResponse, error) {
			return PaymentResponse{
				Status: "paid",
				Amount: input.Amount,
			}, nil
		},
		10000,
	)
	if err != nil {
		t.Fatalf("failed to register payment: %v", err)
	}

	server := New(client)

	conn := testServerConn(t)
	cleanServerDatabase(t, conn)

	_, err = conn.Exec(
		context.Background(),
		`INSERT INTO odyssey_ledger (
			key,
			target,
			sequence,
			mode,
			input
		)
		VALUES ($1, $2, $3, $4, $5)`,
		"order-1",
		"payment",
		1,
		"local",
		[]byte(`{"amount":250}`),
	)

	if err != nil {
		t.Fatalf(
			"failed to create ledger row: %v",
			err,
		)
	}

	body := map[string]string{
		"key":    "order-1",
		"target": "payment",
	}

	rec := executeRequest(t, server, body)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d: %s",
			http.StatusOK,
			rec.Code,
			rec.Body.String(),
		)
	}

	var response map[string]any

	if err := json.Unmarshal(
		rec.Body.Bytes(),
		&response,
	); err != nil {
		t.Fatalf(
			"failed to decode response: %v",
			err,
		)
	}

	if response["status"] != "completed" {
		t.Fatalf(
			"expected completed status, got %v",
			response["status"],
		)
	}

	if response["key"] != "order-1" {
		t.Fatalf(
			"expected order-1, got %v",
			response["key"],
		)
	}

	if response["target"] != "payment" {
		t.Fatalf(
			"expected payment, got %v",
			response["target"],
		)
	}

	result, ok := response["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected result object, got %T", response["result"])
	}

	if result["status"] != "paid" {
		t.Fatalf(
			"expected result status paid, got %v",
			result["status"],
		)
	}

	if result["amount"] != float64(250) {
		t.Fatalf(
			"expected result amount 250, got %v",
			result["amount"],
		)
	}
}

func TestExecuteIncompleteReturnsConflict(t *testing.T) {
	registry.Reset()

	client := testClient(t)

	registerServerTarget(
		t,
		client.config,
		"payment",
	)

	server := New(client)

	conn := testServerConn(t)
	cleanServerDatabase(t, conn)

	ctx := context.Background()

	_, err := conn.Exec(
		ctx,
		`INSERT INTO odyssey_ledger (
			key,
			target,
			sequence,
			mode,
			status
		)
		VALUES ($1, $2, $3, $4, $5)`,
		"order-1",
		"payment",
		1,
		"local",
		"claimed",
	)

	if err != nil {
		t.Fatalf(
			"failed to create ledger row: %v",
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
			status
		)
		VALUES (
			$1,
			$2,
			$3,
			NOW() + INTERVAL '10 seconds',
			NOW(),
			nextval('odyssey_token_seq'),
			'claimed'
		)`,
		"order-1",
		"payment",
		"worker-1",
	)

	if err != nil {
		t.Fatalf(
			"failed to create active journey: %v",
			err,
		)
	}

	body := map[string]string{
		"key":    "order-1",
		"target": "payment",
	}

	rec := executeRequest(t, server, body)

	if rec.Code != http.StatusConflict {
		t.Fatalf(
			"expected status %d, got %d: %s",
			http.StatusConflict,
			rec.Code,
			rec.Body.String(),
		)
	}
}

func TestExecuteFunctionFailure(t *testing.T) {
	registry.Reset()

	client := testClient(t)

	err := client.Register(
		"payment",
		func(ctx context.Context) (map[string]any, error) {
			return nil, errors.New("payment failed")
		},
		10000,
	)

	if err != nil {
		t.Fatalf("failed to register payment: %v", err)
	}

	server := New(client)

	conn := testServerConn(t)
	cleanServerDatabase(t, conn)

	_, err = conn.Exec(
		context.Background(),
		`INSERT INTO odyssey_ledger (
			key,
			target,
			sequence,
			mode
		)
		VALUES ($1, $2, $3, $4)`,
		"order-1",
		"payment",
		1,
		"local",
	)

	if err != nil {
		t.Fatalf("failed to create ledger row: %v", err)
	}

	body := map[string]string{
		"key":    "order-1",
		"target": "payment",
	}

	rec := executeRequest(t, server, body)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d: %s",
			http.StatusInternalServerError,
			rec.Code,
			rec.Body.String(),
		)
	}
}

func TestExecuteFailureDoesNotCompleteLedger(t *testing.T) {
	registry.Reset()

	client := testClient(t)

	err := client.Register(
		"payment",
		func(ctx context.Context) (map[string]any, error) {
			return nil, errors.New("payment failed")
		},
		10000,
	)

	if err != nil {
		t.Fatalf("failed to register payment: %v", err)
	}

	server := New(client)

	conn := testServerConn(t)
	cleanServerDatabase(t, conn)

	_, err = conn.Exec(
		context.Background(),
		`INSERT INTO odyssey_ledger (
			key,
			target,
			sequence,
			mode
		)
		VALUES ($1, $2, $3, $4)`,
		"order-1",
		"payment",
		1,
		"local",
	)

	if err != nil {
		t.Fatalf("failed to create ledger row: %v", err)
	}

	body := map[string]string{
		"key":    "order-1",
		"target": "payment",
	}

	_ = executeRequest(t, server, body)

	var status string

	err = conn.QueryRow(
		context.Background(),
		`SELECT status
		 FROM odyssey_ledger
		 WHERE key = $1
		   AND target = $2`,
		"order-1",
		"payment",
	).Scan(&status)

	if err != nil {
		t.Fatalf("failed to query ledger: %v", err)
	}

	if status == "completed" {
		t.Fatal("failed execution must not complete ledger")
	}
}

