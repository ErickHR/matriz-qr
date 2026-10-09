package services

import (
	"errors"
	"fmt"
	"math"

	"gonum.org/v1/gonum/mat"
	"matrix-api/models"
)

var ErrInvalidMatrix = errors.New("la matriz debe ser un arreglo rectangular no vacío de números finitos")

type MatrixService struct{}

func (MatrixService) Transform(request models.ProcessRequest) (models.TransformResult, error) {
	matrix, err := validateMatrix(request.Matrix)
	if err != nil {
		return models.TransformResult{}, err
	}

	switch request.Operation {
	case "rotate":
		return models.TransformResult{
			Operation: request.Operation,
			Matrices:  map[string]models.Matrix{"result": rotateClockwise(matrix)},
		}, nil
	case "qr":
		q, r, err := factorizeQR(matrix)
		if err != nil {
			return models.TransformResult{}, err
		}
		return models.TransformResult{
			Operation: request.Operation,
			Matrices:  map[string]models.Matrix{"Q": q, "R": r},
		}, nil
	default:
		return models.TransformResult{}, fmt.Errorf("operación no soportada: %s", request.Operation)
	}
}

// validateMatrix comprueba la matriz recibida y la convierte a models.Matrix.
func validateMatrix(raw models.RawMatrix) (models.Matrix, error) {
	if len(raw) == 0 || len(raw[0]) == 0 {
		return nil, ErrInvalidMatrix
	}

	columns := len(raw[0])
	matrix := make(models.Matrix, len(raw))
	for rowIndex, row := range raw {
		if len(row) != columns {
			return nil, ErrInvalidMatrix
		}
		matrix[rowIndex] = make([]float64, columns)
		for columnIndex, value := range row {
			if value == nil || math.IsNaN(*value) || math.IsInf(*value, 0) {
				return nil, ErrInvalidMatrix
			}
			matrix[rowIndex][columnIndex] = *value
		}
	}

	return matrix, nil
}

func rotateClockwise(matrix models.Matrix) models.Matrix {
	rows, columns := len(matrix), len(matrix[0])
	rotated := make(models.Matrix, columns)

	for column := 0; column < columns; column++ {
		rotated[column] = make([]float64, rows)
		for row := 0; row < rows; row++ {
			rotated[column][rows-1-row] = matrix[row][column]
		}
	}

	return rotated
}

func factorizeQR(matrix models.Matrix) (models.Matrix, models.Matrix, error) {
	rows, columns := len(matrix), len(matrix[0])
	if rows < columns {
		return nil, nil, errors.New("la factorización QR requiere que la cantidad de filas sea mayor o igual a la cantidad de columnas")
	}

	values := make([]float64, 0, rows*columns)
	for _, row := range matrix {
		values = append(values, row...)
	}

	source := mat.NewDense(rows, columns, values)
	var decomposition mat.QR
	decomposition.Factorize(source)

	var fullQ, fullR mat.Dense
	decomposition.QTo(&fullQ)
	decomposition.RTo(&fullR)

	q := make(models.Matrix, rows)
	for row := 0; row < rows; row++ {
		q[row] = make([]float64, columns)
		for column := 0; column < columns; column++ {
			q[row][column] = fullQ.At(row, column)
		}
	}

	r := make(models.Matrix, columns)
	for row := 0; row < columns; row++ {
		r[row] = make([]float64, columns)
		for column := 0; column < columns; column++ {
			r[row][column] = fullR.At(row, column)
		}
	}

	return q, r, nil
}
