# api-gateway

Entry point for all Snaply client traffic. Handles JWT validation, per-IP rate limiting, reverse proxying to backend services, and the WebSocket hub.

## Environment Variables

| Variable            | Default                        | Description                            |
|---------------------|--------------------------------|----------------------------------------|
| `SERVER_PORT`       | `8080`                         | HTTP listen port                       |
| `JWT_SECRET`        | `dev_secret_change_in_production` | Must match user-service's secret    |
| `USER_SERVICE_URL`  | `http://localhost:8081`        | user-service base URL                  |
| `POST_SERVICE_URL`  | `http://localhost:8082`        | post-service base URL                  |
| `RATE_LIMIT_RPS`    | `100`                          | Token bucket refill rate (req/sec)     |
| `RATE_LIMIT_BURST`  | `200`                          | Token bucket burst capacity            |

## Route Table

| Method | Path                     | Auth    | Proxied To      |
|--------|--------------------------|---------|-----------------|
| GET    | /health                  | —       | (inline)        |
| POST   | /api/v1/auth/*           | —       | user-service    |
| GET    | /api/v1/users/*          | JWT     | user-service    |
| PUT    | /api/v1/users/*          | JWT     | user-service    |
| *      | /api/v1/posts/*          | JWT     | post-service    |
| *      | /api/v1/comments/*       | JWT     | post-service    |
| GET    | /api/v1/feed             | JWT     | post-service    |
| GET    | /ws                      | JWT     | (inline hub)    |

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
