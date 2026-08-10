import os
import uuid
import sys
import asyncio

import psycopg
import pytest
import pytest_asyncio
from psycopg import AsyncConnection

from odyssey import Odyssey, Step


DATABASE_URL = os.getenv(
    "ODYSSEY_TEST_DATABASE_URL",
    "postgresql://postgres:postgres@localhost:5432/odyssey",
)


@pytest.fixture(scope="session")
def db_url():
    return DATABASE_URL


@pytest.fixture
def unique_key():
    return f"test-{uuid.uuid4().hex}"

@pytest.fixture(scope="session")
def event_loop_policy():
    if sys.platform == "win32":
        return asyncio.WindowsSelectorEventLoopPolicy()

    return asyncio.DefaultEventLoopPolicy()

@pytest.fixture
def registry_config():
    return {
        "registry": {
            "default": {
                "retry": {
                    "policy": "forever",
                    "delay": "1s",
                }
            }
        }
    }

@pytest.fixture
def odyssey(db_url):
    return Odyssey(
        db_url=db_url,
        config={
            "services": {},
            "registry": {
                "default": {
                    "retry": {
                        "policy": "forever",
                        "delay": "1s",
                    }
                }
            },
        },
    )

@pytest.fixture
def odyssey_without_default(db_url):
    return Odyssey(
        db_url=db_url,
        config={
            "services": {},
            "registry": {
                "hello": {
                    "retry": {
                        "policy": "forever",
                        "delay": "1s",
                    }
                }
            },
        },
    )


@pytest.fixture
def hello():
    def hello(name):
        return f"Hello, {name}!"

    return hello


@pytest.fixture
def failing_function():
    def failing_function():
        raise ValueError("boom")

    return failing_function


@pytest.fixture
def async_hello():
    async def async_hello(name):
        return f"Hello, {name}!"

    return async_hello


@pytest_asyncio.fixture
async def async_connection(db_url):
    conn = await AsyncConnection.connect(db_url)

    try:
        yield conn
    finally:
        await conn.close()


@pytest.fixture
def sync_connection(db_url):
    conn = psycopg.connect(db_url)

    try:
        yield conn
    finally:
        conn.close()


@pytest.fixture
def async_connection_factory(db_url):
    connections = []

    async def get_conn():
        conn = await AsyncConnection.connect(db_url)
        connections.append(conn)
        return conn

    yield get_conn

    # Execute closes its own connection.
    # This list is only useful for connections created outside Execute.
    for conn in connections:
        if not conn.closed:
            conn.close()


@pytest.fixture
def ledger(odyssey, hello, unique_key):
    odyssey.register(
        target="hello",
        fn=hello,
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

    return {
        "odyssey": odyssey,
        "key": unique_key,
        "target": "hello",
        "result": result,
    }


@pytest.fixture
def failing_ledger(odyssey, failing_function, unique_key):
    odyssey.register(
        target="failing",
        fn=failing_function,
    )

    odyssey.build_ledger(
        key=unique_key,
        steps=[
            Step("failing"),
        ],
    )

    return {
        "odyssey": odyssey,
        "key": unique_key,
        "target": "failing",
    }


@pytest.fixture
def async_ledger(odyssey, async_hello, unique_key):
    odyssey.register(
        target="async_hello",
        fn=async_hello,
    )

    odyssey.build_ledger(
        key=unique_key,
        steps=[
            Step(
                "async_hello",
                name="Summer",
            )
        ],
    )

    return {
        "odyssey": odyssey,
        "key": unique_key,
        "target": "async_hello",
    }


@pytest_asyncio.fixture
async def clean_database(db_url):
    conn = await AsyncConnection.connect(db_url)

    try:
        async with conn.cursor() as cur:
            await cur.execute("""
                TRUNCATE
                    odyssey_deliveries,
                    odyssey_journeys,
                    odyssey_ledger
                CASCADE;
            """)

        await conn.commit()

        yield

        async with conn.cursor() as cur:
            await cur.execute("""
                TRUNCATE
                    odyssey_deliveries,
                    odyssey_journeys,
                    odyssey_ledger
                CASCADE;
            """)

        await conn.commit()

    finally:
        await conn.close()