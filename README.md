# DocflowWeb

DocflowWeb is a backend system for internal document workflow management, built in Go using a microservices architecture. The system covers the full document lifecycle from creation and assignment through deadline tracking, task management, calendar scheduling, and email notification delivery.

## Table of Contents

- [Architecture](#architecture)
- [Services](#services)
- [Technology Stack](#technology-stack)
- [Database Design](#database-design)
- [API Overview](#api-overview)
- [Authentication and Authorization](#authentication-and-authorization)
- [Inter-Service Communication](#inter-service-communication)
- [Infrastructure](#infrastructure)
- [Getting Started](#getting-started)
- [Environment Variables](#environment-variables)
- [Ports Reference](#ports-reference)
- [Authors](#authors)

---

## Architecture

DocflowWeb follows a microservices pattern with the following structure:

```
Client
  |
  | HTTP REST
  v
api-gateway  (:8080)
  |
  | gRPC
  +---> auth-service         (:9001)
  +---> user-service         (:9002)
  +---> document-service     (:9003)
  +---> task-service         (:9004)
  +---> calendar-service     (:9005)
  +---> notification-service (:9006)
  +---> mail-service         (:9007)
        |
        | NATS
        v
     notification-service <--> mail-service

All services share:
  - PostgreSQL (one container, separate databases per service)
  - Redis      (one container, separate DB index per service)
  - NATS       (one container, subject-based routing)
```

Each service is independently deployed, has its own database, and communicates with other services exclusively via gRPC or NATS. The API Gateway is the single entry point for all external HTTP traffic.

---

## Services

### api-gateway
Receives all external HTTP requests, validates JWT tokens by calling auth-service, enforces rate limiting and idempotency via Redis, and proxies requests to the appropriate backend service via gRPC. Serves the frontend static interface.

### auth-service
Handles user registration, login, token issuance, refresh token rotation, email verification, and password reset. Issues JWT access tokens (15 minutes) and refresh tokens (7 days stored in Redis). Calls user-service to create a user profile after registration and calls notification-service to deliver verification and reset emails.

### user-service
Manages user profiles, roles, banning, and statistics. Maintains a separate database synchronized with auth-service via gRPC on registration. Provides batch lookup and streaming watch endpoint for downstream consumers.

### document-service
Core business domain. Manages documents with full lifecycle support:
- Statuses: `draft`, `assigned`, `in_progress`, `completed`, `overdue`, `archived`
- Status transitions are enforced by a state machine in the service layer
- Every transition is recorded in `document_history`
- Calls notification-service on status change and assignment
- Calls calendar-service when a deadline is set or updated
- Exports document lists to CSV

### task-service
Manages tasks linked to documents. Supports assignment, status changes, priority levels, and history. Calls notification-service on assignment and calls calendar-service when a task deadline is set. Includes an overdue detection endpoint intended for scheduled invocation.

### calendar-service
Stores and retrieves calendar events. Supports queries by day, week, month, and upcoming deadlines. Results are cached in Redis with a 5-minute TTL. Cache is invalidated on write operations.

### notification-service
Creates and delivers in-app notifications. Manages user notification preferences and templates. Provides a gRPC streaming endpoint for real-time delivery. Publishes email jobs to NATS for async processing by mail-service. Includes a scheduler that runs daily to send deadline reminders and detect overdue documents.

### mail-service
Consumes email jobs from NATS and sends emails via SMTP. Maintains a mail job log in its own database. Supports bulk sending and template management. Metrics are exposed for Prometheus scraping.

---

## Technology Stack

| Component | Technology |
|---|---|
| Language | Go 1.22 |
| HTTP framework | Gin |
| Service transport | gRPC (protobuf) |
| Async messaging | NATS |
| Database | PostgreSQL 16 |
| Cache / sessions | Redis 7 |
| Auth | JWT (golang-jwt/jwt/v5) |
| Email | SMTP via net/smtp |
| Observability | Prometheus + Grafana |
| Containerization | Docker, Docker Compose |
| Frontend | Static HTML + JavaScript (served by api-gateway) |
| Proto generation | protoc-gen-go, protoc-gen-go-grpc |
| CI | GitHub Actions |

---

## Database Design

One PostgreSQL container is used. Each service has its own isolated database created during container initialization.

| Database | Owner service | Key tables |
|---|---|---|
| `auth_db` | auth-service | `auth_users`, `refresh_tokens`, `email_verifications`, `password_resets` |
| `user_db` | user-service | `users`, `user_stats` |
| `doc_db` | document-service | `documents`, `document_history` |
| `task_db` | task-service | `tasks`, `task_history` |
| `cal_db` | calendar-service | `events` |
| `notif_db` | notification-service | `notifications`, `notification_preferences`, `notification_templates` |
| `mail_db` | mail-service | `mail_jobs` |

### documents
```
id, title, description, type, status, creator_id, responsible_id,
deadline, file_url, tags, is_overdue, created_at, updated_at, archived_at
```

Status enum: `draft` → `assigned` → `in_progress` → `completed` / `overdue` → `archived`

### tasks
```
id, document_id, title, description, status, priority,
creator_id, assignee_id, deadline, is_overdue, created_at, updated_at, completed_at
```

### events (calendar)
```
id, user_id, title, description, event_type, ref_id, ref_type,
start_time, end_time, created_at, updated_at
```

### notifications
```
id, user_id, document_id, task_id, notif_category, title, body,
ref_id, ref_type, is_read, sent_email, created_at, updated_at, read_at, deleted_at
```

---

## API Overview

All endpoints are under `/api/v1`. Public endpoints do not require authentication. All other endpoints require a valid JWT in the `Authorization: Bearer <token>` header.

### Auth (public)
```
POST   /api/v1/auth/register
POST   /api/v1/auth/login
POST   /api/v1/auth/refresh
```

### Auth (protected)
```
POST   /api/v1/auth/logout
POST   /api/v1/auth/verify-email
POST   /api/v1/auth/forgot-password
POST   /api/v1/auth/reset-password
POST   /api/v1/auth/change-password
```

### Users
```
GET    /api/v1/users
GET    /api/v1/users/:id
PATCH  /api/v1/users/:id
DELETE /api/v1/users/:id
POST   /api/v1/users/:id/ban        (admin, manager)
```

### Documents
```
GET    /api/v1/documents
POST   /api/v1/documents
GET    /api/v1/documents/:id
PATCH  /api/v1/documents/:id
DELETE /api/v1/documents/:id
POST   /api/v1/documents/:id/assign
POST   /api/v1/documents/:id/status
POST   /api/v1/documents/:id/archive    (admin, manager)
GET    /api/v1/documents/:id/history
POST   /api/v1/documents/filter
POST   /api/v1/documents/overdue        (admin, manager)
GET    /api/v1/documents/export/csv
```

### Tasks
```
GET    /api/v1/tasks/:id
POST   /api/v1/tasks
PATCH  /api/v1/tasks/:id
DELETE /api/v1/tasks/:id
POST   /api/v1/tasks/:id/assign
POST   /api/v1/tasks/:id/status
GET    /api/v1/tasks/:id/history
GET    /api/v1/tasks/document/:document_id
GET    /api/v1/tasks/assignee/:assignee_id
POST   /api/v1/tasks/filter
POST   /api/v1/tasks/overdue            (admin, manager)
```

### Calendar
```
POST   /api/v1/events
GET    /api/v1/events/day
GET    /api/v1/events/week
GET    /api/v1/events/month
GET    /api/v1/events/upcoming
GET    /api/v1/events/user/:user_id
PATCH  /api/v1/events/:id
DELETE /api/v1/events/:id
POST   /api/v1/events/filter
```

### Notifications
```
GET    /api/v1/notifications
GET    /api/v1/notifications/unread-count
POST   /api/v1/notifications/:id/read
POST   /api/v1/notifications/read-all
DELETE /api/v1/notifications/:id
GET    /api/v1/notifications/templates/:id
POST   /api/v1/notifications/preferences
GET    /api/v1/notifications/preferences
```

### Mail
```
POST   /api/v1/mail/send              (admin)
POST   /api/v1/mail/send-bulk         (admin)
GET    /api/v1/mail/jobs
GET    /api/v1/mail/jobs/:id
```

### System
```
GET    /health
```

---

## Authentication and Authorization

Authentication uses JWT. The access token has a 15-minute lifetime. The refresh token has a 7-day lifetime and is stored in Redis.

Roles:

| Role | Access |
|---|---|
| `employee` | Own documents and tasks, read own notifications and calendar |
| `manager` | All documents and tasks, assignment, status changes |
| `admin` | Full access, user management, bulk operations |

The API Gateway validates the JWT on every protected request by calling `auth-service.ValidateToken` via gRPC. The user ID and role extracted from the token are forwarded to downstream services via request context.

---

## Inter-Service Communication

All synchronous calls between services use gRPC. Asynchronous email delivery uses NATS.

| Caller | Callee | Trigger |
|---|---|---|
| api-gateway | auth-service | Every protected request (ValidateToken) |
| auth-service | user-service | After registration (CreateUser) |
| auth-service | notification-service | Email verification, password reset |
| document-service | notification-service | Status change, assignment |
| document-service | calendar-service | Deadline set or updated |
| task-service | notification-service | Task assigned, status changed |
| task-service | calendar-service | Task deadline set |
| notification-service | mail-service | Via NATS (queue.mail.jobs) |

Proto definitions are maintained in a separate repository: [docflow-protos-final](https://github.com/Temych228/docflow-protos-final).

---

## Infrastructure

### Redis key usage

| Key pattern | Purpose | TTL |
|---|---|---|
| `auth:refresh:{token_hash}` | Refresh token storage | 7 days |
| `auth:session:{user_id}` | Session cache | 15 minutes |
| `idempotency:{key}` | Request deduplication | 24 hours |
| `ratelimit:{ip}:{route}` | Rate limit counter | 1 minute |
| `notif:unread:{user_id}` | Unread notification count | 5 minutes |
| `calendar:{user_id}:{year}{month}` | Monthly event cache | 5 minutes |

### Rate limiting

Sliding window, 100 requests per minute per IP address, enforced in api-gateway using Redis. Exceeding the limit returns HTTP 429.

### Idempotency

All POST, PATCH, and PUT requests support the `Idempotency-Key` header. When provided, the gateway stores the response in Redis for 24 hours. Duplicate requests with the same key return the cached response without calling the backend.

### Error response format

All errors follow a consistent structure:

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "Document not found",
    "details": {}
  },
  "request_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

### CI/CD

GitHub Actions runs on every push and pull request to `main`. Each service is built and tested independently in parallel. The pipeline runs: dependency download, module verification, formatting check (`gofmt`), static analysis (`go vet`), build, and tests.

---

## Getting Started

### Requirements

- Docker Engine 24 or later
- Docker Compose v2 or later
- Git

### Clone the repository

```bash
git clone https://github.com/Temych228/DocflowWeb.git
cd DocflowWeb
```

### Configure environment

Copy the root environment file and fill in required values:

```bash
cp .env.example .env
```

Open `.env` and set at minimum:

```
JWT_SECRET=<your-random-secret>
SMTP_USERNAME=<your-email>
SMTP_PASSWORD=<your-app-password>
SMTP_FROM=<your-email>
```

All other values have safe defaults for local development. Individual service `.env.example` files exist under each `services/<name>/` directory but are not required for Docker Compose startup since the root `.env` is loaded by the compose file.

### First-time startup

```bash
docker compose up --build -d
```

This builds all service images, creates the PostgreSQL databases via initialization scripts, applies all migrations, and starts the full stack.

To confirm all containers are running:

```bash
docker compose ps
```

To stream logs for all services:

```bash
docker compose logs -f
```

To stream logs for a specific service:

```bash
docker compose logs -f document-service
```

### Verify the stack is up

```bash
curl http://localhost:8080/health
```

Expected response:

```json
{"data": {"status": "ok", "service": "api-gateway"}}
```

### Access the frontend

Open in a browser:

```
http://localhost:8080
```

### Stop the stack

```bash
docker compose down
```

To stop and remove all persistent data (full reset):

```bash
docker compose down -v
```

### Rebuild a single service after code changes

```bash
docker compose up --build -d document-service
```

---

## Environment Variables

The following variables are read from the root `.env` file and injected into each service container by Docker Compose.

| Variable | Description | Default |
|---|---|---|
| `POSTGRES_USER` | PostgreSQL username | `dms` |
| `POSTGRES_PASSWORD` | PostgreSQL password | `dms` |
| `POSTGRES_DB` | PostgreSQL default database | `postgres` |
| `REDIS_PASSWORD` | Redis authentication password | `dms_redis` |
| `JWT_SECRET` | Secret key for JWT signing | required |
| `ACCESS_TOKEN_TTL` | Access token lifetime | `15m` |
| `REFRESH_TOKEN_TTL` | Refresh token lifetime | `168h` |
| `SMTP_HOST` | SMTP server hostname | `smtp.gmail.com` |
| `SMTP_PORT` | SMTP server port | `587` |
| `SMTP_USERNAME` | SMTP login | required for email |
| `SMTP_PASSWORD` | SMTP password or app password | required for email |
| `SMTP_FROM` | Sender email address | required for email |
| `FRONTEND_URL` | Base URL used in email links | `http://localhost:3000` |
| `RATE_LIMIT_RPM` | API Gateway rate limit (requests/min) | `100` |
| `NATS_URL` | NATS server connection string | `nats://nats:4222` |
| `GF_ADMIN_USER` | Grafana admin username | `admin` |
| `GF_ADMIN_PASSWORD` | Grafana admin password | `admin` |

---

## Ports Reference

| Service | HTTP | gRPC | Metrics |
|---|---|---|---|
| api-gateway | 8080 | — | — |
| auth-service | 8082 | 9001 | 9202 |
| user-service | 8081 | 9002 | 9201 |
| document-service | 8083 | 9003 | 9203 |
| task-service | 8084 | 9004 | 9204 |
| calendar-service | 8085 | 9005 | 9205 |
| notification-service | 8086 | 9006 | 9206 |
| mail-service | 8087 | 9007 | 9207 |
| PostgreSQL | 5432 | — | — |
| Redis | 6379 | — | — |
| NATS | 4222 | — | 8222 |
| Prometheus | 9090 | — | — |
| Grafana | 3001 | — | — |

---

## Authors

Built by **Artyom Safaryan**
