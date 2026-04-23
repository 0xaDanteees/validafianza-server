# Deployment en Railway - Validafianza Server

## Resumen

Este documento describe el proceso de deployment de la aplicación Validafianza Server en Railway, incluyendo la configuración de PostgreSQL, variables de entorno, inicialización de base de datos y configuración de custom domain.

---

## Arquitectura de Deployment

### Servicios Desplegados

- **API Service**: Aplicación Go (Fiber) containerizada con Docker
- **PostgreSQL Database**: Base de datos gestionada por Railway
- **Custom Domain**: api.validafianza.com con SSL automático

### Stack Tecnológico

- **Runtime**: Go 1.26.2
- **Framework**: Fiber v2
- **Database**: PostgreSQL 17
- **Container**: Docker multi-stage build
- **Platform**: Railway.app

---

## Prerequisitos

### Herramientas Requeridas

```bash
# Railway CLI
npm i -g @railway/cli

# Docker (para inicialización de DB)
docker --version

# Git
git --version
```

### Cuentas Necesarias

- Cuenta en Railway.app
- Repositorio GitHub con el código
- Cuenta en Resend.com (para emails)
- Dominio configurado (validafianza.com)

---

## Proceso de Deployment

### 1. Autenticación en Railway

```bash
railway login
```

Esto abrirá el navegador para autenticación OAuth.

### 2. Creación del Proyecto

```bash
# Inicializar proyecto
railway init

# Seleccionar workspace
# Ingresar nombre del proyecto: validafianza
```

Railway creará un proyecto nuevo y lo vinculará al directorio actual.

### 3. Provisión de PostgreSQL

Desde el dashboard de Railway:

1. Abrir proyecto: `railway open`
2. Click en "New" → "Database" → "Add PostgreSQL"
3. Railway provisiona automáticamente la base de datos

Variables auto-inyectadas por Railway:

- `DATABASE_URL`
- `PGHOST`
- `PGPORT`
- `PGUSER`
- `PGPASSWORD`
- `PGDATABASE`

### 4. Creación del Servicio de API

Desde el dashboard:

1. Click en "New" → "GitHub Repo"
2. Seleccionar repositorio: `0xaDanteees/validafianza-server`
3. Railway detecta automáticamente `DockerFile` y `railway.json`

### 5. Vinculación de Servicios

```bash
# Vincular al servicio de API
railway service

# Seleccionar: validafianza-server
```

### 6. Configuración de Variables de Entorno

```bash
# Variables de aplicación
railway variables set ENV=production
railway variables set RESEND_API_KEY=re_your_api_key_here
railway variables set RESEND_FROM_EMAIL=noreply@validafianza.com
railway variables set RESEND_FROM_NAME=Validafianza

# Variables de configuración
railway variables set RATE_LIMIT_MAX=60
railway variables set RATE_LIMIT_WINDOW_SECONDS=60
railway variables set CACHE_TTL_HOURS=24
railway variables set SESSION_TTL_DAYS=7
railway variables set PASSCODE_TTL_MINUTES=10
railway variables set MAX_DEVICES_PER_USER=3

# Referencia a PostgreSQL
railway variables set DATABASE_URL='${{Postgres.DATABASE_URL}}'
```

### 7. Inicialización de Base de Datos

Obtener credenciales de PostgreSQL:

```bash
railway connect Postgres
# Copiar la URL de conexión mostrada
```

Ejecutar migraciones usando Docker:

```bash
# Schema principal
docker run --rm -v $(pwd)/db:/db postgres:17-alpine \
  psql postgresql://postgres:PASSWORD@HOST:PORT/railway \
  -f /db/init.sql

# Seeders de proveedores
docker run --rm -v $(pwd)/db:/db postgres:17-alpine \
  psql postgresql://postgres:PASSWORD@HOST:PORT/railway \
  -f /db/seeders/providers.sql

# Migraciones de autenticación
docker run --rm -v $(pwd)/db:/db postgres:17-alpine \
  psql postgresql://postgres:PASSWORD@HOST:PORT/railway \
  -f /db/migrations/001_auth_tables.sql
```

Verificar tablas creadas:

