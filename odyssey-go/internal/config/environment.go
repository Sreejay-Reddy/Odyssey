package config

import (
	"os"
)

func ReadEnvironment() (string, bool) {
    value, exists := os.LookupEnv("DATABASE_URL")
    return value, exists
}