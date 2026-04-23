# Validafianza Server — Architecture

## Stack

- **API**: Go + Fiber v2 — sub-ms routing, connection pooling
- **Scraping**: Go native HTTP — ASPX/JSP form handling, cookie management (no Playwright)
- **DB**: PostgreSQL 17 — JSONB config, UNLOGGED cache tables, SKIP LOCKED job queue
- **Infra**: Docker Compose (local) → GCP Cloud Run + Cloud SQL (prod)

### Why No Playwright?

All 8 afianzadoras use server-side rendering (ASPX/JSP). No JS-rendered content, no CAPTCHA, no WebSockets.

- Latency: 300–800ms vs 2–8s
- Memory: 10–30MB vs 500MB–1GB per browser
- Concurrency: 1000+ req/instance vs 5–10

---

## Project Structure

```text
validafianza-server/
├── cmd/api/main.go              # Entry point, scraper + auth setup, Fiber routes
├── internal/
│   ├── config/config.go         # ENV loading (DATABASE_URL, PORT, RESEND_API_KEY, etc)
│   ├── handlers/
│   │   ├── health_handler.go    # GET /health
│   │   ├── poliza_handler.go    # POST /api/v1/polizas/validate, GET /providers
│   │   └── auth_handler.go      # POST /auth/register-or-login, /verify, /logout, GET /profile
│   ├── services/
│   │   ├── validation_service.go      # Cache check → scrape → cache write
│   │   ├── auth_service.go            # RegisterOrLogin, VerifyPasscode, ValidateSession, Logout
│   │   ├── notification_service.go    # EmailProvider + SMSProvider interfaces
│   │   └── providers/
│   │       ├── resend_provider.go     # Resend API integration
│   │       └── sms_provider.go        # SMTP-to-SMS gateway (Telcel)
│   ├── scrapers/
│   │   ├── base.go              # Scraper interface + BaseScraper
│   │   ├── registry.go          # Register/Get/List (RWMutex map)
│   │   └── polizas/
│   │       ├── shared.go        # searchText, renderNode (package-level utils)
│   │       ├── base_aspx.go     # Generic ASPX scraper (4 providers reuse this)
│   │       ├── mapfre.go        # ASPX + HTML table extraction
│   │       ├── berkley.go       # aspxPolizaScraper — validates: "VIGENTE"
│   │       ├── sofimex.go       # aspxPolizaScraper — validates: !"no se pudo validar"
│   │       ├── dorama.go        # aspxPolizaScraper — validates: "Código de validación:"
│   │       ├── tokio.go         # aspxPolizaScraper — validates: "ANOTE ESTE NÚMERO"
│   │       ├── atlas.go         # ASPX, form fields distintos (txtLineaValidacion)
│   │       ├── aserta.go        # JSP GET — validates: !"no existe"
│   │       └── insurgentes.go   # JSP GET — validates: !"no existe"
│   ├── models/
│   │   ├── validation.go        # ValidationProvider, ValidationRequest/Response
│   │   └── auth.go              # User, Passcode, DeviceFingerprint, Session, SessionValidation
│   ├── repository/
│   │   ├── postgres.go          # GetProviderByKey, Pool()
│   │   ├── cache.go             # Get (UPDATE...RETURNING), Set (UPSERT)
│   │   ├── rate_limit_repo.go   # Allow() → SELECT check_rate_limit($1,$2,$3)
│   │   └── auth_repository.go   # CreateUser, CreatePasscode, ValidateSession, DeleteSession
│   ├── middleware/
│   │   ├── rate_limit.go        # RateLimit() — sliding window via check_rate_limit()
│   │   ├── rate_limit_middleware.go  # LimitByIP, LimitByEmail (auth endpoints)
│   │   ├── auth_middleware.go   # Authenticate — valida token, inyecta user_id/session_id
│   │   └── request_logger.go    # RequestLogger() — slog structured access log
│   ├── logger/logger.go         # Init(env) — JSON (prod) / text (dev) slog handler
│   └── utils/
│       ├── aspx.go              # ASPXClient: GET→ViewState, POST form
│       └── jsp.go               # FetchJSPPage: GET ?txtLineaCaptura=
├── db/
│   ├── init.sql                 # Schema completo con auth tables
│   ├── migrations/
│   │   └── 001_auth_tables.sql  # Migración idempotente (CREATE IF NOT EXISTS)
│   └── seeders/providers.sql    # 8 afianzadoras seed
├── docs/
│   ├── ARCHITECTURE.md          # Este archivo
│   └── swagger files            # Generados por swag init
├── DockerFile                   # Multi-stage: golang:1.26-alpine → alpine
├── compose.yaml                 # Postgres + API con env vars de Resend
├── go.mod
└── .env                         # POSTGRES_*, PORT, RESEND_*, SMTP_*
```

