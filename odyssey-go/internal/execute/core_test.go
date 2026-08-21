package execute

import (
	"context"
	"os"
	"reflect"
	"testing"
	"time"

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
		t.Fatalf("failed to connect to test database: %v", err)
	}

	cleanDatabase(t, conn)

	t.Cleanup(func() {
		cleanDatabase(t, conn)
		conn.Close(context.Background())
	})

	return conn
}

func cleanDatabase(t *testing.T, conn *pgx.Conn) {
	t.Helper()

	_, err := conn.Exec(
		context.Background(),
		`TRUNCATE
			odyssey_ledger,
			odyssey_journeys
		RESTART IDENTITY CASCADE`,
	)

	if err != nil {
		t.Fatalf("failed to clean test database: %v", err)
	}
}

func testExecution(t *testing.T, conn *pgx.Conn, key, target string) *execution {
	t.Helper()

	return &execution{
		key:    key,
		target: target,
		ttlMS:  10_000,
		conn:   conn,
	}
}

func insertClaimedLedger(
	t *testing.T,
	ctx context.Context,
	conn *pgx.Conn,
	key string,
	target string,
) {
	t.Helper()

	_, err := conn.Exec(
		ctx,
		`INSERT INTO odyssey_ledger (
			key,
			target,
			mode,
			status
		)
		VALUES ($1, $2, $3, $4)`,
		key,
		target,
		"local",
		"claimed",
	)

	if err != nil {
		t.Fatalf(
			"failed to insert ledger row: %v",
			err,
		)
	}
}

func insertCompletedJourney(
	t *testing.T,
	ctx context.Context,
	conn *pgx.Conn,
	key string,
	target string,
) {
	t.Helper()

	_, err := conn.Exec(
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
			'test-owner',
			NOW(),
			NOW(),
			nextval('odyssey_token_seq'),
			'completed',
			'{"result":"cached"}'
		)`,
		key,
		target,
	)

	if err != nil {
		t.Fatalf("failed to insert completed journey: %v", err)
	}
}

func TestAcquire(t *testing.T) {
	ctx := context.Background()

	conn := testConn(t)

	key := "test-acquire"
	target := "target-a"

	insertClaimedLedger(t, ctx, conn, key, target)

	e := testExecution(t, conn, key, target)

	acquired, err := e.acquire(ctx)

	if err != nil {
		t.Fatalf("acquire returned error: %v", err)
	}

	if !acquired {
		t.Fatal("expected execution to be acquired")
	}

	if e.ownerID == "" {
		t.Fatal("expected owner ID to be assigned")
	}

	if e.metadata.ownerID == "" {
		t.Fatal("expected journey owner ID")
	}

	if e.metadata.fencingToken == 0 {
		t.Fatal("expected fencing token")
	}

	if e.metadata.status != "claimed" {
		t.Fatalf(
			"expected journey status claimed, got %q",
			e.metadata.status,
		)
	}

	if !e.metadata.journeyAlive {
		t.Fatal("expected journey to be alive")
	}
}

func TestAcquireFailsWhenLedgerAlreadyStarted(t *testing.T) {
	ctx := context.Background()

	conn := testConn(t)

	key := "test-acquire-started"
	target := "target-a"

	_, err := conn.Exec(
		ctx,
		`INSERT INTO odyssey_ledger (
			key,
			target,
			mode,
			input
		)
		VALUES (
			$1,
			$2,
			'local',
			NULL
		)`,
		key,
		target,
	)

	if err != nil {
		t.Fatalf("failed to insert ledger row: %v", err)
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
			'executing'
		)`,
		key,
		target,
		"owner-1",
	)

	if err != nil {
		t.Fatalf("failed to insert active journey: %v", err)
	}

	e := testExecution(t, conn, key, target)

	acquired, err := e.acquire(ctx)

	if err != nil {
		t.Fatalf("acquire returned error: %v", err)
	}

	if acquired {
		t.Fatal("expected acquire to fail when execution is already active")
	}
}

