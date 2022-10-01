package server

import (
	"context"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	fiberapi "github.com/synnaxlabs/synnax/pkg/api/http"
	"github.com/synnaxlabs/x/signal"
	"net"
)

type fiberServer struct {
	app *fiber.App
	api fiberapi.API
}

func newFiberServer(cfg Config) *fiberServer {
	f := &fiberServer{}
	f.app = fiber.New(fiber.Config{
		DisableStartupMessage: true,
		ErrorHandler:          fiberapi.ErrorHandler,
	})
	f.app.Use(cors.New(cors.Config{AllowOrigins: "*"}))
	f.api = cfg.FiberAPI
	f.api.Route(f.app)
	return f
}
func (f fiberServer) start(
	ctx signal.Context,
	lis net.Listener,
) {
	ctx.Go(func(ctx context.Context) error {
		if err := f.app.Listener(lis); !isCloseErr(err) {
			return err
		}
		return nil
	}, signal.WithKey("server.fiber"), signal.CancelOnExit())
}

func (f fiberServer) Stop() error { return f.app.Shutdown() }
