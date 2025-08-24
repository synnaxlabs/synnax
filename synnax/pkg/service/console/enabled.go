// Copyright 2025 Synnax Labs, Inc.
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
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/fhttp"
)

//go:embed dist/*
var embeddedAssets embed.FS

// Service serves the web-based console UI.
type Service struct {
	fs fs.FS
}

var _ fhttp.BindableTransport = (*Service)(nil)

// NewService creates a new console UI service with embedded assets.
func NewService() *Service {
	// Strip the "assets" prefix from the embedded filesystem
	subFS, _ := fs.Sub(embeddedAssets, "dist")
	return &Service{
		fs: subFS,
	}
}

// BindTo binds the console UI service to the provided Fiber app.
// In the ui build, it serves the embedded console assets.
func (s *Service) BindTo(app *fiber.App) {
	// Serve static files with proper caching and content types
	app.Use("/", filesystem.New(filesystem.Config{
		Root:         http.FS(s.fs),
		Browse:       false,
		Index:        "index.html",
		MaxAge:       86400,        // 1 day cache for static assets
		NotFoundFile: "index.html", // Serve index.html for SPA routing
	}))
}

// Use implements freighter.Transport for compatibility.
func (s *Service) Use(middleware ...freighter.Middleware) {}

// Report implements alamos.ReportProvider for compatibility.
func (s *Service) Report() alamos.Report {
	return alamos.Report{"ui": "enabled"}
}
