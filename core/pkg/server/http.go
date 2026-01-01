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
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/gofiber/fiber/v2/middleware/pprof"
	"github.com/synnaxlabs/freighter/fhttp"
	"github.com/synnaxlabs/x/telem"
)

// SecureHTTPBranch is a Branch that serves HTTP requests behind a TLS multiplexer in
// secure mode.
type SecureHTTPBranch struct {
	// Transports is a list of transports that the Branch will serve.
	Transports []fhttp.BindableTransport
	// ContentTypes is a  list of content types that the Branch will serve.
	// internal is the underlying fiber.App instance used to serve requests.
	internal *fiber.App
}

var _ Branch = (*SecureHTTPBranch)(nil)

// Routing implements Branch.
func (b *SecureHTTPBranch) Routing() BranchRouting {
	return BranchRouting{
		Policy:   ServeAlwaysPreferSecure,
		Matchers: []cmux.Matcher{cmux.HTTP1Fast()},
	}
}

// Key implements Branch.
func (b *SecureHTTPBranch) Key() string { return "http" }

// Serve implements Branch.
func (b *SecureHTTPBranch) Serve(ctx BranchContext) error {
	b.internal = fiber.New(b.getConfig(ctx))
	b.maybeRouteDebugUtil(ctx)
	b.internal.Use(cors.New(cors.Config{AllowOrigins: "*"}))
	for _, t := range b.Transports {
		t.BindTo(b.internal)
	}
	return b.internal.Listener(ctx.Lis)
}

// Stop	implements Branch. Stop is safe to call even if Serve has not been called.
func (b *SecureHTTPBranch) Stop() {
	if b.internal != nil {
		_ = b.internal.Shutdown()
	}
}

func (b *SecureHTTPBranch) maybeRouteDebugUtil(ctx BranchContext) {
	if !ctx.Debug {
		return
	}
	b.internal.Get("/metrics", monitor.New(monitor.Config{Title: "Synnax Metrics"}))
	b.internal.Use(pprof.New())
}

var baseFiberConfig = fiber.Config{
	DisableStartupMessage: true,
	ReadBufferSize:        int(100 * telem.Kilobyte),
	ReadTimeout:           5 * time.Second,
}

func (b *SecureHTTPBranch) getConfig(ctx BranchContext) fiber.Config {
	baseFiberConfig.AppName = ctx.ServerName
	return baseFiberConfig
}
