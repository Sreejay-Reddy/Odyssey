from starlette.applications import Starlette
from starlette.responses import JSONResponse
from starlette.routing import Route
from starlette.requests import Request

from .execution import Execution


class OdysseyServer:
    def __init__(self, registry, get_conn):
        self.execution = Execution(
            registry=registry,
            get_conn=get_conn
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