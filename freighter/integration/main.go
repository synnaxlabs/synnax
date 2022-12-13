package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/synnaxlabs/freighter/integration/server"
	"go.uber.org/zap"
)

func main() {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(logger.New())
	server.BindTo(app, zap.S())
	if err := app.Listen(":8080"); err != nil {
		zap.S().Fatalw("server failed", "err", err)
	}
}
