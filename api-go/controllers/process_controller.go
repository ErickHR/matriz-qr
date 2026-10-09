package controllers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"matrix-api/models"
	"matrix-api/services"
)

type StatisticsCalculator interface {
	Calculate(ctx context.Context, operation string, matrices map[string]models.Matrix) (json.RawMessage, error)
}

type ProcessController struct {
	MatrixService services.MatrixService
	Statistics    StatisticsCalculator
}

func (controller ProcessController) Process(c *fiber.Ctx) error {
	var request models.ProcessRequest
	if err := c.BodyParser(&request); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "el cuerpo de la solicitud debe ser un JSON válido"})
	}

	transformed, err := controller.MatrixService.Transform(request)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	statistics, err := controller.Statistics.Calculate(context.Background(), transformed.Operation, transformed.Matrices)
	if err != nil {
		return c.Status(http.StatusBadGateway).JSON(fiber.Map{"error": "el servicio de estadísticas no está disponible"})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"operation":           transformed.Operation,
		"transformedMatrices": transformed.Matrices,
		"statistics":          statistics,
	})
}
