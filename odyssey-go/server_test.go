package odyssey

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"

	"odyssey-go/internal/config"
	"odyssey-go/internal/registry"
)

func testClient(t *testing.T) *Client {
	t.Helper()

	dbURL := os.Getenv("DATABASE_URL")

	if dbURL == "" {
		t.Fatal("DATABASE_URL is not set")
	}

	return &Client{
		dbURL: dbURL,
		config: config.Config{
			Services: map[string]string{},
			Registry: map[string]config.TargetConfig{
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

func registerServerTarget(t *testing.T, cfg config.Config, target string) {
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

	err := client.Register(
		"payment",
		func(ctx context.Context, input map[string]any) (any, error) {
			return map[string]any{
				"status": "paid",
				"amount": input["amount"],
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
			mode,
			input
		)
		VALUES ($1, $2, $3, $4)`,
		"order-1",
		"payment",
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

	body := map[string]string{
		"key":    "missing-ledger",
		"target": "payment",
	}

	rec := executeRequest(t, server, body)

	if rec.Code != http.StatusConflict {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusConflict,
			rec.Code,
		)
	}
}