```bash
docker run --rm postgres:17-alpine \
  psql postgresql://postgres:PASSWORD@HOST:PORT/railway \
  -c "\dt"
```

Salida esperada:

``` table
 Schema |        Name         | Type  |  Owner   
--------+---------------------+-------+----------
 public | device_fingerprints | table | postgres
 public | passcodes           | table | postgres
 public | providers           | table | postgres
 public | rate_limits         | table | postgres
 public | scraping_jobs       | table | postgres
 public | sessions            | table | postgres
 public | users               | table | postgres
 public | validation_cache    | table | postgres
```

### 8. Deployment de la Aplicación

```bash
railway up --detach
```

Railway ejecutará:

1. Build del Dockerfile
2. Push de la imagen
3. Deploy del container
4. Health check en `/health`

### 9. Verificación del Deployment

```bash
# Obtener URL del servicio
railway domain

# Verificar health endpoint
curl https://validafianza-server-production.up.railway.app/health

# Verificar autenticación
curl -X POST https://validafianza-server-production.up.railway.app/api/auth/register-or-login \
  -H "Content-Type: application/json" \
  -d '{"email": "test@example.com", "full_name": "Test User"}'
```

### 10. Configuración de Custom Domain

Desde el dashboard de Railway:

1. Ir a Settings → Networking → Custom Domains
2. Click en "Add Custom Domain"
3. Ingresar: `api.validafianza.com`
4. Puerto: `3000`
5. Click en "Add Domain"

Railway mostrará las instrucciones DNS:

``` text
Type: CNAME
Name: api
Value: validafianza-server-production.up.railway.app
```

Configurar DNS en el proveedor:

1. Acceder al panel de DNS (Vercel, Cloudflare, etc.)
2. Agregar registro CNAME:
   - Name: `api`
   - Value: `validafianza-server-production.up.railway.app`
   - TTL: 3600 (o automático)

Verificar propagación DNS:

```bash
dig api.validafianza.com

# Salida esperada:
# api.validafianza.com. 60 IN CNAME validafianza-server-production.up.railway.app.
```

