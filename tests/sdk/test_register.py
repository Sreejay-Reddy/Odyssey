import pytest

from odyssey.register import Register


def test_register_function(registry_config):
    registry = Register(config=registry_config)

    def hello(name):
        return f"Hello, {name}!"

    result = registry.register(
        target="hello",
        fn=hello,
        ttl_ms=5000,
    )

    assert result is not None
    assert registry.exists("hello")

    registered = registry.get("hello")

    assert registered.fn is hello
    assert registered.ttl_ms == 5000


def test_register_multiple_functions(registry_config):
    registry = Register(config=registry_config)

    def hello():
        return "hello"

    def goodbye():
        return "goodbye"

    registry.register("hello", hello, 1000)
    registry.register("goodbye", goodbye, 2000)

    assert registry.exists("hello")
    assert registry.exists("goodbye")

    assert registry.get("hello").fn is hello
    assert registry.get("goodbye").fn is goodbye


def test_unknown_target():
    registry = Register(config={})

    assert registry.exists("missing") is False


def test_duplicate_target_rejected(registry_config):
    registry = Register(config=registry_config)

    def hello():
        return "hello"

    registry.register(
        "hello",
        hello,
        1000,
    )

    with pytest.raises(Exception):
        registry.register(
            "hello",
            hello,
            1000,
        )