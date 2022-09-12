package server

import (
	"context"
	fiberapi "github.com/synnaxlabs/synnax/pkg/api/fiber"
	"github.com/synnaxlabs/x/signal"
	"github.com/gofiber/fiber/v2"
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
