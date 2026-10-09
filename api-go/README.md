# api-go

API principal del proyecto. Recibe una matriz desde el frontend, la valida, aplica la operación solicitada (**rotación** o **factorización QR**) y le pide a `api-node` las estadísticas de las matrices resultantes. Después devuelve todo en una sola respuesta.

> Para la arquitectura completa y cómo levantar todos los servicios juntos, ver el [README principal](../README.md).

## Contenido

1. [Tecnologías](#tecnologías)
2. [Estructura y capas](#estructura-y-capas)
3. [Flujo de una petición](#flujo-de-una-petición)
4. [Validación de la entrada](#validación-de-la-entrada)
5. [Operaciones](#operaciones)
6. [Comunicación con api-node](#comunicación-con-api-node)
7. [Configuración](#configuración)
8. [Cómo ejecutarlo](#cómo-ejecutarlo)
9. [API](#api)
10. [Manejo de errores](#manejo-de-errores)

## Tecnologías

| Tecnología | Uso |
|---|---|
| Go 1.24 | Lenguaje y compilador. |
| [Fiber v2](https://github.com/gofiber/fiber) | Servidor HTTP, rutas y lectura del cuerpo JSON. |
| [gonum](https://www.gonum.org/) | Álgebra lineal: cálculo de la factorización QR (`gonum.org/v1/gonum/mat`). |
| [godotenv](https://github.com/joho/godotenv) | Carga de variables de entorno desde `.env`. |

## Estructura y capas

El código está dividido en paquetes. Cada uno tiene una sola responsabilidad:

```txt
.
├── main.go                         # Punto de entrada: arma las dependencias y arranca el servidor
├── config/
│   └── config.go                   # Lee la configuración desde el entorno o .env
├── routes/
│   └── routes.go                   # Registra las rutas HTTP
├── controllers/
│   └── process_controller.go       # Traduce HTTP ⇄ servicio y cliente
├── services/
│   └── matrix_service.go           # Validación, rotación y QR (lógica de negocio)
├── clients/
│   └── statistics_client.go        # Cliente HTTP hacia api-node
└── models/
    └── matrix.go                   # Tipos compartidos
```

| Paquete | Responsabilidad |
|---|---|
| `main` | Carga la configuración, crea el servicio y el cliente de estadísticas (con un timeout de 5 s), los inyecta en el controlador, registra las rutas y arranca Fiber. Es el único lugar donde se conectan las piezas. |
| `config` | Carga `.env` si existe y devuelve un `Config` con `Port` y `StatisticsURL`, usando valores por defecto si las variables no están definidas. |
| `routes` | Asocia `GET /health` y `POST /api/process` con sus manejadores. |
| `controllers` | `ProcessController` lee el cuerpo, llama al servicio, llama al cliente de estadísticas y arma la respuesta. Decide el código HTTP de cada error. |
| `services` | `MatrixService` valida la matriz y ejecuta la operación. No sabe nada de HTTP, así que se puede probar de forma aislada. |
| `clients` | `StatisticsClient` envía las matrices a `api-node` y devuelve la parte `statistics` de su respuesta. |
| `models` | Tipos `RawMatrix` (`[][]*float64`, la matriz tal como llega en el JSON), `Matrix` (`[][]float64`, la matriz validada), `ProcessRequest` y `TransformResult`. |

El controlador no depende directamente de `StatisticsClient`, sino de la interfaz `StatisticsCalculator`:

```go
type StatisticsCalculator interface {
	Calculate(ctx context.Context, operation string, matrices map[string]models.Matrix) (json.RawMessage, error)
}
```

Esto permite reemplazar `api-node` por una implementación falsa en tests, sin levantar otro servicio.

## Flujo de una petición

```txt
POST /api/process
        │
        ▼
 routes.go                  Enruta a ProcessController.Process
        │
        ▼
 process_controller.go      BodyParser → ProcessRequest           (400 si el JSON no es válido)
        │
        ▼
 matrix_service.go          Transform(request)
        │                     1. validateMatrix                   (400 si la matriz no es válida)
        │                     2. rotate → rotateClockwise
        │                        qr     → factorizeQR             (400 si filas < columnas)
        │                        otra   → error                   (400 operación no soportada)
        ▼
 statistics_client.go       POST {NODE_STATISTICS_URL}            (502 si falla)
        │
        ▼
 process_controller.go      200 con { operation, transformedMatrices, statistics }
```

## Validación de la entrada

`validateMatrix` comprueba que:

1. La matriz tiene al menos una fila y la primera fila tiene al menos una columna.
2. Todas las filas tienen la misma cantidad de columnas (la matriz es rectangular).
3. Ningún valor es `null`, `NaN` ni infinito.

La matriz se lee como `RawMatrix` (`[][]*float64`). Así un `null` llega como puntero `nil` y se puede rechazar; si se leyera directamente como `[][]float64`, Go lo convertiría en `0` sin avisar. Si la validación pasa, `validateMatrix` devuelve la matriz ya convertida a `Matrix` (`[][]float64`), que es la que usan las operaciones.

Los valores que no son números (por ejemplo cadenas) los rechaza el parseo del JSON. En ese caso el error es el de JSON inválido.

## Operaciones

### `rotate`: rotación 90° en sentido horario

`rotateClockwise` crea una matriz nueva de `columnas × filas`. El valor en la fila `r` y la columna `c` de la original pasa a la fila `c` y la columna `filas - 1 - r` de la rotada:

```txt
 Original (2×3)        Rotada (3×2)

 1  2  3               4  1
 4  5  6      ──▶      5  2
                       6  3
```

Devuelve una sola matriz con la clave `result`.

### `qr`: factorización QR reducida

`factorizeQR` descompone la matriz `A` (m×n) en `A = Q · R`:

- `Q` (m×n): columnas ortonormales.
- `R` (n×n): triangular superior.

Pasos:

1. Comprueba que `m ≥ n`. Si no, devuelve un error.
2. Copia la matriz a un `mat.Dense` de gonum y ejecuta `mat.QR.Factorize`.
3. gonum devuelve la factorización **completa** (`Q` de m×m y `R` de m×n). Se recorta a la forma **reducida**: las primeras `n` columnas de `Q` y las primeras `n` filas de `R`.

Devuelve dos matrices con las claves `Q` y `R`.

No se comprueba que las columnas sean linealmente independientes. Si no lo son, la factorización igual se calcula y `R` tiene algún valor igual o muy cercano a `0` en la diagonal.

## Comunicación con api-node

`StatisticsClient.Calculate`:

1. Serializa `{ "operation": ..., "matrices": ... }` y lo envía con `POST` a `NODE_STATISTICS_URL`, con `Content-Type: application/json`.
2. Usa un `http.Client` con **timeout de 5 segundos**, configurado en `main.go`.
3. Si `api-node` no responde, responde con un código fuera de 2xx o devuelve un JSON que no se puede leer, devuelve un error.
4. Si todo va bien, devuelve solo el campo `statistics` de la respuesta, sin volver a decodificarlo (`json.RawMessage`). Así `api-go` no necesita conocer la forma de las estadísticas.

Cualquier error de este paso se convierte en un `502` en el controlador.

## Configuración

| Variable | Valor por defecto | Descripción |
|---|---|---|
| `PORT` | `3000` | Puerto en el que escucha la API. |
| `NODE_STATISTICS_URL` | `http://localhost:3001/api/statistics` | URL del endpoint de estadísticas de `api-node`. En Docker Compose se define como `http://api-node:3001/api/statistics`. |

Las variables se leen del entorno. Si existe un archivo `.env` en la carpeta desde la que se ejecuta, también se cargan desde ahí (sin pisar las que ya están definidas en el entorno):

```bash
cp .env.example .env
```

`.env` está en `.gitignore` y no debe subirse al repositorio.

## Cómo ejecutarlo

### En local

Requisitos: Go 1.24 o superior, y `api-node` en ejecución (por defecto en `http://localhost:3001`).

```bash
go mod tidy
go run .
```

`go mod tidy` descarga las dependencias y completa `go.mod`; hace falta ejecutarlo la primera vez.

Para generar un binario:

```bash
go build -o matrix-api .
./matrix-api
```

### Con Docker

El `Dockerfile` usa dos etapas:

1. **`build`** (`golang:1.24-alpine`): descarga las dependencias con `go mod tidy` y compila un binario estático (`CGO_ENABLED=0`).
2. **`runtime`** (`alpine:3.21`): solo copia el binario y lo ejecuta. La imagen final no incluye el compilador de Go ni el código fuente.

Para construir y ejecutar solo este servicio:

```bash
docker build -t api-go .
docker run --rm -p 3000:3000 \
  --add-host=host.docker.internal:host-gateway \
  -e NODE_STATISTICS_URL=http://host.docker.internal:3001/api/statistics \
  api-go
```

Normalmente se levanta junto con el resto del proyecto con `docker compose up -d --build` desde la raíz. En ese caso `api-go` espera a que el healthcheck de `api-node` esté en verde antes de arrancar.

## API

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

**Cuerpo de la petición** (`Content-Type: application/json`)

| Campo | Tipo | Descripción |
|---|---|---|
| `operation` | `string` | `rotate` o `qr`. |
| `matrix` | `number[][]` | Matriz rectangular, no vacía y con números finitos. Para `qr`, con igual o mayor cantidad de filas que columnas. |

**Cuerpo de la respuesta**

| Campo | Descripción |
|---|---|
| `operation` | Operación aplicada. |
| `transformedMatrices` | `result` para `rotate`; `Q` y `R` para `qr`. |
| `statistics` | Lo que devolvió `api-node`: `perMatrix` (estadísticas de cada matriz) y `global` (de todas juntas). Ver el [README de api-node](../api-node/README.md#cálculo-de-las-estadísticas). |

#### Ejemplo 1: rotar una matriz

```bash
curl -X POST http://localhost:3000/api/process \
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

```bash
curl -X POST http://localhost:3000/api/process \
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

## Manejo de errores

Los errores del controlador responden con `{ "error": "<mensaje>" }`.

| Código | Causa | Mensaje |
|---|---|---|
| `400` | El cuerpo no es un JSON válido o contiene valores que no son números | `el cuerpo de la solicitud debe ser un JSON válido` |
| `400` | Matriz vacía, no rectangular o con valores `null` o no finitos. También si falta `matrix`, o si la petición no envía `Content-Type: application/json` (el cuerpo no se lee) | `la matriz debe ser un arreglo rectangular no vacío de números finitos` |
| `400` | `operation` distinta de `rotate` o `qr`, o ausente | `operación no soportada: <operación>` |
| `400` | `qr` con menos filas que columnas | `la factorización QR requiere que la cantidad de filas sea mayor o igual a la cantidad de columnas` |
| `502` | `api-node` no responde, tarda más de 5 s o devuelve un error | `el servicio de estadísticas no está disponible` |
| `404` | La ruta no existe (respuesta de Fiber, en texto plano) | `Cannot GET /ruta` |
| `413` | El cuerpo supera el límite por defecto de Fiber (4 MB) (respuesta de Fiber, en texto plano) | `Request Entity Too Large` |

Ejemplo de error (QR con más columnas que filas):

```bash
curl -X POST http://localhost:3000/api/process \
  -H "Content-Type: application/json" \
  -d '{"operation": "qr", "matrix": [[1, 2, 3]]}'
```

```json
{ "error": "la factorización QR requiere que la cantidad de filas sea mayor o igual a la cantidad de columnas" }
```
