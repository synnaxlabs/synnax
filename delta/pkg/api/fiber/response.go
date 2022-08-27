package fiber

import (
	errors "github.com/arya-analytics/delta/pkg/api/errors"
	"github.com/arya-analytics/freighter/ferrors"
	"github.com/gofiber/fiber/v2"
)

func errorResponse(c *fiber.Ctx, err errors.Typed) error {
	switch err.Type {
	case errors.TypeValidation:
		c.Status(fiber.StatusBadRequest)
	case errors.TypeParse:
		c.Status(fiber.StatusBadRequest)
	case errors.TypeGeneral:
		c.Status(fiber.StatusBadRequest)
	case errors.TypeUnexpected:
		c.Status(fiber.StatusInternalServerError)
	case errors.TypeNil:
		c.Status(fiber.StatusInternalServerError)
	}
	encoding := c.Get("Error-Encoding", "")
	if encoding == "freighter" {
		return c.JSON(ferrors.Encode(err))
	}
	return c.JSON(fiber.Map{"error": err})
}

func goodResponse(c *fiber.Ctx, v any) error {
	switch c.Method() {
	case fiber.MethodPost:
		c.Status(fiber.StatusCreated)
	default:
		c.Status(fiber.StatusOK)
	}
	return c.JSON(v)
}

func maybeGoodResponse(c *fiber.Ctx, err errors.Typed, v any) error {
	if !err.Occurred() {
		return goodResponse(c, v)
	}
	return errorResponse(c, err)
}

func maybeErrorResponse(c *fiber.Ctx, err errors.Typed) error {
	if !err.Occurred() {
		c.Status(fiber.StatusNoContent)
		return nil
	}
	return errorResponse(c, errors.MaybeUnexpected(err))
}
