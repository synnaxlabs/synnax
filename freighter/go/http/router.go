// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package http

import (
	"context"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
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

// RouterConfig configures a Router. All fields are optional; pass the zero
// value to accept the defaults.
type RouterConfig struct {
	alamos.Instrumentation
	// StreamWriteDeadline sets the default duration for the write deadline of a stream
	// transport. After the duration has been exceeded, the transport will be closed.
	//
	// [OPTIONAL] - Defaults to 10 seconds.
	StreamWriteDeadline time.Duration
}

var _ config.Config[RouterConfig] = RouterConfig{}

// Validate implements config.Config.
func (RouterConfig) Validate() error { return nil }

// Override implements config.Config.
func (r RouterConfig) Override(other RouterConfig) RouterConfig {
	r.Instrumentation = override.Zero(r.Instrumentation, other.Instrumentation)
	r.StreamWriteDeadline = override.Numeric(
		r.StreamWriteDeadline, other.StreamWriteDeadline,
	)
	return r
}

// NewRouter constructs a Router from the given configs. Servers registered through the
// returned Router will be bound to a fiber.App by calling Router.BindTo. Returns an
// error if the merged config fails to validate.
func NewRouter(configs ...RouterConfig) (*Router, error) {
	cfg, err := config.New(RouterConfig{}, configs...)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.TODO())
	return &Router{
		cfg:           cfg,
		streamCtx:     ctx,
		cancelStreams: cancel,
		streamWg:      &sync.WaitGroup{},
	}, nil
}

// Router collects unary and stream servers and binds them to a fiber.App as a
// single BindableTransport. It also owns the lifecycle of any websocket
// streams: BindTo installs a post-shutdown hook that cancels in-flight streams
// and waits for them to drain.
type Router struct {
	cfg RouterConfig
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
	app.Hooks().OnPostShutdown(func(error) error {
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

// Report implements freighter.Transport. The Router itself is a passive container for
// routes, so its report is empty.
func (*Router) Report() alamos.Report { return alamos.Report{} }

// Use installs the given middleware on every server currently registered with the
// router. Middleware added before BindTo is called is applied to every request handled
// by the resulting fiber.App; middleware added after BindTo applies to subsequent
// requests against existing servers.
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

// NewStreamServer registers a streaming (websocket) server at the given path on the
// router and returns a freighter.StreamServer the caller can attach a handler to via
// BindHandler. The route is bound on the GET method to support the websocket upgrade
// handshake.
func NewStreamServer[RQ, RS freighter.Payload](
	r *Router,
	path string,
	opts ...StreamServerOption,
) freighter.StreamServer[RQ, RS] {
	s := &streamServer[RQ, RS]{
		streamServerOptions: newStreamServerOptions(opts),
		path:                path,
		Instrumentation:     r.cfg.Instrumentation,
		serverCtx:           r.streamCtx,
		writeDeadline:       r.cfg.StreamWriteDeadline,
		wg:                  r.streamWg,
	}
	r.register(path, "GET", s, s.fiberHandler)
	return s
}

// NewUnaryServer registers a unary HTTP server at the given path on the router and
// returns a freighter.UnaryServer the caller can attach a handler to via BindHandler.
// The route is bound on the POST method.
func NewUnaryServer[RQ, RS freighter.Payload](
	r *Router,
	path string,
	opts ...UnaryServerOption,
) freighter.UnaryServer[RQ, RS] {
	us := &unaryServer[RQ, RS]{
		unaryServerOptions: newUnaryServerOptions(opts),
		path:               path,
	}
	r.register(path, "POST", us, us.fiberHandler)
	return us
}
