// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package server

import (
	"time"

	"github.com/cockroachdb/cmux"
	"github.com/gofiber/contrib/v3/monitor"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/pprof"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter/http"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

// SecureHTTPBranch is a Branch that serves HTTP requests behind a TLS multiplexer in
// secure mode.
type SecureHTTPBranch struct {
	// internal is the underlying fiber.App instance used to serve requests.
	internal *fiber.App
	ins      alamos.Instrumentation
	// Transports is a list of transports that the Branch will serve.
	Transports []http.BindableTransport
	// MaxIdleWorkerDuration sets fasthttp's idle worker reap interval, which also
	// bounds how long the server's worker-pool janitor goroutine can outlive Stop. Zero
	// keeps the fasthttp default (10s). Tests set this low so goroutine leak checks
	// don't flag the janitor.
	MaxIdleWorkerDuration time.Duration
}

var _ Branch = (*SecureHTTPBranch)(nil)

// Routing implements Branch.
func (*SecureHTTPBranch) Routing() BranchRouting {
	return BranchRouting{
		Policy:   RoutingPolicyServeAlwaysPreferSecure,
		Matchers: []cmux.Matcher{cmux.HTTP1Fast()},
	}
}

// Key implements Branch.
func (*SecureHTTPBranch) Key() string { return "http" }

// Init implements Branch.
func (b *SecureHTTPBranch) Init(ctx BranchContext) {
	b.ins = ctx.Instrumentation
	b.internal = fiber.New(b.getConfig(ctx))
	if b.MaxIdleWorkerDuration > 0 {
		b.internal.Server().MaxIdleWorkerDuration = b.MaxIdleWorkerDuration
	}
	b.maybeRouteDebugUtil(ctx)
	b.internal.Use(cors.New(cors.Config{AllowOrigins: []string{"*"}}))
	for _, t := range b.Transports {
		t.BindTo(b.internal)
	}
}

// Serve implements Branch.
func (b *SecureHTTPBranch) Serve(ctx BranchContext) error {
	return b.internal.Listener(ctx.Lis, fiber.ListenConfig{DisableStartupMessage: true})
}

// Stop implements Branch.
func (b *SecureHTTPBranch) Stop() {
	if b.internal == nil {
		return
	}
	if err := b.internal.Shutdown(); err != nil {
		b.ins.L.Error("failed to shut down http server", zap.Error(err))
	}
}

func (b *SecureHTTPBranch) maybeRouteDebugUtil(ctx BranchContext) {
	if !ctx.Debug {
		return
	}
	b.internal.Get("/metrics", monitor.New(monitor.Config{Title: "Synnax Metrics"}))
	b.internal.Use(pprof.New())
}

func (*SecureHTTPBranch) getConfig(ctx BranchContext) fiber.Config {
	return fiber.Config{
		AppName:         ctx.ServerName,
		ReadBufferSize:  int(100 * telem.Kilobyte),
		WriteBufferSize: int(64 * telem.Kilobyte),
		ReadTimeout:     5 * time.Second,
		// Defaults to ReadTimeout. A client that reuses a keep-alive connection as the
		// Core closes it for idleness loses the request, so this must outlast the 30s
		// connectivity poll that keeps a client's connection warm.
		IdleTimeout: 2 * time.Minute,
	}
}
