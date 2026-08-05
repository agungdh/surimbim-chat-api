# AGENTS.md

Go backend (Go 1.26, module `surimbim-chat-api`) — chi router, uptrace/bun + SQLite, gorilla/websocket (STOMP over WS), goose migrations (embedded, auto-run on boot).

## Commands

- `make run` — `go run ./main.go` (server on :8080)
- `make build` — build binary; `make build-prod` — optimized; `make clean` — remove binary; `make help` — list targets
- `go build ./...` / `go vet ./...` — verify build (CI runs `go vet ./...` then `go build -o surimbim-chat-api .`)
- **No tests exist** in the repo (`*_test.go`); there is no test command.

## Go toolchain

- `go.mod` requires **go 1.26**. The system `go` in PATH may be older (auto-download of toolchain can fail offline).
- GoLand installs its own SDKs under **the user's home dir** (`~/sdk/<version>/bin/go`, e.g. `~/sdk/go1.26.5/bin/go`). This is per-user (`$HOME`), not a fixed path — locate it there, not under /usr/local.
- When the default `go` is too old, build/run with that SDK:
  ```sh
  export PATH="$HOME/sdk/go1.26.5/bin:$PATH"   # or GOTOOLCHAIN=local
  make run
  ```

## Notes

- Config via env, loaded by `godotenv` from `.env` (optional; all keys have defaults): `PORT` (8080), `DB_PATH` (surimbim.db), `ENV`, `CORS_ORIGINS`, `SESSION_TTL`. See `internal/config/config.go`; example in committed `.env`.
- CORS: credential-bearing headers are sent only when the request `Origin` is same-host or in the `CORS_ORIGINS` allowlist (comma-separated). The WS upgrade uses the same allowlist via `mw.OriginAllowed`. Cross-origin clients not on the list get no CORS headers.
- `POST /api/login` is rate-limited to **10/min per client IP** (in-memory); repeated bad logins return 429.
- Auth: HttpOnly cookie `token`; WS handshake reads the cookie. `GET /api/users/active` is intentionally **unauthenticated** (presence seed).
- Presence: server broadcasts `/topic/presence` on online/offline **transitions** only; client must seed initial state from `GET /api/users/active`.
- History: `GET` via WS `/app/history` (cursor pagination, `limit` header, max 100).