---

## Scraper Patterns

### Pattern 1 — Generic ASPX (`aspxPolizaScraper`)

Berkley, Sofimex, Dorama, Tokio comparten el mismo flow:

```text
GET baseURL → ExtractHiddenInputs (ViewState, EventValidation)
→ Set ctl00$maincontent$ap1_content$lvCompleto = lookupKey
→ Set ctl00$maincontent$bValida = buttonValue
→ POST → searchText(doc, pattern)
```

Factory de 10 líneas por proveedor. Sin duplicación.

### Pattern 2 — Custom ASPX

**MAPFRE**: mismos form fields que el patrón genérico pero extrae `<table class="mainTable">` del HTML resultante.

**Atlas**: form fields distintos — `txtLineaValidacion`, radio `1=RdTextoPDF`, submit `BtnValidarPDF`. Mismo `ASPXClient` subyacente.

### Pattern 3 — JSP GET

Aserta e Insurgentes: `GET baseURL?txtLineaCaptura={lookupKey}` sin form submission. Valid = "no existe" no encontrado en DOM.

### Extensibility

Agregar un proveedor ASPX estándar:

```go
// 1. Factory function en polizas/nuevo.go
func NewNuevoScraper() *aspxPolizaScraper {
    return &aspxPolizaScraper{
        BaseScraper: scrapers.NewBaseScraper("NUEVO_PROVEEDOR", "native"),
        baseURL:     "https://nuevo.com/validar/Default.aspx",
        buttonValue: "Validar",
        isValid: func(doc *html.Node) bool {
            return searchText(doc, regexp.MustCompile(`(?i)VIGENTE`))
        },
    }
}

// 2. Registrar en cmd/api/main.go
scrapers.Register(polizas.NewNuevoScraper())

// 3. Seed en DB
INSERT INTO providers (provider_type, provider_key, scraper_type, config)
VALUES ('poliza', 'NUEVO_PROVEEDOR', 'native', '{"url": "..."}'::jsonb);
```

---

## Database Schema

```sql
-- Core
providers             -- Registro de afianzadoras (UUID PK, JSONB config)
validation_cache      -- UNLOGGED, TTL via expires_at, SHA256 lookup_hash PK
scraping_jobs         -- SKIP LOCKED queue (pending → processing → completed/failed)
rate_limits           -- UNLOGGED, requests JSONB[] timestamps, check_rate_limit() function

-- Auth
users                 -- email UNIQUE, max_devices, is_verified, metadata hstore
passcodes             -- code, user_id FK, expires_at, is_used, attempts
device_fingerprints   -- fingerprint_hash UNIQUE (SHA256), user_id FK, is_trusted
sessions              -- token_hash (SHA256), user_id FK, device_fingerprint_id FK, expires_at
```

### Rate limiting — sliding window

`check_rate_limit(key, limit, window_seconds)` hace UPSERT atómico: poda timestamps vencidos del array JSONB, agrega el actual, retorna `count <= limit`. Aplicado en `POST /api/v1/polizas/validate` vía `middleware.RateLimit`. Configurable con `RATE_LIMIT_MAX` (default 60) y `RATE_LIMIT_WINDOW_SECONDS` (default 60).

### Structured logging

`log/slog` stdlib. `ENV=production` → JSON a stdout (GCP Log Explorer compatible). `ENV=development` → texto legible con nivel DEBUG. El middleware `RequestLogger` emite `method`, `path`, `status`, `latency_ms`, `ip`, `bytes_out` por cada request.

Extensiones: `hstore`, `pg_trgm`, `btree_gin`

Cache query pattern (atomic read + increment):

```sql
UPDATE validation_cache
SET access_count = access_count + 1
WHERE lookup_hash = $1 AND expires_at > NOW()
RETURNING result, scraped_at
```

---

## API Endpoints

### Public

```text
GET  /health                              → service liveness
GET  /swagger/*                           → Swagger UI (doc.json served relative)
POST /api/auth/register-or-login          → { email, full_name?, phone? } → envía OTP
POST /api/auth/verify                     → { code, fingerprint_data, device_name? } → retorna token
```

### Protected (requieren `Authorization: Bearer {token}`)

```text
GET  /api/auth/profile                    → { user_id, email }
POST /api/auth/logout                     → invalida sesión actual
GET  /api/v1/polizas/providers            → lista de afianzadoras disponibles
POST /api/v1/polizas/validate             → { provider_key, lookup_key, force_refresh? }
```

