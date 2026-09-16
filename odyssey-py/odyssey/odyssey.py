from psycopg_pool import AsyncConnectionPool
import os
import asyncio
import threading
from .config import load_config
from .build_ledger import BuildLedger
from .register import Register
from .workers import WorkerPool
from .environment import load_environment
from .db import async_init_db as async_initialize_db

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
    def __init__(self, db_url=None, pool_size=5, workers=4, config=None, namespace=None):
        self.namespace = namespace
        self.pool_size = pool_size
        self._pool_open = False
        self._stop_event = threading.Event()
        
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

        self.worker_pool = WorkerPool(
            registry=self._register,
            workers=workers
        )

        self.pool = AsyncConnectionPool(
            conninfo=self.db_url,
            min_size=self.pool_size,
            max_size=self.pool_size,
            open=False,
        )

    def _async_conn(self):
        return self.pool.connection()

    def _key(self, key):
        return f"{self.namespace}:{key}" if self.namespace else key

    async def _ensure_pool(self):
        if not self._pool_open:
            await self.pool.open()
            await self.pool.wait()
            self._pool_open = True

    async def async_init_db(self):

        await self._ensure_pool()

        async with self._async_conn() as conn:
            await async_initialize_db(conn)

    async def abuild_ledger(self, key, steps):

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
            
            if step.delegate is None and not self._register.exists_by_name(step.target):
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

        await self._ensure_pool()
        
        builder = BuildLedger(
            get_conn = self._async_conn,
            key=key,
            steps=steps
        )

        return await builder.run()

    def build_ledger(self, key, steps):
        try:
            asyncio.get_running_loop()
        except RuntimeError:
            return asyncio.run(self.abuild_ledger(key, steps))

        raise RuntimeError(
            "build_ledger() cannot be called from a running event loop; "
            "use await abuild_ledger() instead."
        )

    def register(self, target, fn):

        return self._register.register(
            target=target,
            fn=fn
        )

    async def _start(self):
        self.worker_pool.create_workers()

        try:
            self.worker_pool.wait()
        except KeyboardInterrupt:
            pass
        finally:
            self.worker_pool.stop()

    def start(self):
        asyncio.run(self._start())