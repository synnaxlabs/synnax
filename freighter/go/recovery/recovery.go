// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package recovery provides panic-recovery boundaries for freighter request handlers.
// A panic in a handler would otherwise unwind through the transport, where nothing
// recovers it, and take down the entire process. Middleware converts such a panic into
// a generic error returned to the caller, records the panic and stack trace
// server-side, and keeps the node running. LogPanic and ErrPanic are exported for
// transport-specific recovery boundaries (e.g. the gRPC interceptors in the freighter
// grpc package) so panic handling stays consistent across transports.
package recovery

import (
	"runtime/debug"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/errors"
	"go.uber.org/zap"
)

// ErrPanic is returned to the caller when a handler panics. Its message is
// intentionally generic: the panic value and stack trace are recorded in the server
// logs and never sent over the wire, to avoid leaking internal detail to clients.
var ErrPanic = errors.New("the server encountered an unexpected internal error")

// Middleware returns a freighter.Middleware that recovers from panics in any
// downstream middleware or in the handler itself, converting them into ErrPanic so the
// transport can encode and return the error to the caller. Install it as the first
// (outermost) middleware so it also covers panics raised by other middleware.
func Middleware(ins alamos.Instrumentation) freighter.Middleware {
	return freighter.MiddlewareFunc(func(
		ctx freighter.Context,
		next freighter.Next,
	) (oCtx freighter.Context, err error) {
		defer func() {
			if r := recover(); r != nil {
				LogPanic(ins, ctx.Target.String(), r, debug.Stack())
				oCtx = freighter.Context{
					Context:  ctx.Context,
					Protocol: ctx.Protocol,
					Params:   make(freighter.Params),
				}
				err = ErrPanic
			}
		}()
		return next(ctx)
	})
}

// LogPanic records a recovered panic and its stack at ERROR via ins. target identifies
// the handler that panicked (a request target or a gRPC method). It is used by both
// Middleware and transport-specific recovery boundaries so the log format stays
// consistent. It is a no-op if ins has no logger.
func LogPanic(ins alamos.Instrumentation, target string, recovered any, stack []byte) {
	if ins.L == nil {
		return
	}
	ins.L.Error(
		"recovered from panic in handler",
		zap.String("target", target),
		zap.Any("panic", recovered),
		zap.ByteString("stack", stack),
	)
}
