package server

import (
	"github.com/cockroachdb/cmux"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/synnaxlabs/freighter/fhttp"
)

type HTTPBranch struct {
	app        *fiber.App
	Transports []fhttp.BindableTransport
}

func (f *HTTPBranch) Key() string { return "http" }

func (f *HTTPBranch) Serve(cfg BranchConfig) error {
	lis := cfg.Mux.Match(cmux.HTTP1Fast())
	f.app = fiber.New(fiber.Config{DisableStartupMessage: true})
	f.app.Use(cors.New(cors.Config{AllowOrigins: "*"}))
	for _, t := range f.Transports {
		t.BindTo(f.app)
	}
	return filterCloseError(f.app.Listener(lis))
}

func (f *HTTPBranch) Stop() { _ = f.app.Shutdown() }
