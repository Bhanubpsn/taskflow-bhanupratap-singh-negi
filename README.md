# TaskFlow

A REST API for task and project management built with Go.

---

## 1. Overview

TaskFlow is a backend API that lets users create projects, manage tasks within those projects, and assign tasks to other users. Authentication is JWT-based; all non-auth endpoints require a Bearer token.

**Stack**

| Layer            | Technology                                                             |
| ---------------- | ---------------------------------------------------------------------- |
| Language         | Go 1.25                                                                |
| Web framework    | [Gin](https://github.com/gin-gonic/gin)                                |
| Database         | PostgreSQL 16                                                          |
| Driver           | [pgx/v5](https://github.com/jackc/pgx) (pgxpool)                       |
| Migrations       | [golang-migrate/migrate](https://github.com/golang-migrate/migrate)    |
| Auth             | JWT (HS256) via [golang-jwt/jwt/v5](https://github.com/golang-jwt/jwt) |
| Password hashing | bcrypt (cost 12)                                                       |
| Containerisation | Docker + Docker Compose                                                |

---

## 2. Architecture Decisions

**Raw SQL over ORM**
Every query is hand-written SQL. ORMs obscure what hits the database, make it harder to reason about performance, and tend to generate unsafe migrations. With raw SQL and `golang-migrate`, every schema change is an explicit, versioned, reversible file.

**`pgxpool` over `database/sql`**
`pgxpool` is the idiomatic connection pool for `pgx`. It supports native PostgreSQL types (UUID, enums, arrays) without driver-level scanning hacks that `database/sql` requires.

**Schema enums in PostgreSQL, not just Go**
`task_status` and `task_priority` are defined as Postgres `ENUM` types, not plain `TEXT` columns with application-level validation. The database rejects invalid values at the constraint level — bad data cannot sneak in through a raw SQL client or a future service that skips validation.

**Migrations run at startup**
`RunMigrations` is called in `main()` before the router starts. This means the schema is always up-to-date on any fresh environment with zero manual steps. `golang-migrate` is idempotent — it tracks the current version in `schema_migrations` and only applies pending files.

**401 vs 403 are intentionally distinct**

- `401 Unauthorized` — no token, expired token, or invalid signature. The caller is not authenticated.
- `403 Forbidden` — valid token, but the caller does not own the resource (e.g. trying to delete someone else's project). These are different failure modes and should not be conflated.

**`GET /projects/:id` returns 404 for non-members**
Returning 403 when a user tries to access a project they are not a member of would confirm the project exists. Instead, we return 404 — the project is invisible to non-members.

**What was intentionally left out**

- Refresh tokens — out of scope for this exercise; the 24-hour access token is sufficient.
- Pagination — not in the spec; straightforward to add with `LIMIT`/`OFFSET`.
- Email verification — not required.
- Tests — not part of the stated requirements.

---

## 3. Running Locally

The only prerequisite is Docker with the Compose plugin.

```bash
git clone https://github.com/Bhanubpsn/taskflow-bhanupratap-singh-negi
cd taskflow-bhanupratap-singh-negi/backend
cp .env.example .env
docker compose up --build
```

The API will be available at `http://localhost:3000`.

What happens on startup:

1. Postgres container starts and passes its health check (`pg_isready`)
2. API container starts only after Postgres is healthy
3. `golang-migrate` applies any pending migrations
4. Gin registers all routes and begins serving

---

## 4. Running Migrations

Migrations run **automatically** on every startup — no manual step required.

If you ever need to run them manually (e.g. against a local Postgres outside Docker):

```bash
# Install migrate CLI
go install -tags 'pgx5' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Apply all migrations
migrate -path ./migrations -database "pgx5://taskflow:taskflow@localhost:5432/taskflow?sslmode=disable" up

# Roll back the last migration
migrate -path ./migrations -database "pgx5://taskflow:taskflow@localhost:5432/taskflow?sslmode=disable" down 1
```

---

## 5. Test Credentials

Run the seed after the stack is up:

```bash
docker compose up --build -d
docker compose run --rm seed
```

This inserts 1 user, 1 project, and 3 tasks (one per status) into the database. The seed is idempotent — safe to run multiple times.

**Credentials**

```
Email:    test@example.com
Password: password123
```

Then log in to get your token:

```bash
curl -s -X POST http://localhost:3000/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}' \
  | jq .
```

Copy the `token` value and use it as `Authorization: Bearer <token>` on all subsequent requests.

---

## 6. API Reference

### Auth

All auth endpoints are public (no token required).

#### `POST /auth/register`

```http
POST /auth/register
Content-Type: application/json

{
  "name": "Bhanu",
  "email": "bhanu@example.com",
  "password": "password123"
}
```

```json
// 201 Created
{
  "id": "a1b2c3d4-...",
  "name": "Bhanu",
  "email": "bhanu@example.com",
  "created_at": "2026-04-12T10:00:00Z"
}
```

| Code  | Reason                   |
| ----- | ------------------------ |
| `201` | User created             |
| `400` | Missing/invalid fields   |
| `409` | Email already registered |

---

#### `POST /auth/login`

```http
POST /auth/login
Content-Type: application/json

{
  "email": "bhanu@example.com",
  "password": "password123"
}
```

```json
// 200 OK
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

| Code  | Reason                  |
| ----- | ----------------------- |
| `200` | Token returned          |
| `401` | Wrong email or password |

---

### Projects

All project endpoints require `Authorization: Bearer <token>`.

#### `GET /projects`

Returns projects the authenticated user owns **or** is assigned to a task in.

```http
GET /projects
Authorization: Bearer <token>
```

```json
// 200 OK
[
  {
    "id": "...",
    "name": "TaskFlow Backend",
    "description": "Building the API",
    "owner_id": "...",
    "created_at": "2026-04-12T10:00:00Z"
  }
]
```

---

#### `POST /projects`

```http
POST /projects
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "TaskFlow Backend",
  "description": "Building the API"
}
```

```json
// 201 Created
{
  "id": "...",
  "name": "TaskFlow Backend",
  "description": "Building the API",
  "owner_id": "...",
  "created_at": "2026-04-12T10:00:00Z"
}
```

`description` is optional.

---

#### `GET /projects/:id`

Returns the project and all its tasks. Visible only to the owner or users assigned to a task in the project.

```json
// 200 OK
{
  "id": "...",
  "name": "TaskFlow Backend",
  "description": "Building the API",
  "owner_id": "...",
  "created_at": "...",
  "tasks": [
    {
      "id": "...",
      "title": "Write controllers",
      "status": "in_progress",
      "priority": "high",
      ...
    }
  ]
}
```

| Code  | Reason                    |
| ----- | ------------------------- |
| `200` | Project + tasks returned  |
| `404` | Not found or not a member |

---

#### `PATCH /projects/:id`

Owner only. Send only the fields you want to change.

```http
PATCH /projects/:id
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "New Name",
  "description": "Updated description"
}
```

| Code  | Reason                   |
| ----- | ------------------------ |
| `200` | Updated project returned |
| `403` | Not the owner            |
| `404` | Project not found        |

---

#### `DELETE /projects/:id`

Owner only. Deletes the project and all its tasks (cascade).

| Code  | Reason                           |
| ----- | -------------------------------- |
| `200` | `{"message": "project deleted"}` |
| `403` | Not the owner                    |
| `404` | Project not found                |

---

### Tasks

All task endpoints require `Authorization: Bearer <token>`.

#### `GET /projects/:id/tasks`

Supports optional query filters:

```http
GET /projects/:id/tasks?status=in_progress&assignee=<uuid>
```

`status` values: `todo` · `in_progress` · `done`

```json
// 200 OK
[
  {
    "id": "...",
    "title": "Write controllers",
    "description": null,
    "status": "in_progress",
    "priority": "high",
    "project_id": "...",
    "assignee_id": null,
    "due_date": null,
    "created_at": "...",
    "updated_at": "...",
    "created_by": "..."
  }
]
```

---

#### `POST /projects/:id/tasks`

Project owner only.

```http
POST /projects/:id/tasks
Authorization: Bearer <token>
Content-Type: application/json

{
  "title": "Write controllers",
  "description": "Implement all CRUD handlers",
  "status": "todo",
  "priority": "high",
  "assignee_id": "<user-uuid>",
  "due_date": "2026-05-01T00:00:00Z"
}
```

`description`, `status`, `priority`, `assignee_id`, and `due_date` are all optional.
Defaults: `status = todo`, `priority = medium`.

| Code  | Reason                |
| ----- | --------------------- |
| `201` | Task created          |
| `403` | Not the project owner |
| `404` | Project not found     |

---

#### `PATCH /tasks/:id`

Any authenticated user. Send only the fields to update.

```http
PATCH /tasks/:id
Authorization: Bearer <token>
Content-Type: application/json

{
  "status": "done"
}
```

`priority` values: `low` · `medium` · `high`

| Code  | Reason                |
| ----- | --------------------- |
| `200` | Updated task returned |
| `400` | No fields provided    |
| `404` | Task not found        |

---

#### `DELETE /tasks/:id`

Project owner **or** task creator only.

| Code  | Reason                        |
| ----- | ----------------------------- |
| `200` | `{"message": "task deleted"}` |
| `403` | Neither owner nor creator     |
| `404` | Task not found                |

## 7. What You'd Do With More Time

### Load Balancer

I engineered a smiliar personal project of mine in which I coded a custom load balancer to distribute the user respuests to multiple local servers using round robin. By doing this the traffic can be distributed among the servers hence faster replies.

### Rate limitter

In that project I also engineerd a custom rate limtter for each user in front of the load balancer, so that the requests are rate limitted before being distributed hence reducing sudden server loads and DDOS.

### Message Broker

I also designed my very own message broker from scratch in that project, in this project I would have used it in the task completion notification system. If an assignee completed a task, an automated worker will push a mail to the project owner's email notifying them that, this particular task has been completed.
