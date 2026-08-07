class RegisteredTarget:
    def __init__(self, target, fn, ttl_ms):
        self.target = target
        self.fn = fn
        self.function_name = fn.__name__
        self.ttl_ms = ttl_ms


class Register:
    def __init__(self, config):
        self.config = config
        self._registry = {}

    def register(self, target, fn, ttl_ms):
        registry = self.config.get("registry", {})
        default = registry.get("default")

        if not callable(fn):
            raise TypeError(
                "fn must be callable."
            )

        if ttl_ms <= 0:
            raise ValueError(
                "ttl_ms must be greater than zero."
            )

        if target not in registry and default is None:
            raise ValueError(
                f"Target '{target}' is not defined in odyssey.yaml."
                "No default is defined in odyssey.yaml"
            )

        if target in self._registry:
            raise ValueError(
                f"Target '{target}' is already registered."
            )

        if not target.strip():
            raise ValueError(
                "target cannot be empty."
            )

        registered = RegisteredTarget(target, fn, ttl_ms)
        self._registry[target] = registered

        return registered

    def get(self, target):
        try:
            return self._registry[target]
        except KeyError:
            raise ValueError(
                f"Target '{target}' has not been registered."
            )

    def exists(self, target):
        return target in self._registry

    def targets(self):
        return list(self._registry.values())

    def registry(self):
        return dict(self._registry)