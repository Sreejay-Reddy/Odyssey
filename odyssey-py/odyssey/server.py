from starlette.applications import Starlette
from starlette.responses import JSONResponse
from starlette.routing import Route
from starlette.requests import Request

from .execution import Execution


class OdysseyServer:
    def __init__(self, registry, pool):
        self.execution = Execution(
            registry=registry,
            pool=pool
        )

        self.app = Starlette(
            routes=[
                Route(
                    "/health",
                    self.health,
                    methods=["GET"]
                ),
                Route(
                    "/execute",
                    self.execution.execute,
                    methods=["POST"]
                ),
            ]
        )

    async def health(self, request: Request):
        return JSONResponse(
            {"status": "ok"},
            status_code=200,
        )