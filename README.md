# api-gateway

Entry point for all Snaply client traffic. Handles JWT validation, per-IP rate limiting, reverse proxying to backend services, and the WebSocket hub.

## Environment Variables

| Variable                    | Default                        | Description                            |
|-----------------------------|--------------------------------|----------------------------------------|
| `SERVER_PORT`               | `8080`                         | HTTP listen port                       |
| `JWT_SECRET`                | `dev_secret_change_in_production` | Must match user-service's secret    |
| `INTERNAL_SECRET`           | `dev_internal_secret_change_in_production` | Must match notification-service's and message-service's secret |
| `USER_SERVICE_URL`          | `http://localhost:8081`        | user-service base URL                  |
| `POST_SERVICE_URL`          | `http://localhost:8082`        | post-service base URL                  |
| `RELATION_SERVICE_URL`      | `http://localhost:8083`        | relation-service base URL              |
| `LIKE_SERVICE_URL`          | `http://localhost:8084`        | like-service base URL                  |
| `NOTIFICATION_SERVICE_URL`  | `http://localhost:8085`        | notification-service base URL          |
| `MESSAGE_SERVICE_URL`       | `http://localhost:8087`        | message-service base URL               |
| `RATE_LIMIT_RPS`            | `100`                          | Token bucket refill rate (req/sec)     |
| `RATE_LIMIT_BURST`          | `200`                          | Token bucket burst capacity            |

## Route Table

| Method | Path                        | Auth    | Proxied To            |
|--------|------------------------------|---------|-------------------------|
| GET    | /health                      | —       | (inline)                |
| POST   | /api/v1/auth/*                | —       | user-service             |
| POST   | /internal/ws/send             | `X-Internal-Secret` | (inline hub) |
| GET    | /api/v1/users/*                | JWT     | user-service             |
| PUT    | /api/v1/users/*                | JWT     | user-service             |
| *      | /api/v1/posts/*                | JWT     | post-service             |
| *      | /api/v1/comments/*             | JWT     | post-service             |
| GET    | /api/v1/feed                   | JWT     | post-service             |
| *      | /api/v1/relations/*            | JWT     | relation-service         |
| *      | /api/v1/likes/*                | JWT     | like-service             |
| *      | /api/v1/notifications          | JWT     | notification-service     |
| *      | /api/v1/conversations           | JWT     | message-service          |
| *      | /api/v1/conversations/*         | JWT     | message-service          |
| *      | /api/v1/messages/*              | JWT     | message-service          |
| GET    | /ws                             | JWT     | (inline hub)             |

## How to Run Locally

```bash
# Copy and edit env
cp ../infra/.env.example .env

# Install deps
make setup

# Run (expects user-service and post-service already running)
make run
```

## WebSocket

Connect with a valid access token:

```
ws://localhost:8080/ws?token=<access_token>
```

Or via `Authorization: Bearer <token>` header. One connection per `user_id`; a new connection replaces the old one.
