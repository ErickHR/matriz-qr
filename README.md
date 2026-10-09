# Matriz QR

Aplicación web para procesar matrices numéricas. Permite **rotar una matriz 90° en sentido horario** o **calcular su factorización QR**, y devuelve las matrices resultantes junto con sus estadísticas (máximo, mínimo, suma, promedio y si es diagonal).

## Contenido

1. [Arquitectura](#arquitectura)
2. [Estructura del proyecto](#estructura-del-proyecto)
3. [Cómo levantar el proyecto](#cómo-levantar-el-proyecto)
4. [API Go](#api-go)
5. [API Node](#api-node)

## Arquitectura

```txt
                       ┌──────────────────────────┐
                       │      nginx  (:8080)      │
  Navegador ─────────▶ │                          │
                       │  /       ─▶ frontend     │
                       │  /api/*  ─▶ api-go       │
                       └─────┬──────────────┬─────┘
                             │              │
                             ▼              ▼
                     ┌────────────┐   ┌────────────┐  POST /api/statistics  ┌────────────┐
                     │  frontend  │   │   api-go   │ ─────────────────────▶ │  api-node  │
                     │ HTML/CSS/JS│   │   (:3000)  │ ◀───────────────────── │   (:3001)  │
                     └────────────┘   └────────────┘      estadísticas      └────────────┘
```

| Componente | Tecnología | Responsabilidad |
|---|---|---|
| `nginx` | nginx 1.27 | Punto de entrada único. Sirve el frontend y redirige `/api/` a `api-go`. |
| `frontend` | HTML, CSS y JavaScript | Formulario para ingresar la matriz, elegir la operación y ver el resultado. |
| `api-go` | Go 1.24, Fiber, gonum | API principal. Valida la matriz, aplica la operación (rotación o QR) y pide las estadísticas a `api-node`. |
| `api-node` | Node.js 24, Express 5 | API interna. Calcula las estadísticas de cada matriz y un resumen global. No guarda datos. |

**Flujo de una solicitud**

1. El usuario envía la matriz y la operación desde el frontend.
2. nginx recibe `POST /api/process` y lo redirige a `api-go`.
3. `api-go` valida la matriz y aplica la operación.
4. `api-go` envía las matrices resultantes a `api-node`, que calcula las estadísticas.
5. `api-go` responde con las matrices transformadas y sus estadísticas.

## Estructura del proyecto

```txt
.
├── api-go/                  # API principal (Go)
│   ├── clients/             # Cliente HTTP hacia api-node
│   ├── config/              # Lectura de variables de entorno
│   ├── controllers/         # Manejo de las peticiones HTTP
│   ├── models/              # Tipos de datos (matriz, petición, resultado)
│   ├── routes/              # Definición de rutas
│   ├── services/            # Validación, rotación y factorización QR
│   └── main.go
├── api-node/                # API de estadísticas (Node.js)
│   └── src/
│       ├── config/          # Lectura de variables de entorno
│       ├── controllers/
│       ├── routes/
│       ├── services/        # Cálculo de estadísticas
│       ├── app.js
│       └── server.js
├── frontend/
│   └── public/              # index.html, css/ y js/
├── nginx/
│   └── local.conf           # Configuración del proxy inverso
└── docker-compose.yml
```

## Cómo levantar el proyecto

### Con Docker (recomendado)

Requisitos: Docker y Docker Compose v2.

```bash
docker compose up -d --build
```

Cuando termine, estos servicios quedan disponibles:

| Servicio | URL |
|---|---|
| Aplicación web (frontend + API vía nginx) | http://localhost:8080 |
| API Go | http://localhost:3000 |
| API Node | http://localhost:3001 |

Si algún puerto está ocupado, se puede cambiar con las variables `NGINX_PORT`, `GO_PORT` y `NODE_PORT`:

```bash
NGINX_PORT=9090 docker compose up -d --build
```

Para detener todo:

```bash
docker compose down
```

### Sin Docker

Requisitos: Go 1.24 o superior y Node.js 24 o superior. Primero se levanta `api-node`, porque `api-go` depende de él.

```bash
# Terminal 1: API Node
cd api-node
npm install
npm start
```

```bash
# Terminal 2: API Go
cd api-go
go mod tidy
go run .
```

Cada servicio lee un archivo `.env` si existe; en cada carpeta hay un `.env.example` como referencia.

> El frontend llama a la API con la ruta relativa `/api/process`, así que necesita nginx delante. Para usar la interfaz web se recomienda levantar el proyecto con Docker.

### Variables de entorno

| Servicio | Variable | Valor por defecto | Descripción |
|---|---|---|---|
| `api-go` | `PORT` | `3000` | Puerto de la API. |
| `api-go` | `NODE_STATISTICS_URL` | `http://localhost:3001/api/statistics` | URL del endpoint de estadísticas de `api-node`. En Docker es `http://api-node:3001/api/statistics`. |
| `api-node` | `PORT` | `3001` | Puerto de la API. |

## API Go

Es la API que consume el frontend. A través de nginx está disponible en `http://localhost:8080/api/...`, y directamente en `http://localhost:3000/api/...`.

| Método | Ruta | Descripción |
|---|---|---|
| `GET` | `/health` | Comprueba que la API está encendida. |
| `POST` | `/api/process` | Rota la matriz o calcula su factorización QR, y devuelve sus estadísticas. |

### `GET /health`

```bash
curl http://localhost:3000/health
```

```json
{ "status": "ok" }
```

### `POST /api/process`

**Cuerpo de la petición**

| Campo | Tipo | Descripción |
|---|---|---|
| `operation` | `string` | `rotate` o `qr`. |
| `matrix` | `number[][]` | Matriz rectangular (todas las filas con el mismo número de columnas), no vacía y con números finitos. |

**Cuerpo de la respuesta**

| Campo | Descripción |
|---|---|
| `operation` | Operación aplicada. |
| `transformedMatrices` | Matrices resultantes. `rotate` devuelve `result`; `qr` devuelve `Q` y `R`. |
| `statistics.perMatrix` | Estadísticas de cada matriz resultante (ver [API Node](#post-apistatistics)). |
| `statistics.global` | Estadísticas calculadas con todos los valores de todas las matrices. |

#### Ejemplo 1: rotar una matriz

Rota la matriz 90° en sentido horario. Una matriz de 2×3 pasa a ser de 3×2.

```bash
curl -X POST http://localhost:8080/api/process \
  -H "Content-Type: application/json" \
  -d '{"operation": "rotate", "matrix": [[1, 2, 3], [4, 5, 6]]}'
```

```json
{
  "operation": "rotate",
  "transformedMatrices": {
    "result": [[4, 1], [5, 2], [6, 3]]
  },
  "statistics": {
    "perMatrix": {
      "result": {
        "rows": 3,
        "columns": 2,
        "max": 6,
        "min": 1,
        "sum": 21,
        "average": 3.5,
        "isDiagonal": false
      }
    },
    "global": {
      "max": 6,
      "min": 1,
      "sum": 21,
      "average": 3.5,
      "anyDiagonal": false
    }
  }
}
```

#### Ejemplo 2: factorización QR

Calcula la factorización QR reducida (`A = Q · R`) con la librería gonum. La matriz debe tener **igual o mayor cantidad de filas que columnas**. Para una matriz de m×n, `Q` es de m×n y `R` es de n×n y triangular superior.

```bash
curl -X POST http://localhost:8080/api/process \
  -H "Content-Type: application/json" \
  -d '{"operation": "qr", "matrix": [[1, 0], [0, 1], [0, 0]]}'
```

```json
{
  "operation": "qr",
  "transformedMatrices": {
    "Q": [[1, 0], [0, 1], [0, 0]],
    "R": [[1, 0], [0, 1]]
  },
  "statistics": {
    "perMatrix": {
      "Q": {
        "rows": 3,
        "columns": 2,
        "max": 1,
        "min": 0,
        "sum": 2,
        "average": 0.3333333333333333,
        "isDiagonal": false
      },
      "R": {
        "rows": 2,
        "columns": 2,
        "max": 1,
        "min": 0,
        "sum": 2,
        "average": 0.5,
        "isDiagonal": true
      }
    },
    "global": {
      "max": 1,
      "min": 0,
      "sum": 4,
      "average": 0.4,
      "anyDiagonal": true
    }
  }
}
```

#### Errores

Todos los errores responden con un objeto `{ "error": "<mensaje>" }`.

| Código | Causa | Mensaje |
|---|---|---|
| `400` | El cuerpo no es un JSON válido o contiene valores no numéricos | `el cuerpo de la solicitud debe ser un JSON válido` |
| `400` | Matriz vacía, no rectangular o con valores `null` o no finitos | `la matriz debe ser un arreglo rectangular no vacío de números finitos` |
| `400` | Operación distinta de `rotate` o `qr` | `operación no soportada: <operación>` |
| `400` | QR con menos filas que columnas | `la factorización QR requiere que la cantidad de filas sea mayor o igual a la cantidad de columnas` |
| `502` | `api-go` no puede comunicarse con `api-node` | `el servicio de estadísticas no está disponible` |

Ejemplo con una matriz no rectangular:

```bash
curl -X POST http://localhost:8080/api/process \
  -H "Content-Type: application/json" \
  -d '{"operation": "rotate", "matrix": [[1, 2], [3]]}'
```

```json
{ "error": "la matriz debe ser un arreglo rectangular no vacío de números finitos" }
```

## API Node

API interna de estadísticas. La consume `api-go` y no está expuesta a través de nginx; para probarla directamente se usa `http://localhost:3001`.

| Método | Ruta | Descripción |
|---|---|---|
| `GET` | `/health` | Comprueba que la API está encendida. |
| `POST` | `/api/statistics` | Calcula las estadísticas de una o varias matrices. |

### `GET /health`

```bash
curl http://localhost:3001/health
```

```json
{ "status": "ok" }
```

### `POST /api/statistics`

**Cuerpo de la petición**

| Campo | Tipo | Descripción |
|---|---|---|
| `operation` | `string` | Operación que generó las matrices. Se devuelve tal cual en la respuesta. |
| `matrices` | `object` | Una o varias matrices identificadas por nombre (por ejemplo `result`, o `Q` y `R`). Cada una debe ser rectangular, no vacía y con números finitos. |

**Estadísticas calculadas**

| Campo | Dónde | Descripción |
|---|---|---|
| `rows`, `columns` | `perMatrix` | Dimensiones de la matriz. |
| `max`, `min` | `perMatrix` y `global` | Mayor y menor valor. |
| `sum` | `perMatrix` y `global` | Suma de todos los valores. |
| `average` | `perMatrix` y `global` | Suma dividida entre la cantidad de valores. En `global` se calcula con todos los valores juntos, no es el promedio de los promedios. |
| `isDiagonal` | `perMatrix` | `true` si la matriz es cuadrada y todos los valores fuera de la diagonal principal son `0`. |
| `anyDiagonal` | `global` | `true` si al menos una de las matrices es diagonal. |

#### Ejemplo

```bash
curl -X POST http://localhost:3001/api/statistics \
  -H "Content-Type: application/json" \
  -d '{"operation": "qr", "matrices": {"Q": [[1, 0], [0, 1], [0, 0]], "R": [[1, 0], [0, 1]]}}'
```

```json
{
  "operation": "qr",
  "statistics": {
    "perMatrix": {
      "Q": {
        "rows": 3,
        "columns": 2,
        "max": 1,
        "min": 0,
        "sum": 2,
        "average": 0.3333333333333333,
        "isDiagonal": false
      },
      "R": {
        "rows": 2,
        "columns": 2,
        "max": 1,
        "min": 0,
        "sum": 2,
        "average": 0.5,
        "isDiagonal": true
      }
    },
    "global": {
      "max": 1,
      "min": 0,
      "sum": 4,
      "average": 0.4,
      "anyDiagonal": true
    }
  }
}
```

#### Errores

| Código | Causa | Mensaje |
|---|---|---|
| `400` | `matrices` falta, está vacío o alguna matriz no es válida | `Cada matriz debe ser un arreglo rectangular no vacío de números finitos.` |
