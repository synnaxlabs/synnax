package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/synnaxlabs/freighter/integration/server"
	"go.uber.org/zap"
)

func main() {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	ws := server.Websocket{Logger: zap.S()}
	ws.BindTo(app)
	http := server.HTTP{}
	http.BindTo(app)
	if err := app.Listen(":8080"); err != nil {
		zap.S().Fatalw("server failed", "err", err)
	}
}
