# Gothenticate

REST API in Go with JWT authentication, order management, and applied Design Patterns (Strategy, State, Observer, Cache Aside with Redis).

Built as a practical study project to apply backend engineering concepts in a real service context.

---

## Tech Stack

- **Go** with [Echo](https://echo.labstack.com/) framework
- **SQLite** via `database/sql` (no ORM)
- **Redis** for caching (`go-redis/v9`)
- **JWT** for authentication (`golang-jwt`)
- **bcrypt** for password hashing

---

## Architecture

The project follows a **package-by-feature** structure:

```
cmd/api/              — entry point, server startup, graceful shutdown
internal/
  auth/               — signup, login, account state machine
  modules/order/      — order domain, handlers, repository
    domain/           — business logic and design patterns
  middleware/         — JWT auth, rate limiting, security headers
  infra/              — database, logger, validator, errors
migrations/           — SQL schema
```

---

## Design Patterns

Five patterns applied with real business motivation — not for the sake of it.

### Strategy — Shipping Calculation
Each carrier (`standard_mail`, `express_mail`, `dhl_international`, `local_drone`) implements a `ShippingCalculator` interface with its own pricing logic. The service selects the right strategy at runtime via a Factory function, without any `if/else` about carrier-specific rules leaking into the handler.

### Factory Method — Strategy and State Instantiation
`GetShippingCalculator` and `stateFromLabel` centralize object creation from string identifiers (request body or database value), keeping construction logic in one place.

### State — Order and Account Lifecycle
Both orders and user accounts use explicit state machines.

**Order flow:** `pending → paid → preparing → sent → received`, with `canceled` available from `pending`. Each state struct defines which transitions are valid — the handler calls `Pagar()`, `Enviar()` or `Cancelar()` and the state decides if it's allowed, returning an error if not.

**Account flow:** `active → locked` (after 5 failed login attempts) `→ blocked` (administrative). Reconstructed from the database label on every request.

### Observer — Order Status Notifications
When an order transitions to a new state, `NotificadorEstoque`, `NotificadorFiscal` and `NotificadorEmail` are notified automatically via `OnStatusChanged`. The handler has no knowledge of downstream systems — adding a new observer requires no changes to existing code.

### Cache Aside — Order Reads with Redis
`GET /orders/:id` checks Redis first. On a cache miss, it fetches from SQLite, stores the result with a TTL, and returns. `PATCH /orders/:id` and `DELETE /orders/:id` invalidate the cache key after a successful write. Redis is injected as a dependency, so the service degrades gracefully if Redis is unavailable.

---

## Endpoints

### Auth
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/signup` | — | Register a new user |
| POST | `/login` | — | Authenticate and receive JWT |
| GET | `/me` | JWT | Return authenticated user info |

### Orders
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/orders` | JWT | List orders with pagination and filters |
| POST | `/orders` | JWT | Create a new order |
| GET | `/orders/:id` | JWT | Get order by ID (cached) |
| PATCH | `/orders/:id` | JWT | Advance order status |
| DELETE | `/orders/:id` | JWT | Delete order (blocked if in progress) |

---

## Getting Started

**Requirements:** Go 1.21+, Redis instance

```bash
git clone https://github.com/GuiFernandess7/gothenticate.git
cd gothenticate
cp .env.example .env   # fill in JWT_SECRET and REDIS_CONN_STRING
```

Run migrations:
```bash
sqlite3 storage/database.sqlite < migrations/table_create.sql
```

Start the server:
```bash
go run cmd/api/main.go
```

---

## Environment Variables

| Variable | Description |
|----------|-------------|
| `SQLITE_DSN` | Path to SQLite file (e.g. `storage/database.sqlite`) |
| `JWT_SECRET` | Secret key for signing JWT tokens |
| `REDIS_CONN_STRING` | Redis connection URL (e.g. `redis://localhost:6379`) |

---

## Running Tests

```bash
go test ./tests/...
```