func TestStartExecution(t *testing.T) {
	ctx := context.Background()

	conn := testConn(t)

	key := "test-start"
	target := "target-a"

	insertClaimedLedger(t, ctx, conn, key, target)

	e := testExecution(t, conn, key, target)

	acquired, err := e.acquire(ctx)

	if err != nil {
		t.Fatalf("acquire failed: %v", err)
	}

	if !acquired {
		t.Fatal("expected acquire to succeed")
	}

	started, err := e.startExecution(ctx)

	if err != nil {
		t.Fatalf("startExecution returned error: %v", err)
	}

	if !started {
		t.Fatal("expected execution to start")
	}

	var ledgerStatus string

	err = conn.QueryRow(
		ctx,
		`SELECT status
		FROM odyssey_ledger
		WHERE key = $1 AND target = $2`,
		key,
		target,
	).Scan(&ledgerStatus)

	if err != nil {
		t.Fatalf("failed to query ledger: %v", err)
	}

	if ledgerStatus != "executing" {
		t.Fatalf(
			"expected ledger status executing, got %q",
			ledgerStatus,
		)
	}

	var journeyStatus string

	err = conn.QueryRow(
		ctx,
		`SELECT status
		FROM odyssey_journeys
		WHERE key = $1 AND target = $2`,
		key,
		target,
	).Scan(&journeyStatus)

	if err != nil {
		t.Fatalf("failed to query journey: %v", err)
	}

	if journeyStatus != "executing" {
		t.Fatalf(
			"expected journey status executing, got %q",
			journeyStatus,
		)
	}
}

func TestStartExecutionRejectsStaleFencingToken(t *testing.T) {
	ctx := context.Background()

	conn := testConn(t)

	key := "test-stale-start"
	target := "target-a"

	insertClaimedLedger(t, ctx, conn, key, target)

	e := testExecution(t, conn, key, target)

	acquired, err := e.acquire(ctx)

	if err != nil {
		t.Fatalf("acquire failed: %v", err)
	}

	if !acquired {
		t.Fatal("expected acquire to succeed")
	}

	_, err = conn.Exec(
		ctx,
		`UPDATE odyssey_journeys
		SET fencing_token = nextval('odyssey_token_seq')
		WHERE key = $1 AND target = $2`,
		key,
		target,
	)

	if err != nil {
		t.Fatalf("failed to advance fencing token: %v", err)
	}

	started, err := e.startExecution(ctx)

	if err != nil {
		t.Fatalf("startExecution returned error: %v", err)
	}

	if started {
		t.Fatal("expected stale execution to be rejected")
	}
}

func TestComplete(t *testing.T) {
	ctx := context.Background()

	conn := testConn(t)

	key := "test-complete"
	target := "target-a"

	insertClaimedLedger(t, ctx, conn, key, target)

	e := testExecution(t, conn, key, target)

	acquired, err := e.acquire(ctx)

	if err != nil {
		t.Fatalf("acquire failed: %v", err)
	}

	if !acquired {
		t.Fatal("expected acquire to succeed")
	}

	started, err := e.startExecution(ctx)

	if err != nil {
		t.Fatalf("startExecution failed: %v", err)
	}

	if !started {
		t.Fatal("expected startExecution to succeed")
	}

	e.metadata.response = map[string]string{
		"result": "success",
	}

	completed, err := e.complete(ctx)

	if err != nil {
		t.Fatalf("complete returned error: %v", err)
	}

	if !completed {
		t.Fatal("expected completion to succeed")
	}

	var ledgerStatus string

	err = conn.QueryRow(
		ctx,
		`SELECT status
		FROM odyssey_ledger
		WHERE key = $1 AND target = $2`,
		key,
		target,
	).Scan(&ledgerStatus)

	if err != nil {
		t.Fatalf("failed to query ledger: %v", err)
	}

	if ledgerStatus != "completed" {
		t.Fatalf(
			"expected ledger status completed, got %q",
			ledgerStatus,
		)
	}

	var journeyStatus string

	err = conn.QueryRow(
		ctx,
		`SELECT status
		FROM odyssey_journeys
		WHERE key = $1 AND target = $2`,
		key,
		target,
	).Scan(&journeyStatus)

	if err != nil {
		t.Fatalf("failed to query journey: %v", err)
	}

	if journeyStatus != "completed" {
		t.Fatalf(
			"expected journey status completed, got %q",
			journeyStatus,
		)
	}
}

