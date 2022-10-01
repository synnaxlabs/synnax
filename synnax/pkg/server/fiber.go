package server

import (
	"context"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/synnaxlabs/freighter/fhttp"
	httpapi "github.com/synnaxlabs/synnax/pkg/api/http"
	"github.com/synnaxlabs/x/signal"
	"net"
)

type fiberServer struct {
	app *fiber.App
}

func newFiberServer(cfg Config) *fiberServer {
	f := &fiberServer{}
	f.app = fiber.New(fiber.Config{DisableStartupMessage: true})
	f.app.Use(cors.New(cors.Config{AllowOrigins: "*"}))
	httpapi.New(fhttp.NewRouter(fhttp.RouterConfig{
		App:    f.app,
		Logger: cfg.Logger.Sugar(),
	}))
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
