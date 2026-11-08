# Gastos API

## 1. Descripción y problema que resuelve

API REST backend para la gestión de gastos personales, escrita en Go. Permite registrar ingresos y gastos, organizarlos por categorías, definir presupuestos mensuales y consultar reportes financieros. Resuelve el problema de llevar un control claro del dinero sin depender de hojas de cálculo: expone datos consistentes, valida las reglas de negocio en el servidor y ofrece autenticación segura con access y refresh tokens.

Es un proyecto de portafolio: **solo API, sin frontend**, con capas separadas, inyección de dependencias, caché opcional con Redis, documentación Swagger y tests automatizados.

---

## 2. Tecnologías utilizadas

| Tecnología | Uso en el proyecto |
| --- | --- |
| **Go 1.22+** | Lenguaje del backend; `go.mod` fija Go 1.22. |
| **Gin** | Framework HTTP: rutas, grupos, middleware y binding de requests. |
| **GORM** | ORM para el acceso a datos y las migraciones automáticas. |
| **SQLite** + **glebarez/sqlite** | Base de datos embebida con driver de Go puro (sin CGO), ideal para Docker. |
| **JWT (golang-jwt/jwt/v5)** | Firma y validación de access tokens con expiración. |
| **bcrypt** | Hash seguro de contraseñas; nunca se guardan en claro. |
| **Redis** + **go-redis/v9** | Caché de reportes y rate limiting distribuido, con degradación elegante. |
| **Swagger (swaggo)** | Documentación interactiva de la API y esquema de seguridad Bearer. |
| **Docker + Docker Compose** | Empaquetado multi-stage y ejecución conjunta de API + Redis. |
| **testify** | Aserciones y helpers en los tests. |
| **godotenv** | Carga de variables de entorno desde `.env`. |
| **encoding/csv** | Exportación de transacciones a CSV. |
| **log/slog** | Logging estructurado, incluyendo warnings cuando Redis falla. |

---

## 3. Características

- Registro y login con contraseñas hasheadas con bcrypt.
- Access token JWT de corta duración y refresh token opaco de larga duración.
- Rotación de refresh tokens y detección de reutilización.
- Cierre de sesión con revocación del refresh token.
- CRUD de categorías de tipo `income` o `expense`.
- CRUD de transacciones con validación de tipo, categoría y pertenencia al usuario.
- Filtros de transacciones por fecha, categoría, tipo y rango de monto.
- Paginación con metadatos (`page`, `limit`, `total`).
- Exportación de transacciones filtradas a CSV.
- Presupuestos mensuales por categoría con gasto, restante, porcentaje usado y estado.
- Reporte mensual con totales y desglose de gastos por categoría usando SQL.
- Reporte de tendencia de los últimos N meses.
- Caché de reportes en Redis con TTL de 10 minutos e invalidación automática.
- Rate limiting en login y registro: 5 intentos por minuto por IP.
- API completamente funcional sin Redis (`REDIS_ENABLED=false` o Redis caído).
- Formato uniforme de respuestas y errores.
- Health check, Swagger, CORS y graceful shutdown.
- Tests de handlers, reportes y rotación de refresh tokens.

---

## 4. Requisitos previos

- **Go 1.22 o superior**.
- **Docker y Docker Compose** (opcional, para la ejecución containerizada).
- **Redis 7** (opcional; solo para caché y rate limiting).
- `curl` para probar la API y, opcionalmente, `jq` para extraer tokens de las respuestas.

El driver de SQLite es de Go puro: **no se necesita CGO**, GCC ni un compilador de C.

---

## 5. Configuración

Crea el archivo `.env` a partir del ejemplo:

```bash
cp .env.example .env
```

En PowerShell:

```powershell
Copy-Item .env.example .env
```

| Variable | Valor por defecto | Descripción |
| --- | --- | --- |
| `PORT` | `8080` | Puerto HTTP donde escucha la API. |
| `GIN_MODE` | `debug` | Modo de Gin: `debug`, `release` o `test`. Usa `release` en producción. |
| `DB_PATH` | `gastos.db` | Ruta del archivo SQLite. En Docker se usa `/data/gastos.db`. |
| `JWT_SECRET` | *(obligatorio)* | Secreto con el que se firman los access tokens. Debe tener al menos 16 caracteres. Genera uno con `openssl rand -hex 32`. |
| `ACCESS_TOKEN_TTL` | `15m` | Duración del access token en formato de Go (`15m`, `1h`). |
| `REFRESH_TOKEN_TTL` | `168h` | Duración del refresh token (168 horas = 7 días). Debe ser mayor que el access token. |
| `REDIS_ENABLED` | `false` | Activa el uso de Redis. Si es `false`, la API funciona sin caché ni rate limit distribuido. |
| `REDIS_ADDR` | `localhost:6379` | Dirección de Redis (`host:puerto`). |
| `REDIS_PASSWORD` | *(vacío)* | Contraseña de Redis, si está configurada. |

