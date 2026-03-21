// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package backup

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	fhttp "github.com/synnaxlabs/freighter/http"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/auth/token"
	svcbackup "github.com/synnaxlabs/synnax/pkg/service/backup"
	"github.com/synnaxlabs/synnax/pkg/service/user"
)

// Transport implements fhttp.BindableTransport to serve .sy backup files
// via a raw Fiber handler, bypassing Freighter's JSON serialization.
type Transport struct {
	internal *svcbackup.Service
	token    *token.Service
	access   *rbac.Service
}

var _ fhttp.BindableTransport = (*Transport)(nil)

// NewTransport creates a new backup transport.
func NewTransport(
	svc *svcbackup.Service,
	tokenSvc *token.Service,
	accessSvc *rbac.Service,
) *Transport {
	return &Transport{internal: svc, token: tokenSvc, access: accessSvc}
}

// BindTo registers the backup endpoints on the Fiber app.
func (t *Transport) BindTo(app *fiber.App) {
	app.Post("/api/v1/export", t.handleExport)
}

func (t *Transport) handleExport(c fiber.Ctx) error {
	// Authenticate — same logic as auth.TokenMiddleware but manual since we're
	// outside Freighter's middleware chain.
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).
			JSON(fiber.Map{"error": "missing authorization header"})
	}
	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	userKey, err := t.token.Validate(tokenStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).
			JSON(fiber.Map{"error": "invalid token"})
	}

	var httpReq struct {
		svcbackup.ExportRequest
		Path string `json:"path"`
	}
	if err := json.Unmarshal(c.Body(), &httpReq); err != nil {
		return c.Status(fiber.StatusBadRequest).
			JSON(fiber.Map{"error": "invalid request body"})
	}

	// Enforce RBAC — same pattern as every other API service handler.
	if err := t.access.Enforce(c.Context(), access.Request{
		Subject: user.OntologyID(userKey),
		Action:  access.ActionRetrieve,
		Objects: httpReq.ExportRequest.OntologyIDs(),
	}); err != nil {
		return c.Status(fiber.StatusForbidden).
			JSON(fiber.Map{"error": err.Error()})
	}

	// Local mode: write to a file on the server's filesystem.
	if httpReq.Path != "" {
		f, err := os.Create(httpReq.Path)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).
				JSON(fiber.Map{"error": "failed to create file: " + err.Error()})
		}
		defer f.Close()
		if err := t.internal.Export(c.Context(), httpReq.ExportRequest, f); err != nil {
			os.Remove(httpReq.Path)
			return c.Status(fiber.StatusInternalServerError).
				JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"path": httpReq.Path})
	}

	// Network mode: stream the ZIP as the HTTP response.
	c.Set("Content-Type", "application/zip")
	c.Set("Content-Disposition", "attachment; filename=\"backup.sy\"")
	return t.internal.Export(c.Context(), httpReq.ExportRequest, c.Response().BodyWriter())
}

// Use implements freighter.Transport.
func (*Transport) Use(...freighter.Middleware) {}

// Report implements alamos.ReportProvider.
func (*Transport) Report() alamos.Report {
	return alamos.Report{"backup": "enabled"}
}