### OpenAPI / Swagger

Generado con `swaggo/swag`. Para regenerar tras cambios en handlers o models:

```bash
swag init -g cmd/api/main.go --output docs
```

Anotaciones relevantes:

- `@Param` / `@Success` / `@Failure` en handlers
- `example:"..."` tags en structs de models — swag los pica como default values en la UI
- `ErrorResponse` struct unificada para todos los errores (400, 500)

La UI usa `URL: "doc.json"` (relativo) en lugar de absoluto — por eso el explore bar funciona en cualquier host/puerto.

---

## Autenticación

Sistema passwordless con OTP (One-Time Passcode) vía email. Sesiones con sliding window y device fingerprinting para prevenir compartición de cuentas.

### Flow

```text
1. POST /api/auth/register-or-login { email }
   → Crea usuario si no existe
   → Genera código de 6 caracteres (10 min TTL)
   → Envía email vía Resend
   
2. POST /api/auth/verify { code, fingerprint_data, device_name }
   → Valida código (3 intentos máx)
   → Crea/actualiza device fingerprint (SHA256 hash)
   → Verifica límite de dispositivos (3 por usuario)
   → Genera session token (base32, 7 días TTL)
   → Retorna token + user data
   
3. Endpoints protegidos usan middleware authMiddleware.Authenticate
   → Valida token vía validate_session() SQL function
   → Actualiza last_activity_at (sliding window)
   → Inyecta user_id, session_id en fiber.Locals
   
4. POST /api/auth/logout
   → DELETE FROM sessions WHERE id = session_id
   → Token inmediatamente inválido
```

### Tablas

```sql
users                 -- email UNIQUE, max_devices (default 3), is_verified
passcodes             -- code, expires_at, is_used, attempts, max_attempts
device_fingerprints   -- fingerprint_hash UNIQUE (SHA256), is_trusted, last_seen_at
sessions              -- token_hash (SHA256), expires_at, last_activity_at
```

### Rate Limiting

- **Email**: 3 códigos / 5 min por email (`auth:email:{email}`)
- **IP**: 10 req/min en `/register-or-login`, 5 req/min en `/verify`
- **Código**: 3 intentos por passcode (columna `attempts`)

### Notificaciones

**Email**: Resend API (`RESEND_API_KEY`, `RESEND_FROM_EMAIL`)

**SMS** (México): SMTP-to-SMS gateway gratuito para Telcel (`{phone}@sms.telcel.com`). Configurado pero no activo por default.

### Security

- Tokens hasheados con SHA256 antes de almacenar
- IPs almacenadas como tipo INET nativo de PostgreSQL
- Device fingerprints basados en User-Agent + navegador + OS
- Cleanup automático: `cleanup_expired_data()` elimina sesiones/passcodes expirados hace >24h
- Todos los endpoints de validación requieren autenticación

---

## Deployment

**Plataforma**: Railway.app

- Docker nativo (usa `DockerFile` + `compose.yaml` directamente)
- PostgreSQL incluido (1GB storage, backups automáticos)
- Auto-deploy desde GitHub (push → deploy)
- SSL + custom domain gratis
- $5 USD crédito mensual (suficiente para ~500 hrs compute)
- Logs estructurados compatibles con `slog`

**Configuración PostgreSQL optimizada para free tier:**

- `max_connections=20` (suficiente para 100k req/día)
- `shared_buffers=128MB` (cabe en 512MB RAM)
- `work_mem=4MB` (queries simples con PK lookups)
- SSD optimizado (`random_page_cost=1.1`, `effective_io_concurrency=200`)

---

## Roadmap

### Scrapers Adicionales

- `internal/scrapers/proveedores/` — SFP (Secretaría de la Función Pública)
- `internal/scrapers/proveedores/` — SCG (Sistema de Contrataciones Gubernamentales)
- `internal/scrapers/licitaciones/` — Sistema de licitaciones gubernamentales

### Observability

- Prometheus metrics (`validation_duration_seconds`, `scraper_errors_total`, `auth_attempts_total`)
- Grafana dashboards (latencia P50/P95/P99, cache hit rate, error rate)
- Alerting (Slack/Discord webhooks para errores críticos) - TBD

### Background Jobs

- `cmd/worker/` — Procesador de jobs con SKIP LOCKED queue
- Cleanup automático programado (sessions/passcodes expirados)
- Retry logic para scrapers fallidos

### Comunicaciones

- WhatsApp Business API para OTP (más confiable que email en México)
- Templates HTML para emails (reemplazar plain text actual)
- SMS