func TestCompleteRejectsStaleFencingToken(t *testing.T) {
	ctx := context.Background()

	conn := testConn(t)

	key := "test-stale-complete"
	target := "target-a"

	insertClaimedLedger(t, ctx, conn, key, target)

	e := testExecution(t, conn, key, target)

	acquired, err := e.acquire(ctx)

	if err != nil {
		t.Fatalf("acquire failed: %v", err)
	}

	if !acquired {
		t.Fatal("expected acquire to succeed")
	}

	started, err := e.startExecution(ctx)

	if err != nil {
		t.Fatalf("startExecution failed: %v", err)
	}

	if !started {
		t.Fatal("expected startExecution to succeed")
	}

	_, err = conn.Exec(
		ctx,
		`UPDATE odyssey_journeys
		SET fencing_token = nextval('odyssey_token_seq')
		WHERE key = $1 AND target = $2`,
		key,
		target,
	)

	if err != nil {
		t.Fatalf("failed to advance fencing token: %v", err)
	}

	completed, err := e.complete(ctx)

	if err != nil {
		t.Fatalf("complete returned error: %v", err)
	}

	if completed {
		t.Fatal("expected stale completion to be rejected")
	}
}

func TestAbandon(t *testing.T) {
	ctx := context.Background()

	conn := testConn(t)

	key := "test-abandon"
	target := "target-a"

	insertClaimedLedger(t, ctx, conn, key, target)

	e := testExecution(t, conn, key, target)

	acquired, err := e.acquire(ctx)

	if err != nil {
		t.Fatalf("acquire failed: %v", err)
	}

	if !acquired {
		t.Fatal("expected acquire to succeed")
	}

	started, err := e.startExecution(ctx)

	if err != nil {
		t.Fatalf("startExecution failed: %v", err)
	}

	if !started {
		t.Fatal("expected startExecution to succeed")
	}

	abandoned, err := e.abandon(ctx)

	if err != nil {
		t.Fatalf("abandon returned error: %v", err)
	}

	if !abandoned {
		t.Fatal("expected abandon to succeed")
	}

	var expiresAt time.Time

	err = conn.QueryRow(
		ctx,
		`SELECT expires_at
		FROM odyssey_journeys
		WHERE key = $1 AND target = $2`,
		key,
		target,
	).Scan(&expiresAt)

	if err != nil {
		t.Fatalf("failed to query journey: %v", err)
	}

	if expiresAt.After(time.Now()) {
		t.Fatal("expected abandoned journey to be expired")
	}
}

func TestAbandonRejectsStaleFencingToken(t *testing.T) {
	ctx := context.Background()

	conn := testConn(t)

	key := "test-stale-abandon"
	target := "target-a"

	insertClaimedLedger(t, ctx, conn, key, target)

	e := testExecution(t, conn, key, target)

	acquired, err := e.acquire(ctx)

	if err != nil {
		t.Fatalf("acquire failed: %v", err)
	}

	if !acquired {
		t.Fatal("expected acquire to succeed")
	}

	started, err := e.startExecution(ctx)

	if err != nil {
		t.Fatalf("startExecution failed: %v", err)
	}

	if !started {
		t.Fatal("expected startExecution to succeed")
	}

	_, err = conn.Exec(
		ctx,
		`UPDATE odyssey_journeys
		SET fencing_token = nextval('odyssey_token_seq')
		WHERE key = $1 AND target = $2`,
		key,
		target,
	)

	if err != nil {
		t.Fatalf("failed to advance fencing token: %v", err)
	}

	abandoned, err := e.abandon(ctx)

	if err != nil {
		t.Fatalf("abandon returned error: %v", err)
	}

	if abandoned {
		t.Fatal("expected stale abandon to be rejected")
	}
}

func TestAcquireExistingCompletedExecution(t *testing.T) {
	ctx := context.Background()

	conn := testConn(t)

	key := "test-existing-completed"
	target := "target-a"

	insertClaimedLedger(t, ctx, conn, key, target)
	insertCompletedJourney(t, ctx, conn, key, target)

	e := testExecution(t, conn, key, target)

	acquired, err := e.acquire(ctx)

	if err != nil {
		t.Fatalf("acquire returned error: %v", err)
	}

	if acquired {
		t.Fatal("expected completed execution not to be acquired")
	}
}

