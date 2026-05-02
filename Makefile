.PHONY: test build e2e-test swagger core-up core-down

test:
	cd agent && go test ./...
	cd core && go test ./...

build:
	cd agent && go build ./...
	cd core && go build ./...

e2e-test:
	python3 -m unittest discover -s e2e -p 'test_*.py'

swagger:
	cd core && go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g cmd/core/main.go -o docs --parseInternal

core-up:
	cd core && docker compose -f deploy/docker-compose.yml up -d

core-down:
	cd core && docker compose -f deploy/docker-compose.yml down
