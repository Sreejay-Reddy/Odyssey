import pytest

from odyssey import Step


def test_build_ledger(
    odyssey,
    hello,
    unique_key,
):
    odyssey.register(
        "hello",
        hello,
    )

    result = odyssey.build_ledger(
        key=unique_key,
        steps=[
            Step(
                "hello",
                name="Summer",
            )
        ],
    )

    assert result is not None


def test_build_ledger_requires_string_key(
    odyssey,
):
    with pytest.raises(TypeError):
        odyssey.build_ledger(
            key=123,
            steps=[
                Step("hello"),
            ],
        )


def test_build_ledger_rejects_empty_key(
    odyssey,
):
    with pytest.raises(ValueError):
        odyssey.build_ledger(
            key="",
            steps=[
                Step("hello"),
            ],
        )


def test_build_ledger_requires_steps(
    odyssey,
):
    with pytest.raises(ValueError):
        odyssey.build_ledger(
            key="test",
            steps=[],
        )


def test_build_ledger_requires_step_objects(
    odyssey,
):
    with pytest.raises(TypeError):
        odyssey.build_ledger(
            key="test",
            steps=["hello"],
        )


def test_duplicate_targets_rejected(
    odyssey,
):
    with pytest.raises(ValueError):
        odyssey.build_ledger(
            key="test",
            steps=[
                Step("hello"),
                Step("hello"),
            ],
        )


def test_unknown_target_rejected(
    odyssey_without_default,
):
    with pytest.raises(ValueError):
        odyssey_without_default.build_ledger(
            key="test",
            steps=[
                Step("unknown"),
            ],
        )


def test_invalid_delegate_rejected(
    odyssey,
):
    with pytest.raises(ValueError):
        odyssey.build_ledger(
            key="test",
            steps=[
                Step(
                    "hello",
                    delegate="missing",
                ),
            ],
        )


def test_build_ledger_assigns_sequence(
    odyssey,
    hello,
    unique_key,
    sync_connection,
):
    odyssey.register(
        "hello",
        hello,
    )
    odyssey.register(
        "hello_2",
        hello,
    )
    odyssey.register(
        "hello_3",
        hello,
    )

    odyssey.build_ledger(
        key=unique_key,
        steps=[
            Step("hello"),
            Step("hello_2"),
            Step("hello_3"),
        ],
    )

    with sync_connection.cursor() as cur:
        cur.execute(
            """
            SELECT target, sequence
            FROM odyssey_ledger
            WHERE key = %s
            ORDER BY sequence
            """,
            (unique_key,),
        )

        rows = cur.fetchall()

    assert rows == [
        ("hello", 1),
        ("hello_2", 2),
        ("hello_3", 3),
    ]


@pytest.mark.asyncio
async def test_acquire_preserves_ledger_sequence(
    clean_database,
    async_connection,
    odyssey,
    hello,
    unique_key,
):
    odyssey.register(
        target="hello",
        fn=hello,
    )

    odyssey.build_ledger(
        key=unique_key,
        steps=[
            Step(
                "hello",
                name="Summer",
            )
        ],
    )

    async with async_connection.cursor() as cur:
        await cur.execute("""
            SELECT sequence
            FROM odyssey_ledger
            WHERE key = %s
              AND target = %s;
        """, (
            unique_key,
            "hello",
        ))

        row = await cur.fetchone()

    assert row[0] == 1