func TestFullLifecycle(t *testing.T) {
	ctx := context.Background()

	conn := testConn(t)

	key := "test-full-lifecycle"
	target := "target-a"

	insertClaimedLedger(t, ctx, conn, key, target)

	e := testExecution(t, conn, key, target)

	acquired, err := e.acquire(ctx)

	if err != nil {
		t.Fatalf("acquire failed: %v", err)
	}

	if !acquired {
		t.Fatal("expected acquire to succeed")
	}

	if e.metadata.status != "claimed" {
		t.Fatalf("expected claimed status, got %q", e.metadata.status)
	}

	started, err := e.startExecution(ctx)

	if err != nil {
		t.Fatalf("startExecution failed: %v", err)
	}

	if !started {
		t.Fatal("expected startExecution to succeed")
	}

	if e.metadata.fencingToken == 0 {
		t.Fatal("expected fencing token")
	}

	e.metadata.response = map[string]string{
		"message": "hello",
	}

	completed, err := e.complete(ctx)

	if err != nil {
		t.Fatalf("complete failed: %v", err)
	}

	if !completed {
		t.Fatal("expected completion to succeed")
	}

	var ledgerStatus string
	var journeyStatus string

	err = conn.QueryRow(
		ctx,
		`SELECT status
		FROM odyssey_ledger
		WHERE key = $1 AND target = $2`,
		key,
		target,
	).Scan(&ledgerStatus)

	if err != nil {
		t.Fatalf("failed to query ledger: %v", err)
	}

	err = conn.QueryRow(
		ctx,
		`SELECT status
		FROM odyssey_journeys
		WHERE key = $1 AND target = $2`,
		key,
		target,
	).Scan(&journeyStatus)

	if err != nil {
		t.Fatalf("failed to query journey: %v", err)
	}

	if ledgerStatus != "completed" {
		t.Fatalf(
			"expected ledger completed, got %q",
			ledgerStatus,
		)
	}

	if journeyStatus != "completed" {
		t.Fatalf(
			"expected journey completed, got %q",
			journeyStatus,
		)
	}
}

func TestTransactionRollbackOnFailedStart(t *testing.T) {
	ctx := context.Background()

	conn := testConn(t)

	key := "test-start-rollback"
	target := "target-a"

	insertClaimedLedger(t, ctx, conn, key, target)

	e := testExecution(t, conn, key, target)

	acquired, err := e.acquire(ctx)
	if err != nil {
		t.Fatalf("acquire failed: %v", err)
	}

	if !acquired {
		t.Fatal("expected acquire to succeed")
	}

	// Force the journey to have a different fencing token.
	_, err = conn.Exec(
		ctx,
		`UPDATE odyssey_journeys
		SET fencing_token = nextval('odyssey_token_seq')
		WHERE key = $1 AND target = $2`,
		key,
		target,
	)

	if err != nil {
		t.Fatalf("failed to invalidate fencing token: %v", err)
	}

	started, err := e.startExecution(ctx)

	if err != nil {
		t.Fatalf("startExecution returned error: %v", err)
	}

	if started {
		t.Fatal("expected startExecution to fail")
	}

	var ledgerStatus string

	err = conn.QueryRow(
		ctx,
		`SELECT status
		FROM odyssey_ledger
		WHERE key = $1 AND target = $2`,
		key,
		target,
	).Scan(&ledgerStatus)

	if err != nil {
		t.Fatalf("failed to query ledger status: %v", err)
	}

	if ledgerStatus != "claimed" {
		t.Fatalf(
			"expected ledger status to be rolled back to claimed, got %s",
			ledgerStatus,
		)
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

func TestAcquireNoRowsReturnsFalse(t *testing.T) {
	ctx := context.Background()

	conn := testConn(t)

	e := testExecution(
		t,
		conn,
		"missing-key",
		"missing-target",
	)

	acquired, err := e.acquire(ctx)

	if err != nil {
		t.Fatalf("acquire returned error: %v", err)
	}

	if acquired {
		t.Fatal("expected acquire to return false")
	}
}

func TestCompleteWithoutExecutionReturnsFalse(t *testing.T) {
	ctx := context.Background()

	conn := testConn(t)

	e := testExecution(
		t,
		conn,
		"missing-complete",
		"target-a",
	)

	completed, err := e.complete(ctx)

	if err != nil {
		t.Fatalf("complete returned error: %v", err)
	}

	if completed {
		t.Fatal("expected complete to return false")
	}
}

func TestAbandonWithoutExecutionReturnsFalse(t *testing.T) {
	ctx := context.Background()

	conn := testConn(t)

	e := testExecution(
		t,
		conn,
		"missing-abandon",
		"target-a",
	)

	abandoned, err := e.abandon(ctx)

	if err != nil {
		t.Fatalf("abandon returned error: %v", err)
	}

	if abandoned {
		t.Fatal("expected abandon to return false")
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
