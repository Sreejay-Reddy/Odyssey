from starlette.requests import Request
from starlette.responses import JSONResponse

from .execute import Execute


class Execution:
    def __init__(self, registry, pool):
        self.registry = registry
        self.pool = pool

    async def execute(self, request: Request):
        try:
            data = await request.json()
        except Exception:
            return JSONResponse(
                {"error": "Invalid JSON payload."},
                status_code=400,
            )

        key = data.get("key")
        target = data.get("target")

        if not key:
            return JSONResponse(
                {"error": "Missing key."},
                status_code=400,
            )

        if not target:
            return JSONResponse(
                {"error": "Missing target."},
                status_code=400,
            )

        if not self.registry.exists(target):
            return JSONResponse(
                {"error": f"Target '{target}' is not registered."},
                status_code=404,
            )

        registered = self.registry.get(target)

        try:
            execution = Execute(
                pool=self.pool,
                key=key,
                target=target,
                ttl_ms=registered.ttl_ms,
                fn=registered.fn
            )

            result = await execution.run()

            return JSONResponse(
                {
                    "status": result.status,
                    "key": key,
                    "target": target,
                    "result": result.response,
                },
                status_code=200,
            )

        except Exception as e:
            return JSONResponse(
                {
                    "status": "failed",
                    "key": key,
                    "target": target,
                    "error": str(e),
                },
                status_code=500,
            )