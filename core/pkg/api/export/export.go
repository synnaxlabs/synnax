// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package export

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	fhttp "github.com/synnaxlabs/freighter/http"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/auth/token"
	svcexport "github.com/synnaxlabs/synnax/pkg/service/export"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/synnax/pkg/service/workspace"
)

// Transport implements fhttp.BindableTransport to serve .syc export files
// via a raw Fiber handler, bypassing Freighter's JSON serialization.
type Transport struct {
	internal *svcexport.Service
	token    *token.Service
	access   *rbac.Service
}

var _ fhttp.BindableTransport = (*Transport)(nil)

// NewTransport creates a new export transport.
func NewTransport(
	svc *svcexport.Service,
	tokenSvc *token.Service,
	accessSvc *rbac.Service,
) *Transport {
	return &Transport{internal: svc, token: tokenSvc, access: accessSvc}
}

// BindTo registers the export endpoint on the Fiber app.
func (t *Transport) BindTo(app *fiber.App) {
	app.Post("/api/v1/export", t.handle)
}

func (t *Transport) handle(c fiber.Ctx) error {
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

	var req svcexport.Request
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return c.Status(fiber.StatusBadRequest).
			JSON(fiber.Map{"error": "invalid request body"})
	}

	// Enforce RBAC — same pattern as every other API service handler.
	subject := user.OntologyID(userKey)
	objects := ontologyIDsFromRequest(req)
	if err := t.access.Enforce(c.Context(), access.Request{
		Subject: subject,
		Action:  access.ActionRetrieve,
		Objects: objects,
	}); err != nil {
		return c.Status(fiber.StatusForbidden).
			JSON(fiber.Map{"error": err.Error()})
	}

	// Local mode: write to a file on the server's filesystem.
	if req.Path != "" {
		f, err := os.Create(req.Path)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).
				JSON(fiber.Map{"error": "failed to create file: " + err.Error()})
		}
		defer f.Close()
		if err := t.internal.Export(c.Context(), req, f); err != nil {
			os.Remove(req.Path)
			return c.Status(fiber.StatusInternalServerError).
				JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"path": req.Path})
	}

	// Network mode: stream the ZIP as the HTTP response.
	c.Set("Content-Type", "application/zip")
	c.Set("Content-Disposition", "attachment; filename=\"export.syc\"")
	return t.internal.Export(c.Context(), req, c.Response().BodyWriter())
}

func ontologyIDsFromRequest(req svcexport.Request) []ontology.ID {
	ids := make([]ontology.ID, 0, len(req.WorkspaceKeys))
	for _, key := range req.WorkspaceKeys {
		ids = append(ids, workspace.OntologyID(key))
	}
	return ids
}

// Use implements freighter.Transport.
func (*Transport) Use(...freighter.Middleware) {}

// Report implements alamos.ReportProvider.
func (*Transport) Report() alamos.Report {
	return alamos.Report{"export": "enabled"}
}
