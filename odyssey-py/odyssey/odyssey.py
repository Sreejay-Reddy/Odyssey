from .config import load_config
from .build_ledger import BuildLedger

class Odyssey:
    def __init__(self, get_conn = None, config = None, default_ttl_ms=10000, namespace=None):
        self.default_ttl_ms = default_ttl_ms
        self.namespace = namespace
        self.config = config or load_config()

        if get_conn is None:
            raise ValueError(
                "get_conn must be provided."
        )

        self.get_conn = get_conn

    def _conn(self):
        return self.get_conn()

    def _ttl(self, ttl_ms):
        return ttl_ms if ttl_ms is not None else self.default_ttl_ms

    def _key(self, key):
        return f"{self.namespace}:{key}" if self.namespace else key

    def build_ledger(self, key, steps, delegates = None):
        delegates = delegates or {}

        if not self.steps:
            raise ValueError("build_ledger requires at least one step")

        if len(self.steps) != len(set(self.steps)):
            raise ValueError("steps must be unique")

        unknown = set(self.delegates) - set(self.steps)
        if unknown:
            raise ValueError(f"delegates references steps not present in `steps`: {sorted(unknown)}")

        for step in steps:
            if step not in self.config["registry"]:
                raise ValueError(
                    f"Unknown target '{step}'. "
                    "Target is not registered in odyssey.yaml."
                )

        for service in delegates.values():
            if service not in self.config["services"]:
                raise ValueError(
                    f"Unknown service '{service}'. "
                    "Service is not defined in odyssey.yaml."
                )
        
        key = self._key(key)
        
        builder = BuildLedger(
            get_conn = self._conn,
            key=key,
            steps=steps,
            delegates=delegates
        )

        return builder.run()


        