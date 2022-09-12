package fiber

import (
	apierrors "github.com/synnaxlabs/synnax/pkg/api/errors"
	"github.com/cockroachdb/errors"
	"github.com/gofiber/fiber/v2"
)

func ErrorHandler(c *fiber.Ctx, err error) error {
	var (
		e    *fiber.Error
		tErr apierrors.Typed
	)
	if errors.As(err, &tErr) {
		return nil
	}
	if errors.As(err, &e) {
		if e.Code == fiber.StatusNotFound {
			tErr = apierrors.Route(err, c.Path())
		}
	} else {
		tErr = apierrors.Unexpected(err)
	}
	return errorResponse(c, tErr)
}
