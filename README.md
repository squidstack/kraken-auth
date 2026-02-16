# 🐙 kraken-auth

Authentication service for SquidStack.  
Responsible for issuing JWTs, validating credentials, and providing user and role management APIs.

---

## ✨ Features

- **Login** with username/password → signed JWT (HS256).
- **JWT** includes `sub`, `name`, `roles`, `exp`, `iat`.
- **RBAC**: route protection for `admin` or `storeadmin` roles.
- **Admin APIs** for listing users and roles.
- **Health & Ready** endpoints for Kubernetes.
- **Liquibase migrations** (PostgreSQL) with idempotent seeds.


---

## 📡 Endpoints

### Public Endpoints

| Method | Path      | Auth | Notes |
|--------|-----------|------|-------|
| GET    | `/health` | none | Liveness probe |
| GET    | `/ready`  | none | DB ping (checks PostgreSQL connection) |
| GET    | `/_flags` | none | Current feature flag values |
| POST   | `/login`  | none | `{username, password}` → `{token, user...}` |

### Admin Endpoints (require Bearer token with `admin` or `storeadmin` role)

| Method | Path                          | Description |
|--------|-------------------------------|-------------|
| GET    | `/admin/roles`                | List all available roles |
| GET    | `/admin/countries`            | List all countries (code, name, restricted, can_ship) |
| GET    | `/admin/users`                | List users with profiles (`?limit=&offset=` pagination) |
| GET    | `/admin/users/{id}`           | Get single user with full profile |
| POST   | `/admin/users`                | Create new user with profile and roles |
| PUT    | `/admin/users/{id}`           | Update user profile (full_name, email, phone_number, address, country) |
| PUT    | `/admin/users/{id}/roles`     | Update user roles |
| PUT    | `/admin/users/{id}/password`  | Reset user password |

### Example `/login` response

```json
{
  "id": "70a0b003-3333-4333-83*****0003",
  "user_id": "70a0b003-3333-4333-83*****aaaa0003",
  "username": "betaadmin",
  "full_name": "Beta Admin",
  "email": "betaadmin@example.com",
  "phone_number": "+44 20 7000 0003",
  "address": "77 Minimal Ave, Leeds, UK",
  "country": "GB",
  "roles": ["user", "admin", "betaadmin"],
  "token": "<JWT>"
}
```

---

## 🔑 JWT

- **Algorithm**: HS256  
- **Claims**:
  - `sub`: user ID
  - `name`: username
  - `roles`: array of strings
  - `exp`: expiry (24h by default)
  - `iat`: issued at

---

## 🗄 Database

- **DB**: PostgreSQL
- **Schemas**:
  - `auth` (core identity)
  - `public` (user profiles)
- **Tables**:
  - `auth.users`, `auth.auth_credentials`
  - `auth.roles`, `auth.user_roles`
  - `public.user_profiles` (full_name, email, phone_number, address, country_code, roles[])
  - `public.countries`

### Migrations

Located under `chart/kraken-auth/liquibase/`

Key migrations:
- `0001-create-auth-schema.xml` — Creates `auth` schema
- `0002-create-users-table.xml` — Creates `auth.users` table with UUID primary key
- `0003-create-auth-credentials-table.xml` — Creates `auth.auth_credentials` with password hashing
- `0004-create-roles-table.xml` — Creates `auth.roles`
- `0005-create-user-roles-table.xml` — Creates `auth.user_roles` junction table
- `0006-create-user-profiles-table.xml` — Creates `public.user_profiles` (profile data)
- `0007-create-countries.xml` — Creates `public.countries` table
- `0008-seed-countries.xml` — Loads 201 countries from CSV
- `0009-seed-roles.xml` — Seeds standard roles (user, admin, storeadmin, betauser, betaadmin)
- `0010-seed-users-from-csv.xml` — Seeds initial users from CSV
- `0012-add-beta-roles-and-users.xml` — Adds additional beta test users

All changesets are **idempotent** (safe to reapply).

---

## ⚙️ Configuration

Environment variables:

| Variable         | Required | Description |
|------------------|----------|-------------|
| `JWT_SECRET`     | ✅       | HMAC secret for signing JWTs |
| `AUTH_DB_URL`    | ✅       | Postgres DSN or connection string |
| `AUTH_DB_USER`   | ✅       | Database user |
| `AUTH_DB_PASSWORD` | ✅     | Database password |

---

## 🚀 Run locally

```bash
go mod tidy
go run ./...
```

### Quick test

```bash
# Login
curl -s -X POST http://localhost:8080/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admintest","password":"cbfoobar1234!"}' | jq .

# Use token
TOKEN="<paste token>"
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/admin/users | jq .
```

<!-- Build trigger: 2025-12-11 -->


# Testing Smart Tests integration for Go
# Testing Go Smart Tests fixes (package paths + file attributes)
# Testing Python XML fix for duplicate attributes
# Test YAML fix for Python script
# Test shell-based XML processing

<!-- Test trigger: Thu 22 Jan 2026 10:46:05 GMT -->
# Testing Smart Tests evidence for Go

Last updated: 2026-02-11 21:51 UTC
