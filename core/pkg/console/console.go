// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package console

import (
	"embed"
	"io/fs"
	"path"
	"slices"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	fhttp "github.com/synnaxlabs/freighter/http"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
)

//go:embed fallback.html
var fallbackFS embed.FS

// Config is the configuration for creating a Console.
type Config struct {
	// FS is the filesystem containing the Console assets. When nil, the Console serves
	// a fallback page indicating that it is not available.
	FS fs.FS
}

var _ config.Config[Config] = Config{}

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.FS = override.Nil(c.FS, other.FS)
	return c
}

// Validate implements config.Config.
func (c Config) Validate() error { return nil }

// Console serves the web-based console UI. When an FS is provided, it serves the
// embedded Console assets. Otherwise, it serves a fallback page indicating that the
// Console is not available.
type Console struct{ fs fs.FS }

var _ fhttp.BindableTransport = (*Console)(nil)

// New creates a new Console. If no FS is provided in the config, it uses the default FS
// from the build. When built with -tags=console, the default FS contains the embedded
// Console assets. Otherwise, it is nil and the Console serves a fallback page.
func New(cfgs ...Config) (*Console, error) {
	cfg, err := config.New(Config{FS: defaultFS}, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Console{fs: cfg.FS}, nil
}

// Cache policies for the two kinds of file the Console build emits. A hashed asset name
// changes with its content, so a browser may keep it forever. Every other name survives
// a release, and embedded files all carry a zero modification time, so a revalidation
// of a stored copy answers 304 even after an upgrade.
const (
	immutableCacheControl = "public, max-age=31536000, immutable"
	noStoreCacheControl   = "no-store"
)

// Directory Vite emits content-hashed files into, at the bundle root and under pluto/.
const hashedAssetDir = "assets"

// BindTo binds the console UI service to the provided Fiber app.
func (c *Console) BindTo(app *fiber.App) {
	sendFile := fiber.SendFile{FS: c.fs, Compress: true, CacheDuration: -1}
	if !c.enabled() {
		app.Get("/", func(ctx fiber.Ctx) error {
			if err := ctx.SendFile("fallback.html", sendFile); err != nil {
				return err
			}
			ctx.Set(fiber.HeaderCacheControl, noStoreCacheControl)
			return nil
		})
		return
	}
	app.Use("/", static.New("", static.Config{
		FS:            c.fs,
		Compress:      true,
		CacheDuration: -1,
		ModifyResponse: func(ctx fiber.Ctx) error {
			ctx.Set(fiber.HeaderCacheControl, cacheControl(ctx.Path()))
			return nil
		},
	}))
	app.Get("/*", func(ctx fiber.Ctx) error {
		// A file the build no longer emits must answer 404. Handing it the app shell
		// fails the caller in whatever way its content type happens to break: a module
		// worker given HTML reports an error event with no message at all.
		if path.Ext(ctx.Path()) != "" {
			return fiber.ErrNotFound
		}
		if err := ctx.SendFile("index.html", sendFile); err != nil {
			return err
		}
		ctx.Set(fiber.HeaderCacheControl, noStoreCacheControl)
		return nil
	})
}

// cacheControl returns the cache policy for the file served at the given request path.
func cacheControl(reqPath string) string {
	if slices.Contains(strings.Split(reqPath, "/"), hashedAssetDir) {
		return immutableCacheControl
	}
	return noStoreCacheControl
}

// Use implements freighter.Transport.
func (*Console) Use(...freighter.Middleware) {}

// Report implements alamos.ReportProvider.
func (c *Console) Report() alamos.Report {
	value := "disabled"
	if c.enabled() {
		value = "enabled"
	}
	return alamos.Report{"console": value}
}

func (c *Console) enabled() bool { return c.fs != fallbackFS }
