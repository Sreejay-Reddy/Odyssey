from pathlib import Path
from pprint import pprint


def print_startup(odyssey):
    print()
    print("=" * 60)
    print("                         ODYSSEY")
    print("=" * 60)

    # Config
    config_path = getattr(odyssey, "config_path", None)

    if config_path and Path(config_path).exists():
        print()
        print("Configuration")
        print("-" * 60)
        print(f"odyssey.yaml: FOUND")
        print(f"path: {config_path}")
        print()
        pprint(odyssey.config, sort_dicts=False)
    else:
        print()
        print("Configuration")
        print("-" * 60)
        print("odyssey.yaml: NOT FOUND")

    # Registry
    print()
    print("Registered Targets")
    print("-" * 60)

    targets = odyssey._register.targets()

    if not targets:
        print("No targets registered.")
    else:
        for registered in targets:
            print(
                f"{registered.target:<30} "
                f"{registered.function_name}"
            )

    print()
    print(f"Total targets: {len(targets)}")

    print()
    print("=" * 60)
    print()