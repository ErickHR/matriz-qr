package routes

import (
	"github.com/gofiber/fiber/v2"
	"matrix-api/controllers"
)

func Register(app *fiber.App, processController controllers.ProcessController) {
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	app.Post("/api/process", processController.Process)
}
