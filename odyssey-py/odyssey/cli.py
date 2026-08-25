from pathlib import Path
from pprint import pprint
import os
import psycopg
import argparse

from .config import find_config
from .environment import load_environment
from .db import init_db


from pathlib import Path
from pprint import pprint


def print_startup(odyssey):
    print()
    print("╔════════════════════════════════════════════════════════════╗")
    print("║                         ODYSSEY                            ║")
    print("╚════════════════════════════════════════════════════════════╝")

    # Configuration
    config_path = getattr(odyssey, "config_path", None)

    print()
    print("Configuration")
    print("────────────────────────────────────────────────────────────")

    if config_path and Path(config_path).exists():
        print("odyssey.yaml    FOUND")
        print(f"path            {config_path}")
        print()
        pprint(odyssey.config, sort_dicts=False)
    else:
        print("odyssey.yaml    NOT FOUND")

    # Registered targets
    print()
    print("Registered Targets")
    print("────────────────────────────────────────────────────────────")

    targets = odyssey._register.targets()

    if not targets:
        print("No targets registered.")
    else:
        for registered in targets:
            print(
                f"✓ {registered.target:<30}"
                f"{registered.function_name}"
            )

    print()
    print(f"Total targets: {len(targets)}")

    # Services
    services = odyssey.config.get("services", {})

    print()
    print("Configured Services")
    print("────────────────────────────────────────────────────────────")

    if not services:
        print("No services configured.")
    else:
        for name, url in services.items():
            print(f"✓ {name:<30}{url}")

    # Runtime
    print()
    print("Runtime")
    print("────────────────────────────────────────────────────────────")
    print(f"Database        {'configured' if odyssey.db_url else 'not configured'}")
    print(f"Namespace       {odyssey.namespace or 'default'}")

    print()
    print("Odyssey is ready.")
    print()
    print("═" * 60)
    print()

def init():
    env_path = load_environment()

    print("Initializing Odyssey...\n")

    if env_path is not None:
        print(f"✓ .env found: {env_path}")
    else:
        print("✗ .env not found")

    db_url = os.getenv("DATABASE_URL")

    if db_url:
        print("✓ DATABASE_URL found")
    else:
        print("✗ DATABASE_URL not found")

    config_path = find_config()

    if config_path is not None:
        print(f"✓ odyssey.yaml found: {config_path}")
    else:
        print("✗ odyssey.yaml not found")

    if not db_url:
        raise ValueError(
            "DATABASE_URL is required to initialize the database."
        )

    conn = psycopg.connect(db_url)

    try:
        init_db(conn)
    finally:
        conn.close()

    print("✓ Database initialized")

def main():
    parser = argparse.ArgumentParser(
        prog="odyssey",
        description="Odyssey is a durable execution engine built around PostgreSQL.",
    )

    subparsers = parser.add_subparsers(
        dest="command",
        title="commands",
        metavar="",
    )

    subparsers.add_parser(
        "init",
        help="Initialize an Odyssey project",
    )

    args = parser.parse_args()

    if args.command is None:
        parser.print_help()
        return

    if args.command == "init":
        init()