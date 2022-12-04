package server

import (
	"github.com/cockroachdb/cmux"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/gofiber/fiber/v2/middleware/pprof"
	"github.com/synnaxlabs/freighter/fhttp"
	"github.com/synnaxlabs/x/telem"
)

type HTTPBranch struct {
	app        *fiber.App
	Transports []fhttp.BindableTransport
}

func (f *HTTPBranch) Match() []cmux.Matcher {
	return []cmux.Matcher{cmux.HTTP1Fast()}
}

func (f *HTTPBranch) Key() string { return "http" }

func (f *HTTPBranch) Serve(cfg BranchConfig) error {
	f.app = fiber.New(fiber.Config{
		DisableStartupMessage: true,
		ReadBufferSize:        int(10 * telem.Kilobyte),
	})
	f.app.Use(cors.New(cors.Config{AllowOrigins: "*"}))
	f.app.Get("/metrics", monitor.New(monitor.Config{Title: "Synnax Metrics"}))
	f.app.Use(pprof.New())
	for _, t := range f.Transports {
		t.BindTo(f.app)
	}
	return filterCloseError(f.app.Listener(cfg.Lis))
}

func (f *HTTPBranch) Stop() { _ = f.app.Shutdown() }
