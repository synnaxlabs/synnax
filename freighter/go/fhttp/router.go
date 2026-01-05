// Copyright 2026 Synnax Labs, Inc.
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
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
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

// Validate implements config.Config.
func (r RouterConfig) Validate() error { return nil }

// Override implements config.Config.
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
		RouterConfig:  cfg,
		streamCtx:     ctx,
		cancelStreams: cancel,
		streamWg:      &sync.WaitGroup{},
	}
}

type Router struct {
	RouterConfig
	// fiber doesn't manage the lifecycle of websocket connections (streams), so we need
	// to manage them ourselves. We'll pass in a context object that gets cancelled when
	// the app is shut down, and we'll use a WaitGroup to wait for all streams to close.
	// streamCtx is the context object that gets cancelled when the app is shut down.
	streamCtx context.Context
	// cancelStreams cancels the streamCtx.
	cancelStreams context.CancelFunc
	// streamWg is a WaitGroup that waits for all streams to close.
	streamWg *sync.WaitGroup
	// routes is a list of all routes that have been registered with the router.
	routes []route
}

var _ BindableTransport = (*Router)(nil)

// BindTo binds the router and all of its routes to the given fiber app.
func (r *Router) BindTo(app *fiber.App) {
	app.Hooks().OnShutdown(func() error {
		// Cancel all streams and wait for them to close.
		r.cancelStreams()
		r.streamWg.Wait()
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

func StreamServer[RQ, RS freighter.Payload](
	r *Router,
	path string,
	opts ...ServerOption,
) freighter.StreamServer[RQ, RS] {
	s := &streamServer[RQ, RS]{
		serverOptions:   newServerOptions(opts),
		Reporter:        streamReporter,
		path:            path,
		Instrumentation: r.Instrumentation,
		serverCtx:       r.streamCtx,
		writeDeadline:   r.StreamWriteDeadline,
		wg:              r.streamWg,
	}
	r.register(path, "GET", s, s.fiberHandler)
	return s
}

func UnaryServer[RQ, RS freighter.Payload](r *Router, path string, opts ...ServerOption) freighter.UnaryServer[RQ, RS] {
	us := &unaryServer[RQ, RS]{
		serverOptions: newServerOptions(opts),
		Reporter:      unaryReporter,
		path:          path,
	}
	r.register(path, "POST", us, us.fiberHandler)
	return us
}
