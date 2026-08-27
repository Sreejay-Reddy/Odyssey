import pytest

from odyssey import Step
from odyssey.core import (
    acquire,
    complete,
    abandon,
)


@pytest.mark.asyncio
async def test_acquire_new_journey(
    clean_database,
    async_connection,
    ledger,
):
    result = await acquire(
        async_connection,
        ledger["key"],
        target=ledger["target"],
        owner_id="worker-1",
        ttl_ms=10000,
    )

    assert result.acquired is True
    assert result.owner_id == "worker-1"
    assert result.target == "hello"
    assert result.fencing_token is not None
    assert result.status == "claimed"
    assert result.journey_alive is True


@pytest.mark.asyncio
async def test_active_journey_cannot_be_acquired(
    clean_database,
    async_connection,
    ledger,
):
    first = await acquire(
        async_connection,
        ledger["key"],
        target="hello",
        owner_id="worker-1",
    )

    second = await acquire(
        async_connection,
        ledger["key"],
        target="hello",
        owner_id="worker-2",
    )

    assert first.acquired is True
    assert second.acquired is False


@pytest.mark.asyncio
async def test_expired_claimed_journey_can_be_reacquired(
    clean_database,
    async_connection,
    ledger,
):
    first = await acquire(
        async_connection,
        ledger["key"],
        target="hello",
        owner_id="worker-1",
    )

    async with async_connection.cursor() as cur:
        await cur.execute("""
            UPDATE odyssey_journeys
            SET
                expires_at = NOW() - INTERVAL '1 second'
            WHERE key = %s
              AND target = %s;
        """, (
            ledger["key"],
            "hello",
        ))

    await async_connection.commit()

    second = await acquire(
        async_connection,
        ledger["key"],
        target="hello",
        owner_id="worker-2",
    )

    assert second.acquired is True
    assert second.owner_id == "worker-2"
    assert second.fencing_token > first.fencing_token

@pytest.mark.asyncio
async def test_complete(
    clean_database,
    async_connection,
    ledger,
):
    acquired = await acquire(
        async_connection,
        ledger["key"],
        target="hello",
    )

    result = await complete(
        async_connection,
        ledger["key"],
        target="hello",
        fencing_token=acquired.fencing_token,
        execution_result={
            "message": "Hello, Summer!",
        },
    )

    assert result.success is True


@pytest.mark.asyncio
async def test_complete_rejects_stale_token(
    clean_database,
    async_connection,
    ledger,
):
    acquired = await acquire(
        async_connection,
        ledger["key"],
        target="hello",
    )

    result = await complete(
        async_connection,
        ledger["key"],
        target="hello",
        fencing_token=acquired.fencing_token + 999,
        execution_result={
            "message": "bad",
        },
    )

    assert result.success is False


@pytest.mark.asyncio
async def test_complete_is_atomic(
    clean_database,
    async_connection,
    ledger,
):
    acquired = await acquire(
        async_connection,
        ledger["key"],
        target="hello",
    )

    result = await complete(
        async_connection,
        ledger["key"],
        target="hello",
        fencing_token=acquired.fencing_token + 999,
        execution_result={
            "message": "bad",
        },
    )

    assert result.success is False

    async with async_connection.cursor() as cur:
        await cur.execute("""
            SELECT status
            FROM odyssey_ledger
            WHERE key = %s
              AND target = %s;
        """, (
            ledger["key"],
            "hello",
        ))

        ledger_status = await cur.fetchone()

        await cur.execute("""
            SELECT status
            FROM odyssey_journeys
            WHERE key = %s
              AND target = %s;
        """, (
            ledger["key"],
            "hello",
        ))

        journey_status = await cur.fetchone()

    assert ledger_status[0] == "claimed"
    assert journey_status[0] == "claimed"


@pytest.mark.asyncio
async def test_abandon(
    clean_database,
    async_connection,
    ledger,
):
    acquired = await acquire(
        async_connection,
        ledger["key"],
        target="hello",
    )

    result = await abandon(
        async_connection,
        ledger["key"],
        target="hello",
        fencing_token=acquired.fencing_token,
    )

    assert result.success is True

    async with async_connection.cursor() as cur:
        await cur.execute("""
            SELECT expires_at <= NOW()
            FROM odyssey_journeys
            WHERE key = %s
              AND target = %s;
        """, (
            ledger["key"],
            "hello",
        ))

        row = await cur.fetchone()

    assert row[0] is True


@pytest.mark.asyncio
async def test_completed_journey_cannot_be_reacquired(
    clean_database,
    async_connection,
    ledger,
):
    first = await acquire(
        async_connection,
        ledger["key"],
        target="hello",
        owner_id="worker-1",
    )

    await complete(
        async_connection,
        ledger["key"],
        target="hello",
        fencing_token=first.fencing_token,
        execution_result={
            "message": "Hello, Summer!",
        },
    )

    second = await acquire(
        async_connection,
        ledger["key"],
        target="hello",
        owner_id="worker-2",
    )

    assert first.acquired is True
    assert second.acquired is False
    assert second.status == "completed"


@pytest.mark.asyncio
async def test_abandoned_journey_can_be_reacquired(
    clean_database,
    async_connection,
    ledger,
):
    first = await acquire(
        async_connection,
        ledger["key"],
        target="hello",
        owner_id="worker-1",
    )

    result = await abandon(
        async_connection,
        ledger["key"],
        target="hello",
        fencing_token=first.fencing_token,
    )

    assert result.success is True

    second = await acquire(
        async_connection,
        ledger["key"],
        target="hello",
        owner_id="worker-2",
    )

    assert second.acquired is True
    assert second.owner_id == "worker-2"
    assert second.fencing_token > first.fencing_token


@pytest.mark.asyncio
async def test_two_workers_only_one_acquires(
    clean_database,
    async_connection,
    ledger,
):
    first = await acquire(
        async_connection,
        ledger["key"],
        target="hello",
        owner_id="worker-1",
    )

    second = await acquire(
        async_connection,
        ledger["key"],
        target="hello",
        owner_id="worker-2",
    )

    assert first.acquired is True
    assert second.acquired is False

    assert first.owner_id == "worker-1"
    assert second.owner_id == "worker-1"