package fiber

import (
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

func logMiddleware(log *zap.Logger) fiber.Handler {
	sugaredL := log.Sugar().Named("server.fiber")
	return func(c *fiber.Ctx) error {
		err := c.Next()
		logFunc(sugaredL, c.Response().StatusCode())("",
			"method", c.Method(),
			"path", c.Path(),
			"status", c.Response().StatusCode(),
		)
		return err
	}
}

func logFunc(logger *zap.SugaredLogger, status int) func(msg string, keysAndValues ...interface{}) {
	if status >= fiber.StatusInternalServerError {
		return logger.Errorw
	}
	return logger.Debugw
}
