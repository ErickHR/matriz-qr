# api-node

Servicio interno de estadísticas. Recibe una o varias matrices y calcula, para cada una y para todas en conjunto, el máximo, el mínimo, la suma, el promedio y si son diagonales.

Lo consume `api-go` después de transformar la matriz del usuario. No guarda datos: todo se calcula en memoria en cada petición.

> Para la arquitectura completa y cómo levantar todos los servicios juntos, ver el [README principal](../README.md).

## Contenido

1. [Tecnologías](#tecnologías)
2. [Estructura y capas](#estructura-y-capas)
3. [Flujo de una petición](#flujo-de-una-petición)
4. [Validación de la entrada](#validación-de-la-entrada)
5. [Cálculo de las estadísticas](#cálculo-de-las-estadísticas)
6. [Configuración](#configuración)
7. [Cómo ejecutarlo](#cómo-ejecutarlo)
8. [API](#api)
9. [Manejo de errores](#manejo-de-errores)

## Tecnologías

| Tecnología | Uso |
|---|---|
| Node.js 24 | Entorno de ejecución. El proyecto usa módulos ES (`"type": "module"`). |
| Express 5 | Servidor HTTP, rutas y lectura del cuerpo JSON (`express.json()`). |

No tiene más dependencias. Las variables de entorno se leen con `process.loadEnvFile`, que viene incluido en Node.js, así que no se necesita `dotenv`.

## Estructura y capas

El código está dividido en capas. Cada una tiene una sola responsabilidad y solo conoce a la capa siguiente:

```txt
src/
├── server.js                          # Punto de entrada: arranca el servidor
├── app.js                             # Crea la app de Express y registra las rutas
├── config/
│   └── env.js                         # Lee y valida la configuración
├── routes/
│   └── statistics.routes.js           # Define las rutas de /api/statistics
├── controllers/
│   └── statistics.controller.js       # Traduce HTTP ⇄ servicio
└── services/
    └── statistics.service.js          # Validación y cálculo (lógica de negocio)
```

| Archivo | Responsabilidad |
|---|---|
| `server.js` | Crea la app con `createApp()` y la pone a escuchar en el puerto configurado. Es el único archivo que abre un puerto. |
| `app.js` | Construye la app de Express: activa `express.json()`, registra `GET /health` y monta el router en `/api/statistics`. Se exporta como función (`createApp`) para poder crear la app sin abrir un puerto, por ejemplo en tests. |
| `config/env.js` | Carga `.env` si existe y valida que `PORT` sea un entero entre 1 y 65535. Si no es válido, el servicio no arranca. |
| `routes/statistics.routes.js` | Asocia `POST /` (es decir, `POST /api/statistics`) con el controlador. |
| `controllers/statistics.controller.js` | Extrae `operation` y `matrices` del cuerpo, llama al servicio y arma la respuesta HTTP. Convierte los errores de validación en `400`. |
| `services/statistics.service.js` | Valida las matrices y calcula las estadísticas. No sabe nada de HTTP ni de Express, así que se puede probar de forma aislada. |

## Flujo de una petición

```txt
POST /api/statistics
        │
        ▼
 express.json()            Convierte el cuerpo en objeto (400 si el JSON está mal formado)
        │
        ▼
 statistics.routes.js      Enruta a getStatistics
        │
        ▼
 statistics.controller.js  Toma operation y matrices del cuerpo
        │
        ▼
 statistics.service.js     calculateStatistics(matrices)
        │                    1. Valida cada matriz
        │                    2. Calcula las estadísticas de cada una (perMatrix)
        │                    3. Calcula las estadísticas de todas juntas (global)
        ▼
 statistics.controller.js  200 con { operation, statistics }
                           400 si la validación falla
```

## Validación de la entrada

Antes de calcular nada, `calculateStatistics` comprueba que:

1. `matrices` es un objeto (no un arreglo ni `null`) con al menos una matriz.
2. Cada matriz es un arreglo no vacío cuya primera fila también es un arreglo no vacío.
3. Todas las filas tienen la misma cantidad de columnas (la matriz es rectangular).
4. Cada valor es de tipo `number` y es finito (se rechazan `NaN`, `Infinity`, `null`, cadenas, etc.).

Si alguna regla falla, lanza un `TypeError` con el mensaje de `MATRIX_ERROR`, que el controlador convierte en un `400`.

## Cálculo de las estadísticas

**Por matriz (`perMatrix`)**: para cada matriz recibida se calcula:

| Campo | Cómo se calcula |
|---|---|
| `rows`, `columns` | Cantidad de filas y de columnas de la primera fila. |
| `max`, `min` | `Math.max` y `Math.min` sobre todos los valores de la matriz aplanada. |
| `sum` | Suma de todos los valores. |
| `average` | `sum` dividido entre la cantidad de valores (`rows × columns`). |
| `isDiagonal` | `true` si la matriz es cuadrada y todo valor fuera de la diagonal principal es `0`. Una matriz de 1×1 siempre es diagonal. |

**Global (`global`)**: se juntan los valores de todas las matrices en una sola lista y se calcula:

| Campo | Cómo se calcula |
|---|---|
| `max`, `min`, `sum` | Igual que por matriz, pero sobre todos los valores juntos. |
| `average` | Suma total dividida entre la cantidad total de valores. **No** es el promedio de los promedios. Por ejemplo, con `Q` (6 valores, suma 2) y `R` (4 valores, suma 2), el promedio global es `4 / 10 = 0.4`, no `(0.33 + 0.5) / 2`. |
| `anyDiagonal` | `true` si al menos una matriz tiene `isDiagonal: true`. |

## Configuración

| Variable | Valor por defecto | Descripción |
|---|---|---|
| `PORT` | `3001` | Puerto en el que escucha el servicio. Debe ser un entero entre 1 y 65535. |

En local se puede crear un `.env` a partir de `.env.example`:

```bash
cp .env.example .env
```

`.env` está en `.gitignore` y no debe subirse al repositorio.

## Cómo ejecutarlo

### En local

Requisitos: Node.js 24 o superior.

```bash
npm install
npm start
```

| Script | Comando | Uso |
|---|---|---|
| `npm start` | `node src/server.js` | Ejecuta el servicio. |
| `npm run dev` | `node --watch src/server.js` | Ejecuta el servicio y lo reinicia al guardar cambios. |

Al arrancar muestra:

```txt
Matrix statistics service listening on port 3001
```

### Con Docker

El `Dockerfile` usa dos etapas para que la imagen final sea liviana:

1. **`build`**: instala solo las dependencias de producción con `npm ci --omit=dev`, respetando las versiones exactas de `package-lock.json`.
2. **`runtime`**: copia `node_modules`, `package.json` y `src/`, define `NODE_ENV=production` y `PORT=3001`, y arranca con `npm start`.

Para construir y ejecutar solo este servicio:

```bash
docker build -t api-node .
docker run --rm -p 3001:3001 api-node
```

Normalmente se levanta junto con el resto del proyecto con `docker compose up -d --build` desde la raíz. En ese caso `docker-compose.yml` le añade un healthcheck sobre `GET /health`, y `api-go` no arranca hasta que este servicio responde.

## API

| Método | Ruta | Descripción |
|---|---|---|
| `GET` | `/health` | Comprueba que el servicio está encendido. |
| `POST` | `/api/statistics` | Calcula las estadísticas de una o varias matrices. |

### `GET /health`

```bash
curl http://localhost:3001/health
```

```json
{ "status": "ok" }
```

### `POST /api/statistics`

**Cuerpo de la petición** (`Content-Type: application/json`)

| Campo | Tipo | Descripción |
|---|---|---|
| `operation` | `string` | Operación que generó las matrices (`rotate` o `qr`). No se valida; se devuelve tal cual. |
| `matrices` | `object` | Matrices identificadas por nombre. `api-go` envía `result` para una rotación, y `Q` y `R` para una factorización QR. |

#### Ejemplo 1: una matriz (rotación)

```bash
curl -X POST http://localhost:3001/api/statistics \
  -H "Content-Type: application/json" \
  -d '{"operation": "rotate", "matrices": {"result": [[4, 1], [5, 2], [6, 3]]}}'
```

```json
{
  "operation": "rotate",
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

#### Ejemplo 2: varias matrices (QR)

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

## Manejo de errores

| Código | Causa | Respuesta |
|---|---|---|
| `400` | `matrices` falta, está vacío, no es un objeto o alguna matriz no cumple las [reglas de validación](#validación-de-la-entrada). También ocurre si la petición no envía `Content-Type: application/json`, porque el cuerpo no se lee. | `{ "error": "Cada matriz debe ser un arreglo rectangular no vacío de números finitos." }` |
| `400` | El cuerpo no es un JSON bien formado. Lo detecta `express.json()` antes de llegar al controlador. | Página HTML de error de Express. |
| `404` | La ruta no existe. | Página HTML de error de Express. |
| `500` | Cualquier error inesperado. El controlador solo captura los errores de validación y deja pasar el resto al manejador por defecto de Express. | Página HTML de error de Express. |

Ejemplo de error de validación (matriz no rectangular):

```bash
curl -X POST http://localhost:3001/api/statistics \
  -H "Content-Type: application/json" \
  -d '{"operation": "rotate", "matrices": {"result": [[1, 2], [3]]}}'
```

```json
{ "error": "Cada matriz debe ser un arreglo rectangular no vacío de números finitos." }
```
