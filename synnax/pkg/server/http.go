package server

import (
	"bytes"
	"github.com/cockroachdb/cmux"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/gofiber/fiber/v2/middleware/pprof"
	"github.com/synnaxlabs/freighter/fhttp"
	"github.com/synnaxlabs/synnax/pkg/ui"
	"github.com/synnaxlabs/x/telem"
	"net/http"
	"time"
)

// SecureHTTPBranch is a Branch that serves HTTP requests behind a TLS multiplexer in
// secure mode.
type SecureHTTPBranch struct {
	// Transports is a list of transports that the Branch will serve.
	Transports []fhttp.BindableTransport
	// ContentTypes is a  list of content types that the Branch will serve.
	ContentTypes []string
	// internal is the underlying fiber.App instance used to serve requests.
	internal *fiber.App
}

var _ Branch = (*SecureHTTPBranch)(nil)

// Routing implements Branch.
func (b *SecureHTTPBranch) Routing() BranchRouting {
	return BranchRouting{
		Policy:   ServeAlwaysPreferSecure,
		Matchers: []cmux.Matcher{cmux.HTTP1Fast()},
	}
}

// Key implements Branch.
func (b *SecureHTTPBranch) Key() string { return "http" }

// Serve implements Branch.
func (b *SecureHTTPBranch) Serve(ctx BranchContext) error {
	b.internal = fiber.New(b.getConfig(ctx))
	b.maybeRouteDebugUtil(ctx)
	b.routeUI()
	b.internal.Use(cors.New(cors.Config{AllowOrigins: "*"}))
	for _, t := range b.Transports {
		t.BindTo(b.internal)
	}
	return filterCloserError(b.internal.Listener(ctx.Lis))
}

// Stop	implements Branch.
func (b *SecureHTTPBranch) Stop() { _ = b.internal.Shutdown() }

func (b *SecureHTTPBranch) maybeRouteDebugUtil(ctx BranchContext) {
	if ctx.Debug {
		b.internal.Get("/metrics", monitor.New(monitor.Config{Title: "Synnax Metrics"}))
		b.internal.Use(pprof.New())
	}
}

func (b *SecureHTTPBranch) routeUI() {
	if ui.HaveUI {
		b.internal.Get("/", filesystem.New(filesystem.Config{
			Root:       http.FS(ui.Dist),
			PathPrefix: "dist",
			Browse:     true,
		}))
	} else {
		b.internal.Get("/", func(c *fiber.Ctx) error {
			return c.SendStream(bytes.NewReader(ui.BareHTML))
		})
	}
}

var baseFiberConfig = fiber.Config{
	DisableStartupMessage: true,
	ReadBufferSize:        int(10 * telem.Kilobyte),
	ReadTimeout:           500 * time.Millisecond,
}

func (b *SecureHTTPBranch) getConfig(ctx BranchContext) fiber.Config {
	baseFiberConfig.AppName = ctx.ServerName
	return baseFiberConfig
}
