// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package whip binds the WHIP endpoints (RFC 9725) of WHIP read tasks: an encoder
// publishes to a task with POST /api/v1/whip/{task} and ends its session with DELETE
// on the session URL. The encoder sends an API key as its bearer token. RFC 9725 fixes
// the methods, status codes, and headers, so the endpoints are raw Fiber routes. Each
// one serves a Freighter server, which the API layer binds like any other.
package whip

import (
	"mime"
	"strings"
	"uuid"

	"github.com/gofiber/fiber/v3"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	fhttp "github.com/synnaxlabs/freighter/http"
	apiwhip "github.com/synnaxlabs/synnax/pkg/api/whip"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/license"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

const (
	basePath   = "/api/v1/whip/"
	sdpType    = "application/sdp"
	bearerAuth = "Bearer "
	protocol   = "http"
)

// Transport binds the WHIP endpoints to a Fiber app.
type Transport struct {
	publish   server[apiwhip.PublishRequest, apiwhip.PublishResponse]
	unpublish server[apiwhip.UnpublishRequest, struct{}]
}

var _ fhttp.BindableTransport = (*Transport)(nil)

// New returns a Transport whose servers have no handlers bound.
func New() *Transport { return &Transport{} }

// Publish returns the server that starts a session.
func (t *Transport) Publish() freighter.UnaryServer[
	apiwhip.PublishRequest,
	apiwhip.PublishResponse,
] {
	return &t.publish
}

// Unpublish returns the server that ends a session.
func (t *Transport) Unpublish() freighter.UnaryServer[
	apiwhip.UnpublishRequest,
	struct{},
] {
	return &t.unpublish
}

// BindTo implements fhttp.BindableTransport.
func (t *Transport) BindTo(app *fiber.App) {
	app.Post(basePath+":task", t.handlePublish)
	app.Patch(basePath+":task/:session", t.handlePatch)
	app.Delete(basePath+":task/:session", t.handleUnpublish)
}

// Use implements freighter.Transport. It adds the middleware to both servers.
func (t *Transport) Use(middleware ...freighter.Middleware) {
	t.publish.Use(middleware...)
	t.unpublish.Use(middleware...)
}

// Report implements alamos.ReportProvider.
func (*Transport) Report() alamos.Report { return report() }

func report() alamos.Report {
	return freighter.Reporter{Protocol: "whip", Encodings: []string{sdpType}}.Report()
}

func (t *Transport) handlePublish(c fiber.Ctx) error {
	task, err := parseKey(c, "task")
	if err != nil {
		return fail(c, err)
	}
	key, err := bearer(c)
	if err != nil {
		return fail(c, err)
	}
	if mediaType, _, _ := mime.ParseMediaType(
		c.Get(fiber.HeaderContentType),
	); mediaType != sdpType {
		return c.Status(fiber.StatusUnsupportedMediaType).
			SendString("the body must be an SDP offer with type " + sdpType)
	}
	res, err := t.publish.exec(c, apiwhip.PublishRequest{
		Task:  task,
		Key:   key,
		Offer: string(c.Body()),
	})
	if err != nil {
		return fail(c, err)
	}
	c.Location(basePath + task.String() + "/" + res.Session.String())
	c.Set(fiber.HeaderContentType, sdpType)
	return c.Status(fiber.StatusCreated).SendString(res.Answer)
}

// handlePatch answers that the endpoint takes no trickle ICE or ICE restarts. The
// answer to a publish holds every candidate.
func (*Transport) handlePatch(c fiber.Ctx) error {
	return c.SendStatus(fiber.StatusMethodNotAllowed)
}

func (t *Transport) handleUnpublish(c fiber.Ctx) error {
	task, err := parseKey(c, "task")
	if err != nil {
		return fail(c, err)
	}
	session, err := parseKey(c, "session")
	if err != nil {
		return fail(c, err)
	}
	key, err := bearer(c)
	if err != nil {
		return fail(c, err)
	}
	if _, err = t.unpublish.exec(c, apiwhip.UnpublishRequest{
		Task:    task,
		Session: session,
		Key:     key,
	}); err != nil {
		return fail(c, err)
	}
	return c.SendStatus(fiber.StatusOK)
}

func bearer(c fiber.Ctx) (string, error) {
	key, ok := strings.CutPrefix(c.Get(fiber.HeaderAuthorization), bearerAuth)
	if !ok {
		return "", errors.Wrap(
			auth.ErrInvalidCredentials,
			"the request needs an API key as its bearer token",
		)
	}
	return key, nil
}

func parseKey(c fiber.Ctx, param string) (uuid.UUID, error) {
	key, err := uuid.Parse(c.Params(param))
	if err != nil {
		return uuid.Nil(), errors.Wrapf(
			query.ErrNotFound,
			"%s %q does not exist",
			param,
			c.Params(param),
		)
	}
	return key, nil
}

// fail answers a request with the status code that matches err.
func fail(c fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	switch {
	case errors.Is(err, auth.ErrAccessDenied), errors.Is(err, license.ErrLicense):
		code = fiber.StatusForbidden
	case errors.Is(err, auth.ErrAuth):
		code = fiber.StatusUnauthorized
	case errors.Is(err, query.ErrNotFound):
		code = fiber.StatusNotFound
	case errors.Is(err, validate.ErrValidation):
		code = fiber.StatusBadRequest
	}
	return c.Status(code).SendString(err.Error())
}

// server is a freighter.UnaryServer whose requests come from a WHIP route.
type server[RQ, RS freighter.Payload] struct {
	freighter.MiddlewareCollector
	handle freighter.UnaryHandler[RQ, RS]
}

// BindHandler implements freighter.UnaryServer.
func (s *server[RQ, RS]) BindHandler(handle freighter.UnaryHandler[RQ, RS]) {
	s.handle = handle
}

// Report implements alamos.ReportProvider.
func (*server[RQ, RS]) Report() alamos.Report { return report() }

// exec runs req through the middleware of the server, then its handler.
func (s *server[RQ, RS]) exec(c fiber.Ctx, req RQ) (RS, error) {
	var res RS
	_, err := s.Exec(
		freighter.Context{
			Context:  c.Context(),
			Params:   make(freighter.Params),
			Protocol: protocol,
			Target:   address.Address(c.Path()),
			Remote:   address.Address(c.RequestCtx().RemoteAddr().String()),
			Role:     freighter.RoleServer,
			Variant:  freighter.VariantUnary,
		},
		freighter.FinalizerFunc(func(ctx freighter.Context) (freighter.Context, error) {
			var err error
			res, err = s.handle(ctx, req)
			return freighter.Context{
				Context:  ctx,
				Protocol: protocol,
				Params:   make(freighter.Params),
			}, err
		}),
	)
	return res, err
}
