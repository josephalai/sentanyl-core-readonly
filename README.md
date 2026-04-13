# core-service

The foundational platform service. Handles tenant authentication, email domain management, the SentanylScript DSL compiler, and the story execution engine that orchestrates email campaigns.

**Port:** `8081`

## Responsibilities

- Tenant registration, login, and account management
- Email sending domain configuration and DNS/DKIM verification
- SentanylScript DSL compilation and validation
- Story (campaign) builder and scheduler
- Email click and event tracking
- Inter-service coordination via a service bridge

## Directory Structure

```
core-service/
├── cmd/
│   └── main.go          # Entry point
├── routes/
│   ├── tenant_auth.go   # Tenant registration, login, profile
│   ├── domains.go       # Sending domain lifecycle and DNS verification
│   ├── story.go         # Story CRUD — storylines, scenes, messages, badges, triggers
│   ├── story_engine.go  # Story execution, session tracking, background scheduler
│   ├── script.go        # SentanylScript compiler and validator endpoints
│   ├── tracking.go      # Click/event tracking
│   ├── bridge.go        # Outbound HTTP calls to other services
│   ├── dkim.go          # DKIM helpers
│   ├── dns.go           # DNS helpers
│   ├── tenant_domains.go
│   └── tenant_reset.go
├── scripting/
│   ├── compiler.go      # DSL → executable form
│   ├── lexer_test.go
│   ├── parser.go
│   ├── parser_video.go  # Video-specific DSL parsing
│   ├── validator.go
│   └── e2e_test.go
└── hydrator/            # Async task worker stub
```

## API Endpoints

### Public

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/tenant/register` | Create a new tenant account |
| `POST` | `/api/tenant/login` | Tenant login — returns JWT |
| `POST` | `/api/customer/login` | Customer/subscriber login |

### Tenant (JWT required)

| Method | Path | Description |
|--------|------|-------------|
| `GET/PUT` | `/api/tenant/profile` | Tenant account details |
| `GET/DELETE` | `/api/tenant/reset` | Account reset |
| `POST/GET/DELETE` | `/api/tenant/domains` | Custom domain management |
| `POST` | `/api/domain` | Add a sending domain (calls sidecar) |
| `GET` | `/api/domains` | List all sending domains |
| `POST` | `/api/domain/:id/verify-dns` | Verify DNS records |
| `POST` | `/api/domain/:id/test-send` | Send a test email |
| `GET` | `/api/domain/:id/stats` | Sending statistics |
| `POST` | `/api/domain/:id/pause` | Pause sending on a domain |
| `POST` | `/api/domain/:id/resume` | Resume sending on a domain |

### Internal

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/internal/story/start` | Trigger story execution (called by other services) |

Script compiler and story builder/engine routes are registered via their own `Register*Routes` helpers.

## Key Concepts

**Stories** are email campaign definitions. They contain Storylines → Acts (Enactments) → Scenes, each with messages and badge-based branching logic. The story scheduler runs as a background goroutine and advances sessions through their state machine.

**SentanylScript** is a domain-specific language for authoring campaign logic. The compiler in `scripting/compiler.go` converts DSL source to an executable form that the story engine runs.

**Sending Domains** are DKIM-signed outbound email domains. Core-service owns their lifecycle and delegates SMTP/queue control to the PowerMTA sidecar in `email_server`.

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `CORE_SERVICE_PORT` | `8081` | HTTP listen port |
| `MONGO_HOST` | — | MongoDB host |
| `MONGO_PORT` | — | MongoDB port |
| `MONGO_DB` | — | Database name |
| `LMS_SERVICE_URL` | — | URL for lms-service |
| `MARKETING_SERVICE_URL` | — | URL for marketing-service |
| `PUBLIC_BASE_URL` | — | Base URL embedded in email links |

## Dependencies

- [`gin-gonic/gin`](https://github.com/gin-gonic/gin) — HTTP framework
- `gopkg.in/mgo.v2` — MongoDB driver
- `../pkg` — Shared auth, config, db, models, HTTP utilities
