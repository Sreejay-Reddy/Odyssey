import psycopg
from psycopg_pool import AsyncConnectionPool
import asyncio
import selectors
import sys
import os
import uvicorn
from .config import load_config
from .build_ledger import BuildLedger
from .register import Register
from .server import OdysseyServer
from .cli import print_startup
from .environment import load_environment
from .db import async_init_db as async_initialize_db
from .db import init_db as sync_initialize_db

class Step:
    def __init__(
        self,
        target,
        *,
        delegate=None,
        **kwargs,
    ):
        self.target = target
        self.delegate = delegate
        self.kwargs = dict(kwargs)

class Odyssey:
    def __init__(self, db_url = None, config = None, default_ttl_ms=10000, namespace=None):
        self.default_ttl_ms = default_ttl_ms
        self.namespace = namespace
        
        if config is None:
            self.config, self.config_path = load_config()
        else:
            self.config = config
            self.config_path = None

        self._register = Register(config=self.config)

        load_environment()

        db_url = db_url or os.getenv("DATABASE_URL")

        if db_url is None:
            raise ValueError(
                "db_url must be provided. "
                "DATABASE_URL must be set."
            )

        self.db_url = db_url

        self.pool = AsyncConnectionPool(
            conninfo=self.db_url,
            min_size=20,
            max_size=20,
            open=False,
        )

    def _sync_conn(self):
        return psycopg.connect(self.db_url)

    async def _async_conn(self):
        return await self.pool.connection()

    def _ttl(self, ttl_ms):
        return ttl_ms if ttl_ms is not None else self.default_ttl_ms

    def _key(self, key):
        return f"{self.namespace}:{key}" if self.namespace else key

    def init_db(self):
        conn = self._sync_conn()

        try:
            sync_initialize_db(conn)
        finally:
            conn.close()

    async def async_init_db(self):
        conn = await self._async_conn()

        try:
            await async_initialize_db(conn)
        finally:
            await conn.close()

    def build_ledger(self, key, steps):

        if not isinstance(key, str):
            raise TypeError(
                "key must be a string."
            )

        if not key.strip():
            raise ValueError(
                "key cannot be empty."
            )

        if not steps:
            raise ValueError("build_ledger requires at least one step")

        for step in steps:
            if not isinstance(step, Step):
                raise TypeError(
                "steps must contain only Step objects."
                )

        targets = [step.target for step in steps]

        if len(targets) != len(set(targets)):
            raise ValueError("step targets must be unique")

        for step in steps:

            if not isinstance(step.target, str):
                raise TypeError(
                    "target must be a string."
                )

            if not step.target.strip():
                raise ValueError(
                    "target cannot be empty."
                )
            
            if step.delegate is None and not self._register.exists(step.target):
                raise ValueError(
                    f"Unknown target '{step.target}' and the Step is not delegated"
                )

        services = self.config.get("services", {})

        for step in steps:
            if step.delegate is not None:

                if not isinstance(step.delegate, str):
                    raise TypeError(
                        "delegate must be a string."
                    )

                if not step.delegate.strip():
                    raise ValueError(
                        "delegate cannot be empty."
                    )

                if step.delegate not in services:
                    raise ValueError(
                        f"Unknown service '{step.delegate}'."
                    )
        
        key = self._key(key)
        
        builder = BuildLedger(
            get_conn = self._sync_conn,
            key=key,
            steps=steps
        )

        return builder.run()

    def register(self, target, fn, ttl_ms = None):

        ttl_ms = self._ttl(ttl_ms)

        return self._register.register(
            target=target,
            fn=fn,
            ttl_ms=ttl_ms
        )

    def serve(self, host="127.0.0.1", port=8765):
        print_startup(self)

        server = OdysseyServer(
            registry=self._register,
            pool=self.pool,
        )

        config = uvicorn.Config(
            server.app,
            host=host,
            port=port,

            loop="auto",
            http="httptools",

            access_log=False,

            backlog=2048,
            timeout_keep_alive=30,

            limit_concurrency=None,
            limit_max_requests=None,
        )

        uvicorn_server = uvicorn.Server(config)

        if sys.platform == "win32":
            loop = asyncio.SelectorEventLoop(
                selectors.SelectSelector()
            )
            asyncio.set_event_loop(loop)

        else:
            loop = asyncio.new_event_loop()
            asyncio.set_event_loop(loop)

        try:
            loop.run_until_complete(
                self.pool.open()
            )

            loop.run_until_complete(
                self.pool.wait()
            )

            loop.run_until_complete(
                uvicorn_server.serve()
            )

        finally:
            loop.run_until_complete(
                self.pool.close()
            )

            loop.run_until_complete(
                loop.shutdown_asyncgens()
            )

            loop.close()