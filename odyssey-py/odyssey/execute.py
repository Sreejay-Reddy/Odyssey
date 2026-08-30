from .core import (
    acquire, 
    abandon, 
    complete
)

from .helper import(
    fetch_cached_response
)

import inspect
from starlette.concurrency import run_in_threadpool

from .results import ExecuteResult

class Execute:
    def __init__(
        self,
        pool,
        key,
        target,
        fn,
        ttl_ms
    ):

        self.pool = pool
        self.key = key
        self.target = target
        self.fn = fn
        self.ttl_ms = ttl_ms

    async def run(self):

        async with self.pool.connection() as conn:

            acquired = await acquire(
                conn=conn,
                key=self.key,
                target=self.target,
                ttl_ms=self.ttl_ms
            )

            if not acquired.acquired:

                if acquired.status == "completed":
                    cached = await fetch_cached_response(
                        conn,
                        self.key,
                        target= self.target
                    )

                    return ExecuteResult(
                        key=self.key,
                        target=self.target,
                        success=True,
                        status="completed",
                        response=cached["response"]
                    )

                return ExecuteResult(
                    key=self.key,
                    target=self.target,
                    success=False,
                    status=acquired.status,
                )
                
        try:

            kwargs = acquired.input or {}

            if inspect.iscoroutinefunction(self.fn):
                response = await self.fn(**kwargs)
            else:
                response = await run_in_threadpool(
                    self.fn,
                    **kwargs
                )

        except Exception as e:
            async with self.pool.connection() as conn:
                try:
                    await abandon(
                        conn,
                        self.key,
                        target=self.target,
                        attempt=acquired.attempts
                    )
                            
                except Exception as cleanup_error:
                    raise RuntimeError(
                        "Could not expire lease after fn() failure"
                        f"Exception:{cleanup_error}"
                    )

            raise RuntimeError(
                "Execution terminated with an exception after execution started. "
                "Side effects may have partially completed. "
                f"exception: {e}"
            )

        async with self.pool.connection() as conn:

            completed = await complete(
                conn,
                self.key,
                target=self.target,
                attempt=acquired.attempts,
                execution_result=response
            )

            if not completed.success:
                raise RuntimeError(
                    "Execution completed but could not be canonically finalized. " 
                )

            return ExecuteResult(
                key=self.key,
                target=self.target,
                success=True,
                status="completed",
                response=response
            )

        

