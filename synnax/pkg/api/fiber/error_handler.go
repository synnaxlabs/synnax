package fiber

import (
	"github.com/cockroachdb/errors"
	"github.com/gofiber/fiber/v2"
	"github.com/samber/lo"
	apierrors "github.com/synnaxlabs/synnax/pkg/api/errors"
)

var routeErrorStatuses = []int{
	fiber.StatusNotFound,
	fiber.StatusMethodNotAllowed,
}

func ErrorHandler(c *fiber.Ctx, err error) error {
	var (
		e    *fiber.Error
		tErr apierrors.Typed
	)
	if errors.As(err, &tErr) {
		return nil
	}
	if errors.As(err, &e) {
		if lo.Contains(routeErrorStatuses, e.Code) {
			tErr = apierrors.Route(err, c.Path())
		}
	} else {
		tErr = apierrors.Unexpected(err)
	}
	return errorResponse(c, tErr)
}
