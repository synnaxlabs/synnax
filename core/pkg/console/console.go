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
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	fhttp "github.com/synnaxlabs/freighter/http"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

//go:embed fallback.html
var fallbackFS embed.FS

const rootHTMLFile = "index.html"

// Config is the configuration for creating a Console.
type Config struct {
	// Enabled controls whether the service serves the embedded console UI or a
	// fallback page. Defaults to defaultEnabled, which is set by the build tag.
	Enabled *bool
}

var _ config.Config[Config] = Config{}

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Enabled = override.Nil(c.Enabled, other.Enabled)
	return c
}

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("console")
	validate.NotNil(v, "enabled", c.Enabled)
	return v.Error()
}

// Console serves the web-based console UI. When enabled, it serves the embedded console
// assets built from the console package. When disabled, it serves a fallback page
// indicating that the console is not available.
type Console struct {
	fs      fs.FS
	enabled bool
}

var _ fhttp.BindableTransport = (*Console)(nil)

// New creates a new Console with the given configurations.
func New(cfgs ...Config) (*Console, error) {
	cfg, err := config.New(Config{Enabled: new(defaultEnabled)}, cfgs...)
	if err != nil {
		return nil, err
	}
	c := &Console{enabled: *cfg.Enabled}
	if !c.enabled {
		return c, nil
	}
	if c.fs, err = fs.Sub(embeddedAssets, "dist"); err != nil {
		return nil, err
	}
	return c, nil
}

// BindTo binds the console UI service to the provided Fiber app.
func (c *Console) BindTo(app *fiber.App) {
	if !c.enabled {
		app.Get("/", func(ctx fiber.Ctx) error {
			return ctx.SendFile("fallback.html", fiber.SendFile{FS: fallbackFS})
		})
		return
	}
	app.Use("/", static.New("", static.Config{
		FS:     c.fs,
		Browse: false,
		MaxAge: int((24 * time.Hour).Seconds()),
	}))
	app.Get("/*", func(ctx fiber.Ctx) error {
		return ctx.SendFile(rootHTMLFile, fiber.SendFile{FS: c.fs})
	})
}

// Use implements freighter.Transport.
func (*Console) Use(...freighter.Middleware) {}

// Report implements alamos.ReportProvider.
func (c *Console) Report() alamos.Report {
	value := "disabled"
	if c.enabled {
		value = "enabled"
	}
	return alamos.Report{"console": value}
}
