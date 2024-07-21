// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package fhttp

import (
	"context"
	"github.com/gofiber/fiber/v2"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/httputil"
	"github.com/synnaxlabs/x/override"
	"time"
)

type route struct {
	path       string
	handler    fiber.Handler
	transport  freighter.Transport
	httpMethod string
}

type RouterConfig struct {
	alamos.Instrumentation
	// StreamWriteDeadline sets the default duration for the write deadline of a stream
	// transport. After the duration has been exceeded, the transport will be closed.
	StreamWriteDeadline time.Duration
}

var _ config.Config[RouterConfig] = RouterConfig{}

// Validate implements config.Properties.
func (r RouterConfig) Validate() error { return nil }

// Override implements config.Properties.
func (r RouterConfig) Override(other RouterConfig) RouterConfig {
	r.Instrumentation = override.Zero(r.Instrumentation, other.Instrumentation)
	r.StreamWriteDeadline = override.Numeric(r.StreamWriteDeadline, other.StreamWriteDeadline)
	return r
}

func NewRouter(configs ...RouterConfig) *Router {
	cfg, err := config.New(RouterConfig{}, configs...)
	if err != nil {
		panic(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Router{
		RouterConfig: cfg,
		ctx:          ctx,
		cancel:       cancel,
	}
}

type Router struct {
	RouterConfig
	ctx    context.Context
	cancel context.CancelFunc
	routes []route
}

var _ BindableTransport = (*Router)(nil)

func (r *Router) BindTo(app *fiber.App) {
	app.Hooks().OnShutdown(func() error {
		r.cancel()
		return nil
	})
	for _, route := range r.routes {
		if route.httpMethod == "GET" {
			app.Get(route.path, route.handler)
		} else {
			app.Post(route.path, route.handler)
		}
	}
}

func (r *Router) Report() alamos.Report {
	return alamos.Report{}
}

func (r *Router) Use(middleware ...freighter.Middleware) {
	for _, route := range r.routes {
		route.transport.Use(middleware...)
	}
}

func (r *Router) register(
	path string,
	httpMethod string,
	t freighter.Transport,
	h fiber.Handler,
) {
	r.routes = append(r.routes, route{
		httpMethod: httpMethod,
		path:       path,
		handler:    h,
		transport:  t,
	})
}

func StreamServer[RQ, RS freighter.Payload](r *Router, internal bool, path string) freighter.StreamServer[RQ, RS] {
	s := &streamServer[RQ, RS]{
		internal:        internal,
		Reporter:        streamReporter,
		path:            path,
		Instrumentation: r.Instrumentation,
		serverCtx:       r.ctx,
		writeDeadline:   r.StreamWriteDeadline,
	}
	r.register(path, "GET", s, s.fiberHandler)
	return s
}

func UnaryServer[RQ, RS freighter.Payload](r *Router, internal bool, path string) freighter.UnaryServer[RQ, RS] {
	us := &unaryServer[RQ, RS]{
		internal: internal,
		Reporter: unaryReporter,
		path:     path,
		requestParser: func(c *fiber.Ctx, ecd httputil.EncoderDecoder) (req RQ, _ error) {
			return req, c.BodyParser(&req)
		},
	}
	r.register(path, "POST", us, us.fiberHandler)
	return us
}
