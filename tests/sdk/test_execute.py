import pytest

from odyssey import Step
from odyssey.execute import Execute


@pytest.mark.asyncio
async def test_execute_queries_input_from_ledger(
    clean_database,
    async_connection_factory,
    ledger,
):
    execution = Execute(
        get_conn=async_connection_factory,
        key=ledger["key"],
        target=ledger["target"],
        fn=ledger["odyssey"]._register.get("hello").fn,
        ttl_ms=10000,
    )

    result = await execution.run()

    assert result.success is True
    assert result.status == "completed"
    assert result.response == "Hello, Summer!"


@pytest.mark.asyncio
async def test_execute_does_not_require_input_from_agent(
    clean_database,
    async_connection_factory,
    ledger,
):
    execution = Execute(
        get_conn=async_connection_factory,
        key=ledger["key"],
        target=ledger["target"],
        fn=ledger["odyssey"]._register.get("hello").fn,
        ttl_ms=10000,
    )

    # There is deliberately no kwargs/input argument here.
    result = await execution.run()

    assert result.success is True
    assert result.response == "Hello, Summer!"


@pytest.mark.asyncio
async def test_execute_async_function(
    clean_database,
    async_connection_factory,
    async_ledger,
):
    fn = async_ledger["odyssey"]._register.get("async_hello").fn

    execution = Execute(
        get_conn=async_connection_factory,
        key=async_ledger["key"],
        target=async_ledger["target"],
        fn=fn,
        ttl_ms=10000,
    )

    result = await execution.run()

    assert result.success is True
    assert result.status == "completed"
    assert result.response == "Hello, Summer!"


@pytest.mark.asyncio
async def test_execute_failure_abandons(
    clean_database,
    async_connection_factory,
    failing_ledger,
):
    fn = failing_ledger["odyssey"]._register.get("failing").fn

    execution = Execute(
        get_conn=async_connection_factory,
        key=failing_ledger["key"],
        target=failing_ledger["target"],
        fn=fn,
        ttl_ms=10000,
    )

    with pytest.raises(
        RuntimeError,
        match="Execution terminated",
    ):
        await execution.run()

    conn = await async_connection_factory()

    try:
        async with conn.cursor() as cur:
            await cur.execute("""
                SELECT expires_at <= NOW()
                FROM odyssey_journeys
                WHERE key = %s
                  AND target = %s;
            """, (
                failing_ledger["key"],
                failing_ledger["target"],
            ))

            row = await cur.fetchone()

        assert row[0] is True

    finally:
        await conn.close()