Railway provisiona automáticamente certificado SSL (Let's Encrypt). Esperar 2-5 minutos.

Verificar HTTPS:

```bash
curl https://api.validafianza.com/health
```

---

## Configuración de PostgreSQL

### Parámetros Optimizados

La configuración de PostgreSQL está optimizada para el free tier de Railway (512MB RAM):

```yaml
max_connections: 20
shared_buffers: 128MB
effective_cache_size: 384MB
work_mem: 4MB
maintenance_work_mem: 64MB
wal_buffers: 4MB
random_page_cost: 1.1
effective_io_concurrency: 200
```

Estas configuraciones están definidas en `compose.yaml` para desarrollo local. Railway usa configuraciones similares por defecto.

### Extensiones Habilitadas

```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "hstore";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
```

---

## Variables de Entorno

### Variables de Producción

| Variable | Valor | Descripción |
| -------- | ----- | ----------- |
| `ENV` | `production` | Modo de ejecución (logs JSON, nivel INFO) |
| `DATABASE_URL` | `${{Postgres.DATABASE_URL}}` | Referencia automática a PostgreSQL |
| `PORT` | `3000` | Puerto de escucha de la aplicación |
| `RESEND_API_KEY` | `re_live_xxx` | API key de Resend para emails |
| `RESEND_FROM_EMAIL` | `noreply@validafianza.com` | Email remitente |
| `RESEND_FROM_NAME` | `Validafianza` | Nombre del remitente |
| `RATE_LIMIT_MAX` | `60` | Máximo de requests por ventana |
| `RATE_LIMIT_WINDOW_SECONDS` | `60` | Ventana de rate limiting en segundos |
| `CACHE_TTL_HOURS` | `24` | TTL del cache de validaciones |
| `SESSION_TTL_DAYS` | `7` | Duración de sesiones de usuario |
| `PASSCODE_TTL_MINUTES` | `10` | Duración de códigos OTP |
| `MAX_DEVICES_PER_USER` | `3` | Límite de dispositivos por usuario |

### Variables Auto-Inyectadas por Railway

Railway inyecta automáticamente estas variables:

- `RAILWAY_ENVIRONMENT`: `production`
- `RAILWAY_PROJECT_ID`: ID del proyecto
- `RAILWAY_SERVICE_ID`: ID del servicio
- `RAILWAY_PRIVATE_DOMAIN`: Dominio interno
- `DATABASE_URL`: URL completa de PostgreSQL

---

## Dockerfile Multi-Stage

El deployment usa un Dockerfile multi-stage para optimizar el tamaño de la imagen:

```dockerfile
# Stage 1: Builder
FROM golang:1.26.2-alpine3.23 AS builder
WORKDIR /app
RUN apk add --no-cache git ca-certificates tzdata
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags="-w -s" -o main ./cmd/api

# Stage 2: Runtime
FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata wget
WORKDIR /root/
COPY --from=builder /app/main .
EXPOSE 3000
HEALTHCHECK --interval=30s --timeout=30s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:3000/health || exit 1
CMD ["./main"]
```

Tamaño final de la imagen: ~15MB

---

## Health Checks

Railway monitorea automáticamente el endpoint `/health`:

```json
{
  "status": "ok",
  "service": "validafianza-api"
}
```

Configuración en `railway.json`:

```json
{
  "deploy": {
    "healthcheckPath": "/health",
    "healthcheckTimeout": 100,
    "restartPolicyType": "ON_FAILURE",
    "restartPolicyMaxRetries": 10
  }
}
```

---

## Logs y Monitoreo

### Visualizar Logs

```bash
# Logs en tiempo real
railway logs

# Últimas 100 líneas
railway logs --tail 100
```

### Formato de Logs

Producción (JSON estructurado):

```json
{
  "time": "2026-04-23T02:24:45.342Z",
  "level": "INFO",
  "msg": "server starting",
  "addr": ":3000",
  "env": "production"
}
```

Desarrollo (texto legible):

``` log
[INFO] 2026-04-23 02:24:45 server starting addr=:3000 env=development
```

### Métricas en Dashboard

Railway proporciona métricas automáticas:

- CPU usage
- Memory usage
- Network traffic
- Request count
- Response times

---

## Troubleshooting

### Error: "failed to connect to db"

Causa: `DATABASE_URL` no está configurada o es incorrecta.

Solución:

```bash
# Verificar variables
railway variables --kv | grep DATABASE_URL

# Reconfigurar referencia
railway variables set DATABASE_URL='${{Postgres.DATABASE_URL}}'
```

### Error: "SSL certificate mismatch"

Causa: Certificado SSL aún no provisionado para custom domain.

Solución: Esperar 2-5 minutos después de agregar el custom domain.

### Error: "Health check failed"

Causa: Aplicación no responde en el puerto correcto.

Solución:

```bash
# Verificar que PORT=3000
railway variables --kv | grep PORT

# Ver logs de la aplicación
railway logs --tail 50
```

### Error: "No service linked"

Causa: CLI no está vinculado al servicio correcto.

Solución:

```bash
# Re-vincular al servicio
railway service

# Seleccionar: validafianza-server
```

---

## Rollback

### Rollback a Deployment Anterior

Desde el dashboard:

1. Ir a Deployments
2. Seleccionar deployment anterior
3. Click en "Redeploy"

Desde CLI:

```bash
railway rollback
```

### Rollback de Variables

Las variables no tienen historial. Mantener backup manual:

```bash
# Exportar variables actuales
railway variables --kv > variables-backup-$(date +%Y%m%d).txt
```

---

## Backup de Base de Datos

### Backup Automático

Railway PostgreSQL incluye:

- Backups diarios automáticos
- Retención de 7 días (free tier)
- Point-in-time recovery (planes pagos)

### Backup Manual

```bash
# Exportar base de datos
docker run --rm postgres:17-alpine \
  pg_dump postgresql://postgres:PASSWORD@HOST:PORT/railway \
  > backup-$(date +%Y%m%d).sql

# Restaurar desde backup
docker run --rm -i postgres:17-alpine \
  psql postgresql://postgres:PASSWORD@HOST:PORT/railway \
  < backup-20260423.sql
```

---

## Costos y Límites

### Free Tier

- Crédito mensual: $5 USD
- Compute: ~500 horas/mes
- PostgreSQL: 1GB storage
- Bandwidth: 100GB/mes
- Backups: 7 días retención

### Uso Estimado (20k usuarios)

- Compute: ~300 horas/mes
- Storage: ~200MB
- Bandwidth: ~50GB/mes
- Costo: $0 (dentro del free tier)

### Upgrade a Paid Plan

Cuando excedas el free tier:

- Compute: ~$5/mes por instancia
- PostgreSQL: Incluido en compute
- Total estimado: $5-7/mes

---

## Seguridad

### Checklist de Seguridad

- [x] HTTPS habilitado (SSL automático)
- [x] Credenciales de DB auto-rotadas por Railway
- [x] Secrets en variables de entorno (no en código)
- [x] Rate limiting configurado
- [x] Tokens hasheados con SHA256
- [x] Sesiones con TTL y sliding window
- [x] Device fingerprinting habilitado
- [ ] 2FA habilitado en cuenta Railway
- [ ] Monitoring alerts configurados

### Rotación de Secrets

```bash
# Rotar API key de Resend
railway variables set RESEND_API_KEY=re_live_NEW_KEY

# Railway redeploya automáticamente
```

---

## CI/CD

### Auto-Deploy desde GitHub

Railway está configurado para auto-deploy en cada push a `main`:

1. Push a GitHub
2. Railway detecta cambios
3. Build automático del Dockerfile
4. Deploy si build exitoso
5. Health check
6. Rollback automático si falla

### Deshabilitar Auto-Deploy

Desde dashboard:

1. Settings → Deployments
2. Desmarcar "Auto-deploy on push"

### Deploy Manual

```bash
railway up --detach
```

---

## Endpoints Productivos

### URLs

- **API Base**: <https://api.validafianza.com>
- **Health**: <https://api.validafianza.com/health>
- **Swagger**: <https://api.validafianza.com/swagger/index.html>
- **Auth**: <https://api.validafianza.com/api/auth/>*
- **Validación**: <https://api.validafianza.com/api/v1/polizas/>*

### Ejemplos de Uso

```bash
# Health check
curl https://api.validafianza.com/health

# Registro/Login
curl -X POST https://api.validafianza.com/api/auth/register-or-login \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "full_name": "Usuario"}'

# Verificar código
curl -X POST https://api.validafianza.com/api/auth/verify \
  -H "Content-Type: application/json" \
  -d '{"code": "ABC123", "fingerprint_data": "device-id", "device_name": "iPhone"}'

# Validar póliza (requiere autenticación)
curl -X POST https://api.validafianza.com/api/v1/polizas/validate \
  -H "Authorization: Bearer TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"provider_key": "MAPFRE", "lookup_key": "0032400003909"}'
```

---

## Mantenimiento

### Actualización de Dependencias

```bash
# Actualizar módulos Go
go get -u ./...
go mod tidy

# Commit y push
git add go.mod go.sum
git commit -m "chore: update dependencies"
git push

# Railway redeploya automáticamente
```

### Limpieza de Datos Expirados

La función `cleanup_expired_data()` se ejecuta automáticamente:

```sql
-- Elimina sesiones y passcodes expirados hace >24h
SELECT cleanup_expired_data();
```

Para ejecutar manualmente:

```bash
docker run --rm postgres:17-alpine \
  psql postgresql://postgres:PASSWORD@HOST:PORT/railway \
  -c "SELECT cleanup_expired_data();"
```

---

## Contacto y Soporte

- **Railway Support**: <https://railway.app/help>
- **Documentación Railway**: <https://docs.railway.app>
- **Status Page**: <https://status.railway.app>

---

## Changelog

### 2026-04-23 - Deployment Inicial

- Creación de proyecto en Railway
- Provisión de PostgreSQL 17
- Deployment de API con Docker
- Configuración de custom domain api.validafianza.com
- SSL automático habilitado
- Variables de entorno configuradas
- Base de datos inicializada con schema y seeders
