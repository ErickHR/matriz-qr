export const MATRIX_ERROR = 'Cada matriz debe ser un arreglo rectangular no vacío de números finitos.';

function isPlainObject(value) {
  return value !== null && typeof value === 'object' && !Array.isArray(value);
}

function isMatrix(value) {
  if (!Array.isArray(value) || value.length === 0 || !Array.isArray(value[0]) || value[0].length === 0) {
    return false;
  }

  const columns = value[0].length;
  return value.every((row) =>
    Array.isArray(row) &&
    row.length === columns &&
    row.every((cell) => typeof cell === 'number' && Number.isFinite(cell))
  );
}

function calculateMatrixStatistics(matrix) {
  const values = matrix.flat();
  const sum = values.reduce((total, value) => total + value, 0);
  const rows = matrix.length;
  const columns = matrix[0].length;

  return {
    rows,
    columns,
    max: Math.max(...values),
    min: Math.min(...values),
    sum,
    average: sum / values.length,
    isDiagonal: rows === columns && matrix.every((row, rowIndex) =>
      row.every((value, columnIndex) => rowIndex === columnIndex || value === 0)
    )
  };
}

export function calculateStatistics(matrices) {
  if (!isPlainObject(matrices) || Object.keys(matrices).length === 0 || !Object.values(matrices).every(isMatrix)) {
    throw new TypeError(MATRIX_ERROR);
  }

  const perMatrix = Object.fromEntries(
    Object.entries(matrices).map(([name, matrix]) => [name, calculateMatrixStatistics(matrix)])
  );
  const values = Object.values(matrices).flat(2);
  const sum = values.reduce((total, value) => total + value, 0);

  return {
    perMatrix,
    global: {
      max: Math.max(...values),
      min: Math.min(...values),
      sum,
      average: sum / values.length,
      anyDiagonal: Object.values(perMatrix).some((statistics) => statistics.isDiagonal)
    }
  };
}
