from .config import load_config
from .build_ledger import BuildLedger
from .register import Register

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
    def __init__(self, get_conn = None, config = None, default_ttl_ms=10000, namespace=None):
        self.default_ttl_ms = default_ttl_ms
        self.namespace = namespace
        
        if config is None:
            self.config, self.config_path = load_config()
        else:
            self.config = config
            self.config_path = None

        self._register = Register(config=self.config)

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

        registry = self.config.get("registry", {})

        for step in steps:

            if not isinstance(step.target, str):
                raise TypeError(
                    "target must be a string."
                )

            if not step.target.strip():
                raise ValueError(
                    "target cannot be empty."
                )
            
            if step.target not in registry and "default" not in registry:
                raise ValueError(
                    f"Unknown target '{step.target}'. "
                    "No default target configuration exists in odyssey.yaml."
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
            get_conn = self._conn,
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
        