package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"matrix-api/clients"
	"matrix-api/config"
	"matrix-api/controllers"
	"matrix-api/routes"
	"matrix-api/services"
)

func main() {
	appConfig, err := config.Load()
	if err != nil {
		log.Fatalf("load configuration: %v", err)
	}

	app := fiber.New()
	processController := controllers.ProcessController{
		MatrixService: services.MatrixService{},
		Statistics: clients.StatisticsClient{
			URL:        appConfig.StatisticsURL,
			HTTPClient: &http.Client{Timeout: 5 * time.Second},
		},
	}
	routes.Register(app, processController)

	log.Fatal(app.Listen(":" + appConfig.Port))
}
