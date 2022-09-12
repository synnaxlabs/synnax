package fiber

import (
	errors "github.com/synnaxlabs/synnax/pkg/api/errors"
	"github.com/gofiber/fiber/v2"
)

func parseBody[V any](c *fiber.Ctx, v *V) errors.Typed {
	return errors.MaybeParse(c.BodyParser(v))
}

func parseQueryParams[V any](c *fiber.Ctx, v *V) errors.Typed {
	return errors.MaybeParse(c.QueryParser(v))
}
