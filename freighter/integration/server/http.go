package server

import (
	"github.com/gofiber/fiber/v2"
	"github.com/synnaxlabs/freighter/ferrors"
)

type HTTP struct{}

func (h HTTP) BindTo(f fiber.Router) {
	router := f.Group("http")
	router.Get("/echo", httpGETEcho)
	router.Post("/echo", httpPOSTEcho)
}

func httpGETEcho(c *fiber.Ctx) error {
	msg := Message{}
	if err := c.QueryParser(&msg); err != nil {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(ferrors.Encode(TestError{Code: 1, Message: "unable to parse body"}))
	}
	msg.ID++
	return c.JSON(msg)
}

func httpPOSTEcho(c *fiber.Ctx) error {
	msg := Message{}
	if err := c.BodyParser(&msg); err != nil {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(ferrors.Encode(TestError{Code: 1, Message: "unable to parse body"}))
	}
	msg.ID++
	return c.JSON(msg)
}
