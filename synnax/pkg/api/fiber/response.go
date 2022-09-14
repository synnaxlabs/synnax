package fiber

import (
	"github.com/gofiber/fiber/v2"
	"github.com/synnaxlabs/freighter/ferrors"
	errors "github.com/synnaxlabs/synnax/pkg/api/errors"
)

const (
	ErrorEncodingHeaderKey = "Error-Encoding"
	ErrorEncodingFreighter = "freighter"
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
	case errors.TypeRoute:
		c.Status(fiber.StatusNotFound)
	case errors.TypeNil:
		c.Status(fiber.StatusInternalServerError)
	}
	encoding := c.Get(ErrorEncodingHeaderKey)
	if encoding == ErrorEncodingFreighter {
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
