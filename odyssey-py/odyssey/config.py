from pathlib import Path
import yaml

def find_config(filename="odyssey.yaml"):
    current = Path.cwd()

    while True:
        candidate = current / filename
        if candidate.exists():
            return candidate

        if current.parent == current:
            return None

        current = current.parent


def load_config(filename="odyssey.yaml"):
    path = find_config(filename)

    if path is None:
        raise FileNotFoundError(f"Could not locate {filename}")

    with path.open("r", encoding="utf-8") as f:
        return yaml.safe_load(f)

config = load_config()
print(config["registry"])
