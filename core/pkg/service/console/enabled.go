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
	"go.uber.org/zap"
)

//go:embed dist/*
var embeddedAssets embed.FS

const rootHTMLFile = "index.html"

// Service serves the web-based console UI.
type Service struct{ fs fs.FS }

var _ fhttp.BindableTransport = (*Service)(nil)

// OpenService creates a new console UI service with embedded assets.
func OpenService() *Service {
	subFS, err := fs.Sub(embeddedAssets, "dist")
	if err != nil {
		zap.S().DPanic("Failed to load embedded assets", zap.Error(err))
	}
	return &Service{fs: subFS}
}

// BindTo binds the console UI service to the provided Fiber app. In the ui build, it
// serves the embedded console assets.
func (s *Service) BindTo(app *fiber.App) {
	app.Use("/", filesystem.New(filesystem.Config{
		Root:         http.FS(s.fs),
		Browse:       false,
		Index:        rootHTMLFile,
		MaxAge:       86400,        // 1 day cache for static assets
		NotFoundFile: rootHTMLFile, // Serve index.html for SPA routing
	}))
}

func (s *Service) Use(...freighter.Middleware) {}

// Report implements alamos.ReportProvider.
func (s *Service) Report() alamos.Report { return alamos.Report{"console": "enabled"} }
