.PHONY: test build core-up core-down

test:
	cd agent && go test ./...
	cd core && go test ./...

build:
	cd agent && go build ./...
	cd core && go build ./...

core-up:
	cd core && docker compose -f deploy/docker-compose.yml up -d

core-down:
	cd core && docker compose -f deploy/docker-compose.yml down
