from pathlib import Path
import yaml

def find_config(filename=None):
    current = Path.cwd().resolve()

    filenames = (
        ("odyssey.yaml", "odyssey.yml")
        if filename is None
        else (filename,)
    )

    while True:
        for name in filenames:
            candidate = current / name

            if candidate.is_file():
                return candidate

        if current.parent == current:
            return None

        current = current.parent


def load_config(filename="odyssey.yaml"):
    path = find_config(filename)

    if path is None:
        raise FileNotFoundError(f"Could not locate {filename}")

    with path.open("r", encoding="utf-8") as f:
        return yaml.safe_load(f) or {}, path


