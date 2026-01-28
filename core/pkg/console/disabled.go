// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build !console

package console

import (
	_ "embed"

	"github.com/gofiber/fiber/v2"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/fhttp"
)

//go:embed fallback.html
var fallbackHTML []byte

// Service serves the web-based console UI.
type Service struct{}

var _ fhttp.BindableTransport = (*Service)(nil)

// NewService creates a new console UI service.
func NewService() *Service { return &Service{} }

// BindTo binds the console UI service to the provided Fiber app. In the non-ui build,
// it serves a fallback page indicating the Console is not available.
func (s *Service) BindTo(app *fiber.App) {
	app.Get("/", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/html")
		return c.Send(fallbackHTML)
	})
}

// Use implements freighter.Transport.
func (s *Service) Use(...freighter.Middleware) {}

// Report implements alamos.ReportProvider.
func (s *Service) Report() alamos.Report { return alamos.Report{"console": "disabled"} }
