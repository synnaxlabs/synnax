package main

import (
	"github.com/arya-analytics/freighter/integration/server"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

func main() {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	w := server.Websocket{Logger: zap.S()}
	w.BindTo(app)
	if err := app.Listen(":8080"); err != nil {
		zap.S().Fatalw("server failed", "err", err)
	}
}
