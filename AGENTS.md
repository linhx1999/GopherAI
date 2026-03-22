# Repository Guidelines

## Project Structure & Module Organization
`main.go` boots the Gin server. `controller/` handles requests and validation, `service/` owns business orchestration, `dao/` wraps PostgreSQL and Redis access, `model/` defines entities, `router/` registers routes, and `middleware/` contains middleware. Shared integrations live under `common/` (LLM, MCP, DeepAgent, storage, cache, messaging). The frontend is isolated in `frontend/`; routes live in `frontend/src/router`, views in `frontend/src/views`, components in `frontend/src/components`, and helpers in `frontend/src/utils`. Keep environment examples in `.env.example`, and Docker assets in `docker/` plus `docker-compose.yml`.

## Documentation Maintenance
- Documentation must be updated synchronously when code behavior changes.
- Update `README.md` for user-facing changes.
- Update `AGENTS.md` for developer-facing changes.
- Documentation content must remain consistent with the current implementation.

## Build, Test, and Development Commands
- `docker compose up -d`: start local dependencies such as PostgreSQL, Redis, RabbitMQ, and MinIO.
- `go mod download && go run main.go`: run the backend locally on `http://localhost:9090`.
- `go test ./...`: run all backend tests.
- `go build -o gopherai main.go`: build the backend binary.
- `cd frontend && pnpm install && pnpm dev`: start the Vite frontend on `http://localhost:5173`.
- `cd frontend && pnpm build`: create a production frontend build.
- `cd frontend && pnpm lint`: run ESLint for the React app.

## Coding Style & Naming Conventions
Format Go code with `gofmt` and keep package names lowercase. Respect the existing layering: request parsing in `controller`, business rules in `service`, persistence in `dao`. Prefer clear exported names and keep JSON field names aligned with the current API. In the frontend, use `PascalCase` for React components and `camelCase` for utilities and helpers. Follow `frontend/eslint.config.js`; fix lint issues rather than silencing them.

## Testing Guidelines
Backend tests use Go's built-in `testing` package and sit next to the code as `*_test.go`. Add focused regression tests for changed packages, especially stream handlers, service flows, and shared utilities in `common/`. No dedicated frontend test runner is configured yet, so UI changes should at minimum pass `pnpm lint` and `pnpm build`.

## Commit & Pull Request Guidelines
Use Conventional Commits, matching the current history: `feat(mcp): ...`, `fix(chat): ...`, `docs(readme): ...`, `refactor(deep-agent): ...`. Keep each commit scoped to one logical change. Pull requests should explain user-visible behavior, list config or schema changes, link related issues, and include screenshots or request/response examples when UI or API behavior changes.

## Security & Configuration Tips
Copy `.env.example` to `.env` for local setup and never commit real secrets. Treat `MCP_SECRET_KEY`, JWT settings, OpenAI credentials, database passwords, and MinIO keys as sensitive. If you test DeepAgent container flows, build `docker/deepagent/Dockerfile` first and document new runtime requirements in `README.md` and this file.
