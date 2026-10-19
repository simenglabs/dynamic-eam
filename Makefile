# Env (MONGO_URI, ES_URL, JWT_SECRET, ...) dari .env, lihat .env.example.
-include .env
export

SERVICES := $(patsubst services/%/,%,$(dir $(wildcard services/*/main.go)))

.PHONY: install dev build

install:
	pnpm install
	cd services && go mod download

# Semua service Go + semua zone Next.js. Ctrl-C mematikan semuanya.
dev:
	@trap 'kill 0' INT TERM; \
	for s in $(SERVICES); do (cd services && go run ./$$s) & done; \
	pnpm dev & \
	wait

build:
	pnpm build
	cd services && for s in $(SERVICES); do go build -o ../bin/$$s ./$$s || exit 1; done