El archivo `.env` está en `.gitignore` y nunca debe versionarse.

> **Importante:** si Redis no está disponible o `REDIS_ENABLED=false`, la API **no falla**. Simplemente consulta SQLite directamente y registra warnings en el log cuando corresponde.

---

## 6. Cómo ejecutarlo

### 6.1 Local con Go (sin Redis)

Es la forma más rápida de arrancar. La API usará SQLite y no intentará conectarse a Redis.

```bash
cp .env.example .env
# Asegúrate de que REDIS_ENABLED=false en .env
go mod download
go run ./cmd/api
```

La API queda disponible en `http://localhost:8080`.

- Health check: <http://localhost:8080/health>
- Swagger UI: <http://localhost:8080/swagger/index.html>

### 6.2 Local con Go y Redis

1. Inicia Redis (por ejemplo, con Docker):

```bash
docker run --rm -d --name gastos-redis -p 6379:6379 redis:7-alpine
```

2. Edita `.env` y activa Redis:

```dotenv
REDIS_ENABLED=true
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
```

3. Arranca la API:

```bash
go run ./cmd/api
```

Con Redis activo se habilitan la caché de reportes y el rate limiting de login/registro. Puedes comprobarlo en `GET /health`: el campo `checks.redis` aparecerá como `up`.

### 6.3 Con Docker Compose

Un solo comando levanta la API y Redis:

```bash
docker compose up --build
```

La API queda en `http://localhost:8080` y Redis se conecta por la red interna de Compose.

Para definir un secreto propio en lugar del valor de desarrollo:

```bash
export JWT_SECRET="$(openssl rand -hex 32)"
docker compose up --build
```

En PowerShell:

```powershell
$env:JWT_SECRET = "un-secreto-largo-y-aleatorio-de-al-menos-32-caracteres"
docker compose up --build
```

El archivo SQLite se guarda en un volumen persistente (`gastos-data`), por lo que los datos sobreviven a reinicios de contenedores. Redis también usa un volumen para su persistencia AOF.

Detener los servicios:

```bash
docker compose down
```

Detener y eliminar también los volúmenes (borra la base de datos):

```bash
docker compose down -v
```

---

## 7. Cómo usarlo, paso a paso

Todos los ejemplos usan `curl`. Los importes se envían en **centavos**: `125050` equivale a `1250.50`.

Define la URL base:

```bash
export BASE_URL="http://localhost:8080/api/v1"
```

### 7.1 Registrarse

```bash
curl -s -X POST "$BASE_URL/auth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Ana García",
    "email": "ana@example.com",
    "password": "super-secret-123"
  }'
```

Respuesta:

```json
{
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "9f8c7b6a5d4e3f2a...",
    "token_type": "Bearer",
    "expires_in": 900,
    "user": {
      "id": 1,
      "name": "Ana García",
      "email": "ana@example.com",
      "created_at": "2026-11-08T00:00:00Z"
    }
  }
}
```

### 7.2 Iniciar sesión y guardar los tokens

```bash
LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "ana@example.com",
    "password": "super-secret-123"
  }')

export ACCESS_TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.data.access_token')
export REFRESH_TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.data.refresh_token')
```

Si no usas `jq`, copia los valores `access_token` y `refresh_token` de la respuesta manualmente.

### 7.3 Usar el access token

Todas las rutas protegidas requieren la cabecera `Authorization: Bearer <access_token>`:

```bash
curl -s "$BASE_URL/categories" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

### 7.4 Renovar los tokens con el refresh token

El refresh token se rota: el usado queda revocado y se devuelve uno nuevo.

```bash
REFRESH_RESPONSE=$(curl -s -X POST "$BASE_URL/auth/refresh" \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\":\"$REFRESH_TOKEN\"}")

