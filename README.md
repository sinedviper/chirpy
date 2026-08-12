# Chirpy

A Twitter/X-like microblogging HTTP API written in Go, backed by PostgreSQL. Users can register, log in, post short "chirps" (max 140 characters), and manage them, with JWT-based authentication and refresh tokens. Includes a Polka webhook endpoint for upgrading users to "Chirpy Red".

## Tech stack

- **Go** (`net/http`, standard library only — no routing framework)
- **PostgreSQL** as the database
- **[sqlc](https://sqlc.dev/)** — generates type-safe Go code from SQL (`internal/database/`, do not edit by hand)
- **[goose](https://github.com/pressly/goose)** — SQL schema migrations
- **JWT** ([golang-jwt/jwt/v5](https://github.com/golang-jwt/jwt)) — HS256 access tokens (1h expiry)
- **Refresh tokens** — random 32-byte hex tokens stored in the DB (60-day expiry, revocable)
- **[argon2id](https://github.com/alexedwards/argon2id)** — password hashing
- **[godotenv](https://github.com/joho/godotenv)** — loads config from `.env`

## Project structure

```
main.go                      # entry point, route wiring
internal/
  admin/                     # health check, metrics, dev-only reset
  auth/                      # login, refresh, revoke, JWT + password helpers
  chirps/                    # create/list/get/delete chirps
  users/                     # create/update user
  webhooks/                  # Polka "user upgraded" webhook
  middleware/                # hit counter + JWT auth middleware
  response/                  # shared JSON/text HTTP response helpers
  database/                  # sqlc-generated DB code (do not edit)
sql/
  schema/                    # goose migrations
  queries/                   # sqlc named queries
```

## Setup

1. Install dependencies: `go mod download`
2. Create a PostgreSQL database named `chirpy`.
3. Create a `.env` file in the project root:

   ```env
   DB_URL="postgres://<user>:@localhost:5432/chirpy?sslmode=disable"
   PLATFORM="dev"
   PUBLIC_KEY=<base64-encoded secret used to sign JWTs>
   POLKA_KEY=<shared secret for the Polka webhook>
   ```

4. Run migrations:

   ```bash
   goose postgres "postgres://<user>:@localhost:5432/chirpy" up
   ```

5. Start the server:

   ```bash
   go run main.go
   ```

   The API listens on `http://localhost:8080`.

## Development commands

```bash
# Run the server
go run main.go

# Build the binary
go build -o chirpy .

# Apply database migrations
goose postgres "postgres://<user>:@localhost:5432/chirpy" up

# Roll back the last migration
goose postgres "postgres://<user>:@localhost:5432/chirpy" down

# Regenerate internal/database from sql/queries and sql/schema
sqlc generate

# Run tests
go test ./...
```

## Data model

**users**
| column | type | notes |
|---|---|---|
| id | UUID | primary key |
| created_at / updated_at | timestamp | |
| email | text | |
| hashed_password | text | argon2id hash |
| is_chirpy_red | boolean | set via the Polka webhook |

**chirps**
| column | type | notes |
|---|---|---|
| id | UUID | primary key |
| created_at / updated_at | timestamp | |
| body | text | max 140 chars, profanity-filtered |
| user_id | UUID | FK → users(id), `ON DELETE CASCADE` |

**refresh_tokens**
| column | type | notes |
|---|---|---|
| token | text | primary key |
| created_at / updated_at | timestamp | |
| user_id | UUID | FK → users(id), `ON DELETE CASCADE` |
| expires_at | timestamp | 60 days from creation |
| revoked_at | timestamp, nullable | set on `/api/revoke` |

## API reference

### Health & admin

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/api/healthz` | – | Health check, returns `200 OK` |
| GET | `/admin/metrics` | – | HTML page showing the request hit counter |
| POST | `/admin/reset` | dev only | Resets the hit counter and deletes all users (requires `PLATFORM=dev`) |

### Auth

| Method | Path | Auth | Description |
|---|---|---|---|
| POST | `/api/login` | – | Log in with email/password, returns access + refresh tokens |
| POST | `/api/refresh` | Bearer refresh token | Issues a new access token |
| POST | `/api/revoke` | Bearer refresh token | Revokes a refresh token |

**POST `/api/login`**

Request:
```json
{ "email": "user@example.com", "password": "secret" }
```
Response `200`:
```json
{
  "id": "uuid",
  "created_at": "...",
  "updated_at": "...",
  "email": "user@example.com",
  "token": "<jwt access token>",
  "refresh_token": "<refresh token>",
  "is_chirpy_red": false
}
```

**POST `/api/refresh`** — header `Authorization: Bearer <refresh_token>`
Response `200`: `{ "token": "<new jwt access token>" }`

**POST `/api/revoke`** — header `Authorization: Bearer <refresh_token>`
Response: `204 No Content`

### Users

| Method | Path | Auth | Description |
|---|---|---|---|
| POST | `/api/users` | – | Create a new user |
| PUT | `/api/users` | Bearer access token | Update the authenticated user's email/password |

Request body (both endpoints):
```json
{ "email": "user@example.com", "password": "secret" }
```
Response:
```json
{
  "id": "uuid",
  "created_at": "...",
  "updated_at": "...",
  "email": "user@example.com",
  "is_chirpy_red": false
}
```

### Chirps

| Method | Path | Auth | Description |
|---|---|---|---|
| POST | `/api/chirps` | Bearer access token | Create a chirp |
| GET | `/api/chirps` | – | List chirps, optionally filtered/sorted |
| GET | `/api/chirps/{chirpID}` | – | Get a single chirp |
| DELETE | `/api/chirps/{chirpID}` | Bearer access token | Delete a chirp (author only) |

**POST `/api/chirps`**

Request:
```json
{ "body": "hello world" }
```
- Max 140 characters, or `400` is returned.
- Profanity filter replaces the words `kerfuffle`, `sharbert`, `fornax` with `****` (case-insensitive).
- `user_id` is taken from the authenticated JWT, not the request body.

Response `201`:
```json
{ "id": "uuid", "created_at": "...", "updated_at": "...", "body": "hello world", "user_id": "uuid" }
```

**GET `/api/chirps`**

Query params (both optional):
- `author_id` — filter chirps by author UUID
- `sort` — `asc` (default) or `desc`, ordered by `created_at`

**DELETE `/api/chirps/{chirpID}`** — `204` on success, `403` if the authenticated user isn't the chirp's author.

### Webhooks

| Method | Path | Auth | Description |
|---|---|---|---|
| POST | `/api/polka/webhooks` | `Authorization: ApiKey <POLKA_KEY>` | Upgrades a user to Chirpy Red |

Request:
```json
{ "event": "user.upgraded", "data": { "user_id": "uuid" } }
```
Only the `user.upgraded` event is handled; other events are ignored. Response: `204 No Content`.

## Authentication

- **Access tokens**: JWT (HS256), signed with `PUBLIC_KEY`, 1 hour expiry. Sent as `Authorization: Bearer <token>` on protected endpoints.
- **Refresh tokens**: opaque random hex strings stored in the `refresh_tokens` table, 60-day expiry, revocable via `/api/revoke`.
- Passwords are hashed with argon2id before storage.