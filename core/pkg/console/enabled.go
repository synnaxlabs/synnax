// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build console

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
	"go.uber.org/zap"
)

//go:embed all:dist
var embeddedAssets embed.FS

const rootHTMLFile = "index.html"

// Service serves the web-based console UI.
type Service struct{ fs fs.FS }

var _ fhttp.BindableTransport = (*Service)(nil)

// NewService creates a new console UI service with embedded assets.
func NewService() *Service {
	subFS, err := fs.Sub(embeddedAssets, "dist")
	if err != nil {
		zap.S().DPanic("Failed to load embedded assets", zap.Error(err))
	}
	return &Service{fs: subFS}
}

// BindTo binds the console UI service to the provided Fiber app. In the ui build, it
// serves the embedded console assets.
func (s *Service) BindTo(app *fiber.App) {
	app.Use("/", static.New("", static.Config{
		FS:         s.fs,
		Browse:     false,
		IndexNames: []string{rootHTMLFile},
		MaxAge:     int((24 * time.Hour).Seconds()), // 1 day cache for static assets
		NotFoundHandler: func(c fiber.Ctx) error {
			return c.SendFile(rootHTMLFile, fiber.SendFile{FS: s.fs})
		}, // Serve index.html for SPA routing
	}))
}

func (*Service) Use(...freighter.Middleware) {}

// Report implements alamos.ReportProvider.
func (*Service) Report() alamos.Report { return alamos.Report{"console": "enabled"} }
