# api-gateway

Entry point for all Snaply client traffic. Go 1.22, port 8080. Handles JWT validation, per-IP rate limiting, reverse proxying, and the WebSocket hub. Does not talk to any database directly.

## Layout

`cmd/main.go` → `internal/middleware/auth.go` (JWT validation) + `internal/middleware/ratelimit.go` (per-IP token bucket) → `internal/proxy/proxy.go` (reverse proxy to backend services) → `internal/ws/hub.go` (WebSocket connections)

## Route table

| Method | Path                        | Auth | Proxied to       |
|--------|------------------------------|------|-------------------|
| GET    | /health                     | —    | inline            |
| POST   | /api/v1/auth/*              | —    | user-service      |
| POST   | /api/v1/users/{id}/posts       | JWT | post-service (registered before the `/users/*` wildcard, which would otherwise swallow it) |
| GET    | /api/v1/users/{id}/posts/count | JWT | post-service (same reason — a literal sub-path registered ahead of the `/users/*` wildcard) |
| GET/PUT| /api/v1/users/*             | JWT  | user-service      |
| *      | /api/v1/posts               | JWT  | post-service      |
| *      | /api/v1/posts/*             | JWT  | post-service      |
| *      | /api/v1/comments/*          | JWT  | post-service      |
| GET    | /api/v1/feed                | JWT  | post-service      |
| *      | /api/v1/relations/*         | JWT  | relation-service  |
| *      | /api/v1/likes/*             | JWT  | like-service      |
| GET    | /ws                         | JWT  | inline hub        |

## Conventions

- `JWT_SECRET` must match `user-service` exactly — this is the only place tokens are validated; downstream services trust the `X-User-ID` header this gateway injects.
- Rate limiting is a token bucket per client IP: `RATE_LIMIT_RPS` (refill rate) / `RATE_LIMIT_BURST` (default 100/200).
- WebSocket: one connection per `user_id` — a new connection from the same user replaces the old one. Auth via `?token=` query param or `Authorization: Bearer` header.
- Backend URLs configured via `USER_SERVICE_URL` (default `http://localhost:8081`), `POST_SERVICE_URL` (default `http://localhost:8082`), `RELATION_SERVICE_URL` (default `http://localhost:8083`), and `LIKE_SERVICE_URL` (default `http://localhost:8084`).
- **chi wildcard gotcha:** `r.Handle("/foo/*", h)` does NOT match the bare `/foo` (no trailing segment) — that's why `/api/v1/posts` needs its own explicit route alongside `/api/v1/posts/*`, and why `/api/v1/users/{id}/posts/count` needs its own route ahead of the `/api/v1/users/*` wildcard (which would otherwise route it to user-service instead of post-service). Keep this in mind when adding new proxied paths — a sub-path one level under something already routed elsewhere needs an explicit, more-specific route.

## Running

Expects `user-service` and `post-service` already running.

```bash
cp ../infra/.env.example .env
make setup
make run
```
