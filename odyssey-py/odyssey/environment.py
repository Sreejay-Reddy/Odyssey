from dotenv import load_dotenv
from .config import find_config

def load_environment(filename=".env"):
    path = find_config(filename)

    if path is None:
        return None

    load_dotenv(path)

    return path