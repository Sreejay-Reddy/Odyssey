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