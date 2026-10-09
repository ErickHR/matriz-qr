import { MATRIX_ERROR, calculateStatistics } from '../services/statistics.service.js';

export function getStatistics(request, response) {
  const { operation, matrices } = request.body ?? {};

  try {
    return response.status(200).json({
      operation,
      statistics: calculateStatistics(matrices)
    });
  } catch (error) {
    if (error instanceof TypeError && error.message === MATRIX_ERROR) {
      return response.status(400).json({ error: MATRIX_ERROR });
    }

    throw error;
  }
}
