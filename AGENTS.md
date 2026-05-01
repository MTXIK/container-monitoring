# Repository Guidelines

## Project Structure & Module Organization

This repository is a minimal scaffold for `container-monitoring`. It is expected to contain several backends and a frontend. Keep each application in its own top-level folder.

Recommended layout:

- `backend-*/` or `services/*/` for backend services.
- `frontend/` for the web client.
- `shared/` for code, schemas, or contracts intentionally reused across apps.
- `docs/` for design notes and operational documentation.
- App-local `testdata/` folders for fixtures.

Keep app-specific code, tests, and configuration near the app that owns them. Keep binaries, coverage files, dependency folders, and local environment files out of Git.

## Build, Test, and Development Commands

There are no project-specific commands yet. Add commands in each app folder and document them in that app’s README:

- `cd backend-api && go test ./...` runs Go backend tests.
- `cd backend-api && go build ./...` verifies Go packages compile.
- `cd frontend && npm install` installs frontend dependencies.
- `cd frontend && npm run dev` starts the frontend dev server.
- `cd frontend && npm test` runs frontend tests.

If root orchestration is added, prefer a `Makefile`, `docker-compose.yml`, or workspace script with targets like `make test`, `make build`, and `make dev`.

## Coding Style & Naming Conventions

Match each app’s language conventions. For Go backends, run `gofmt`; package names should be short and lowercase, such as `collector`, `metrics`, or `dockerapi`. For frontend code, use the formatter and linter configured in `frontend/`.

Prefer small modules with explicit responsibilities. Use uppercase environment variables, for example `DOCKER_HOST`, `METRICS_PORT`, or `VITE_API_URL`.

## Testing Guidelines

Place tests next to implementation unless a framework expects a dedicated test folder. Use Go’s `*_test.go` convention for Go services. For frontend code, follow conventions such as `*.test.ts` or `*.spec.ts`.

Run the relevant app-level test command before opening a pull request. Add focused tests for new behavior and bug fixes, especially around shared API contracts.

## Commit & Pull Request Guidelines

The Git history contains only `Initial commit`, so no detailed convention is established. Use concise imperative subjects, for example `Add Docker metrics collector` or `Document local development setup`.

Pull requests should include a short summary, affected apps, tests run, linked issues, and screenshots or logs for user-visible behavior. Keep PRs focused.

## Security & Configuration Tips

Do not commit `.env`, credentials, Docker socket tokens, or production endpoint details. Provide examples in documentation and keep real configuration local or in deployment secrets.
