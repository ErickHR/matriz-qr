package models

type Matrix [][]float64

// RawMatrix usa punteros para distinguir un null del JSON (nil) de un 0 real.
type RawMatrix [][]*float64

type ProcessRequest struct {
	Operation string    `json:"operation"`
	Matrix    RawMatrix `json:"matrix"`
}

type TransformResult struct {
	Operation string            `json:"operation"`
	Matrices  map[string]Matrix `json:"matrices"`
}
