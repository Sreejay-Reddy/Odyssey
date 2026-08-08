from .core import (
    acquire, 
    start_execution, 
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
        get_conn,
        key,
        target,
        fn,
        ttl_ms,
        kwargs=None
    ):

        self.get_conn = get_conn
        self.key = key
        self.target = target
        self.fn = fn
        self.ttl_ms = ttl_ms
        self.kwargs = dict(kwargs or {})

    async def run(self):
        conn = await self.get_conn()

        try:
            acquired = await acquire(
                conn=conn,
                key=self.key,
                target=self.target,
                ttl_ms=self.ttl_ms
            )

            if not acquired.acquired:

                if acquired.status == "completed":
                    cached = fetch_cached_response(
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

            # Enter execution boundary
            started = await start_execution(
                conn,
                self.key,
                target=self.target,
                fencing_token=acquired.fencing_token
            )

            if not started.success:
                return ExecuteResult(
                    success=False,
                    status=started.status
                )

            try:

                if inspect.iscoroutinefunction(self.fn):
                    response = await self.fn(**self.kwargs)
                else:
                    response = await run_in_threadpool(
                        self.fn,
                        **self.kwargs
                    )

            except Exception as e:
                try:
                    await abandon(
                        conn,
                        self.key,
                        target=self.target,
                        fencing_token=acquired.fencing_token
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

            completed = await complete(
                conn,
                self.key,
                target=self.target,
                fencing_token=acquired.fencing_token,
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
        
        finally:
            await conn.close()


        