export ACCESS_TOKEN=$(echo "$REFRESH_RESPONSE" | jq -r '.data.access_token')
export REFRESH_TOKEN=$(echo "$REFRESH_RESPONSE" | jq -r '.data.refresh_token')
```

Si intentas reutilizar el refresh token anterior, la API responde `401 UNAUTHORIZED` y revoca todas las sesiones del usuario por seguridad.

### 7.5 Cerrar sesión

```bash
curl -s -X POST "$BASE_URL/auth/logout" \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\":\"$REFRESH_TOKEN\"}"
```

### 7.6 Crear categorías

Categoría de gasto:

```bash
CATEGORY_RESPONSE=$(curl -s -X POST "$BASE_URL/categories" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Alimentación","type":"expense"}')

export FOOD_CATEGORY_ID=$(echo "$CATEGORY_RESPONSE" | jq -r '.data.id')
```

Categoría de ingreso:

```bash
SALARY_RESPONSE=$(curl -s -X POST "$BASE_URL/categories" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Salario","type":"income"}')

export SALARY_CATEGORY_ID=$(echo "$SALARY_RESPONSE" | jq -r '.data.id')
```

Listar categorías:

```bash
curl -s "$BASE_URL/categories" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

### 7.7 Registrar un gasto

```bash
curl -s -X POST "$BASE_URL/transactions" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"category_id\": $FOOD_CATEGORY_ID,
    \"type\": \"expense\",
    \"amount\": 125050,
    \"description\": \"Compra semanal\",
    \"date\": \"2026-11-08\"
  }"
```

### 7.8 Registrar un ingreso

```bash
curl -s -X POST "$BASE_URL/transactions" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"category_id\": $SALARY_CATEGORY_ID,
    \"type\": \"income\",
    \"amount\": 2500000,
    \"description\": \"Salario de noviembre\",
    \"date\": \"2026-11-01\"
  }"
```

### 7.9 Filtrar transacciones

```bash
curl -s "$BASE_URL/transactions?from=2026-11-01&to=2026-11-30&category_id=$FOOD_CATEGORY_ID&type=expense&min_amount=1000&max_amount=200000&page=1&limit=20" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

Todos los filtros son opcionales. `page` vale `1` por defecto y `limit` vale `20` (máximo `100`).

### 7.10 Crear un presupuesto

```bash
curl -s -X POST "$BASE_URL/budgets" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"category_id\": $FOOD_CATEGORY_ID,
    \"month\": 11,
    \"year\": 2026,
    \"limit_amount\": 500000
  }"
```

Listar presupuestos con gasto y estado:

```bash
curl -s "$BASE_URL/budgets?month=11&year=2026" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

### 7.11 Consultar el resumen mensual

```bash
curl -s "$BASE_URL/reports/summary?month=11&year=2026" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

Si omites `month` y `year`, se usa el mes actual.

### 7.12 Consultar la tendencia

```bash
curl -s "$BASE_URL/reports/trend?months=6" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

`months` es opcional, vale `6` por defecto y admite entre `1` y `24`.

### 7.13 Exportar transacciones a CSV

```bash
curl -s "$BASE_URL/transactions/export?from=2026-11-01&to=2026-11-30" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -o transacciones.csv
```

Acepta los mismos filtros que el listado (excepto `page` y `limit`). El CSV incluye `id`, `date`, `type`, `category`, `amount` y `description`. El monto se exporta en formato decimal legible (por ejemplo `1250.50`).

---

## 8. Documentación de endpoints

Prefijo de la API: `/api/v1`. Todas las rutas salvo `/auth/*`, `/health` y `/swagger/*` requieren `Authorization: Bearer <access_token>`.

### 8.1 Autenticación

| Método | Ruta | Auth | Descripción |
| --- | --- | --- | --- |
| `POST` | `/api/v1/auth/register` | No | Crea un usuario y devuelve tokens. Rate limit: 5/min por IP. |
| `POST` | `/api/v1/auth/login` | No | Valida credenciales y devuelve tokens. Rate limit: 5/min por IP. |
| `POST` | `/api/v1/auth/refresh` | No | Rota el refresh token y devuelve un par nuevo. |
| `POST` | `/api/v1/auth/logout` | No | Revoca el refresh token indicado. |

Cuerpos:

```json
// POST /auth/register
{"name": "Ana García", "email": "ana@example.com", "password": "super-secret-123"}

// POST /auth/login
{"email": "ana@example.com", "password": "super-secret-123"}

// POST /auth/refresh y /auth/logout
{"refresh_token": "9f8c7b6a5d4e3f2a..."}
```

Respuesta exitosa de registro/login/refresh:

```json
{
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "9f8c7b6a5d4e3f2a...",
    "token_type": "Bearer",
    "expires_in": 900,
    "user": {"id": 1, "name": "Ana García", "email": "ana@example.com", "created_at": "2026-11-08T00:00:00Z"}
  }
}
```

### 8.2 Categorías

| Método | Ruta | Auth | Descripción |
| --- | --- | --- | --- |
| `GET` | `/api/v1/categories` | Sí | Lista las categorías del usuario. |
| `POST` | `/api/v1/categories` | Sí | Crea una categoría. |
| `PUT` | `/api/v1/categories/{id}` | Sí | Actualiza nombre y tipo. |
| `DELETE` | `/api/v1/categories/{id}` | Sí | Elimina una categoría sin transacciones. |

Cuerpo de creación/actualización:

```json
{"name": "Alimentación", "type": "expense"}
```

Ejemplo de respuesta:

```json
{
  "data": [
    {"id": 1, "name": "Alimentación", "type": "expense", "created_at": "2026-11-08T00:00:00Z"},
    {"id": 2, "name": "Salario", "type": "income", "created_at": "2026-11-08T00:00:00Z"}
  ]
}
```

### 8.3 Transacciones

| Método | Ruta | Auth | Descripción |
| --- | --- | --- | --- |
| `GET` | `/api/v1/transactions` | Sí | Lista paginada con filtros. |
| `POST` | `/api/v1/transactions` | Sí | Registra un ingreso o gasto. |
| `GET` | `/api/v1/transactions/{id}` | Sí | Obtiene una transacción. |
| `PUT` | `/api/v1/transactions/{id}` | Sí | Actualiza una transacción. |
| `DELETE` | `/api/v1/transactions/{id}` | Sí | Elimina una transacción. |
| `GET` | `/api/v1/transactions/export` | Sí | Descarga CSV con los mismos filtros. |

Parámetros de listado/exportación:

| Parámetro | Tipo | Descripción |
| --- | --- | --- |
| `from` | fecha `YYYY-MM-DD` | Fecha inicial inclusive. |
| `to` | fecha `YYYY-MM-DD` | Fecha final inclusive. |
| `category_id` | entero | Filtra por categoría. |
| `type` | `income` \| `expense` | Filtra por tipo. |
| `min_amount` | entero | Monto mínimo en centavos. |
| `max_amount` | entero | Monto máximo en centavos. |
| `page` | entero | Página, por defecto `1`. |
| `limit` | entero | Tamaño de página, por defecto `20`, máximo `100`. |

Cuerpo de creación/actualización:

```json
{
  "category_id": 1,
  "type": "expense",
  "amount": 125050,
  "description": "Compra semanal",
  "date": "2026-11-08"
}
```

Respuesta paginada:

```json
{
  "data": [
    {
      "id": 10,
      "category_id": 1,
      "category_name": "Alimentación",
      "type": "expense",
      "amount": 125050,
      "description": "Compra semanal",
      "date": "2026-11-08",
      "created_at": "2026-11-08T00:00:00Z",
      "updated_at": "2026-11-08T00:00:00Z"
    }
  ],
  "meta": {"page": 1, "limit": 20, "total": 1}
}
```

### 8.4 Presupuestos

| Método | Ruta | Auth | Descripción |
| --- | --- | --- | --- |
| `GET` | `/api/v1/budgets` | Sí | Lista presupuestos con gasto y estado. |
| `POST` | `/api/v1/budgets` | Sí | Crea un presupuesto mensual. |
| `PUT` | `/api/v1/budgets/{id}` | Sí | Actualiza un presupuesto. |
| `DELETE` | `/api/v1/budgets/{id}` | Sí | Elimina un presupuesto. |

Parámetros de listado: `month` (1-12) y `year` (2000-2100). Deben enviarse juntos o ninguno; sin filtros se listan todos los periodos.

Cuerpo:

```json
{"category_id": 1, "month": 11, "year": 2026, "limit_amount": 500000}
```

Respuesta:

```json
{
  "data": [
    {
      "id": 1,
      "category_id": 1,
      "category_name": "Alimentación",
      "month": 11,
      "year": 2026,
      "limit_amount": 500000,
      "spent": 425000,
      "remaining": 75000,
      "percentage_used": 85,
      "status": "warning",
      "created_at": "2026-11-08T00:00:00Z"
    }
  ]
}
```

### 8.5 Reportes

| Método | Ruta | Auth | Descripción |
| --- | --- | --- | --- |
| `GET` | `/api/v1/reports/summary` | Sí | Totales del mes y gasto por categoría. |
| `GET` | `/api/v1/reports/trend` | Sí | Ingresos, gastos y balance de los últimos N meses. |

Parámetros:

| Endpoint | Parámetro | Descripción |
| --- | --- | --- |
| `summary` | `month` | Mes 1-12. Por defecto, mes actual. |
| `summary` | `year` | Año 2000-2100. Por defecto, año actual. |
| `trend` | `months` | Cantidad de meses 1-24. Por defecto, `6`. |

Respuesta de `summary`:

```json
{
  "data": {
    "month": 11,
    "year": 2026,
    "total_income": 2500000,
    "total_expense": 1250500,
    "balance": 1249500,
    "by_category": [
      {"category_id": 1, "category_name": "Alimentación", "amount": 1250500, "percentage": 100}
    ]
  }
}
```

Respuesta de `trend`:

```json
{
  "data": {
    "months": 3,
    "trend": [
      {"month": "2026-09", "income": 0, "expense": 0, "balance": 0},
      {"month": "2026-10", "income": 2500000, "expense": 1100000, "balance": 1400000},
      {"month": "2026-11", "income": 2500000, "expense": 1250500, "balance": 1249500}
    ]
  }
}
```

### 8.6 Sistema

| Método | Ruta | Auth | Descripción |
| --- | --- | --- | --- |
| `GET` | `/health` | No | Comprueba SQLite y, si aplica, Redis. |
| `GET` | `/swagger/index.html` | No | Documentación interactiva Swagger. |

Respuesta de `health`:

```json
{
  "data": {
    "status": "ok",
    "checks": {"database": "up", "redis": "up"}
  }
}
```

Si Redis está deshabilitado, `checks.redis` es `disabled`. Si Redis está configurado pero caído, `status` pasa a `degraded`, pero la API sigue respondiendo `200` porque SQLite sigue operativo.

---

## 9. Reglas de negocio importantes

### Importes y fechas

- Todos los importes son `int64` en **centavos**; nunca se usan `float` para dinero.
- `amount` y `limit_amount` deben ser mayores que cero.
- Las fechas de transacciones usan el formato `YYYY-MM-DD`.

### Categorías

- El tipo solo puede ser `income` o `expense`.
- El nombre es único por usuario y tipo.
- Una categoría con transacciones asociadas no puede eliminarse (`409 CONFLICT`).
- Tampoco puede cambiarse el tipo de una categoría que ya tiene transacciones, porque rompería la coherencia de esos movimientos.

### Transacciones

- El tipo de la transacción debe coincidir con el tipo de su categoría.
- La categoría debe existir y pertenecer al usuario autenticado.
- Cada usuario solo puede ver, editar o borrar sus propias transacciones.
- Un recurso de otro usuario responde `404 NOT_FOUND` para no revelar su existencia.
- `from` no puede ser posterior a `to`, ni `min_amount` mayor que `max_amount`.

### Presupuestos

- Solo se permiten para categorías de tipo `expense`.
- Son únicos por `(usuario, categoría, mes, año)`.
- `spent` se obtiene con una agregación SQL de las transacciones de gasto del mes.
- `remaining = limit_amount - spent`; puede ser negativo si se superó el límite.
- Estado:
  - `ok`: porcentaje usado menor que `80%`.
  - `warning`: mayor o igual que `80%` y menor que `100%`.
  - `exceeded`: mayor o igual que `100%`.

### Reportes

- Los totales y agrupaciones se calculan con agregaciones SQL, no en memoria.
- `summary` devuelve `total_income`, `total_expense`, `balance` y `by_category` con el porcentaje de cada categoría sobre el gasto total.
- `trend` rellena con ceros los meses sin movimientos y devuelve siempre el número de meses solicitado.
- Los reportes se cachean 10 minutos en Redis, si está disponible.

### Tokens

- Access token: 15 minutos por defecto (`ACCESS_TOKEN_TTL`).
- Refresh token: 7 días por defecto (`REFRESH_TOKEN_TTL`).
- El refresh token se guarda solo como hash SHA-256; el valor en claro nunca se persiste.
- Cada refresh rota el token: el anterior se revoca y se emite uno nuevo.
- Reutilizar un refresh token revocado responde `401` y revoca todas las sesiones del usuario.
- `logout` es idempotente: revocar un token ya revocado o desconocido no produce error.
- Las contraseñas se hashean con bcrypt y nunca se devuelven en las respuestas.

### Caché y rate limiting

- Claves de caché:
  - `summary:{user_id}:{año}-{mes}` para el resumen mensual.
  - `trend:{user_id}:{months}` para la tendencia.
- TTL: 10 minutos.
- Crear, actualizar o eliminar una transacción invalida las claves `summary:{user_id}:*` y `trend:{user_id}:*`.
- Si Redis falla, se registra un warning y la operación continúa contra SQLite. La API nunca se cae por Redis.
- Rate limit: 5 intentos por minuto por IP **y por endpoint** en `login` y `register`; al excederlo responde `429` con `Retry-After`.
- Si Redis no está disponible, el rate limiting hace *fail-open*: permite la petición y deja un warning en el log.

### Paginación

- `page` por defecto: `1`.
- `limit` por defecto: `20`; máximo `100`.
- Las respuestas paginadas incluyen `meta` con `page`, `limit` y `total`.

---

## 10. Formato de errores y códigos HTTP

Todos los errores usan este formato:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "el campo email debe ser un email válido"
  }
}
```

| Código HTTP | `code` | Cuándo se usa |
| --- | --- | --- |
| `400` | `VALIDATION_ERROR` | Cuerpo o parámetros inválidos, formato incorrecto. |
| `401` | `UNAUTHORIZED` | Token ausente/inválido/expirado o credenciales incorrectas. |
| `404` | `NOT_FOUND` | Recurso inexistente o perteneciente a otro usuario. |
| `409` | `CONFLICT` | Email/categoría/presupuesto duplicado, borrado de categoría con transacciones. |
| `422` | `UNPROCESSABLE_ENTITY` | Regla de negocio incumplida: tipo no coincide, presupuesto sobre categoría de ingreso, etc. |
| `429` | `TOO_MANY_REQUESTS` | Se superó el rate limit de login/registro. |
| `500` | `INTERNAL_ERROR` | Error inesperado del servidor. No expone detalles internos. |

---

## 11. Cómo ver la documentación Swagger

Con la API en marcha, abre:

<http://localhost:8080/swagger/index.html>

Para probar rutas protegidas:

1. Ejecuta `POST /api/v1/auth/register` o `/login` desde Swagger.
2. Copia el `access_token`.
3. Pulsa el botón **Authorize**.
4. Escribe `Bearer <access_token>` (incluye la palabra `Bearer` y un espacio).
5. Prueba cualquier endpoint protegido.

La carpeta `docs/` está versionada. Si modificas anotaciones o handlers, regenera la documentación con:

```bash
go run github.com/swaggo/swag/cmd/swag@v1.16.3 init \
  -g cmd/api/main.go \
  -o docs \
  --parseDependency \
  --parseInternal
```

---

## 12. Cómo correr los tests

```bash
go test ./...
```

Con detalle:

```bash
go test ./... -v
```

Con cobertura:

```bash
go test ./... -cover
```

Los tests usan SQLite en memoria y Redis deshabilitado, así que **no requieren Docker ni Redis**. Cubren, entre otras cosas:

- Registro, login, refresh con rotación, reutilización y logout.
- Middleware JWT y aislamiento de datos entre usuarios.
- CRUD de transacciones, filtros, paginación y exportación CSV.
- CRUD de presupuestos y cálculo de `spent`, `remaining`, porcentaje y estado.
- Reportes `summary` y `trend` con agregaciones SQL.
- Hash de refresh tokens y expiración.

---

## 13. Comandos rápidos de referencia

```bash
# Desarrollo local
go run ./cmd/api

# Build
go build ./...

# Tests
go test ./...

# Docker
docker compose up --build
docker compose down

# Regenerar Swagger
go run github.com/swaggo/swag/cmd/swag@v1.16.3 init -g cmd/api/main.go -o docs --parseDependency --parseInternal
```