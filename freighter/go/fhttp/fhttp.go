package fhttp

import "github.com/gofiber/fiber/v2"

type BindableTransport interface {
	BindTo(app *fiber.App)
}
