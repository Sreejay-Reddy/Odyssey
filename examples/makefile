DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/odyssey_bench?sslmode=disable
JOBS ?= 100000

.PHONY: bench-seed bench-run

bench-seed:
	psql "$(DATABASE_URL)" \
		-v ON_ERROR_STOP=1 \
		-v jobs=$(JOBS) \
		-f utils/seed.sql

bench-run: bench-seed
	go run .