# Python E2E Tests

These tests verify the local Docker Compose MVP stack from the outside through
the HTTP API and Docker Compose commands.

Start the stack first:

```bash
docker compose up
```

Run the e2e tests from another terminal:

```bash
python3 -m unittest discover -s e2e -p 'test_*.py'
```

Useful environment variables:

- `E2E_API_URL` defaults to `http://localhost:8080`.
- `E2E_COMPOSE_FILE` defaults to `docker-compose.yml`.
- `E2E_DEMO_SERVICE` defaults to `target-nginx`.
- `E2E_TIMEOUT_SECONDS` defaults to `90`.
