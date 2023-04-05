// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package alamos

import (
	"context"
)

// Sub returns new Instrumentation with the given key and options and a context
// with the Instrumentation attached.
func Sub(ctx context.Context, key string) context.Context {
	ins, ok := Extract(ctx)
	if !ok {
		return ctx
	}
	return Attach(ctx, ins.Sub(key))
}

// Attach attaches the given Instrumentation to the given context.
func Attach(ctx context.Context, ins *Instrumentation) context.Context {
	return context.WithValue(ctx, contextKey, ins)
}

const contextKey = "alamos-instrumentation"

// Extract extracts the Instrumentation from the given context.
// If the Instrumentation is not present, ok will be false.
func Extract(ctx context.Context) (*Instrumentation, bool) {
	v := ctx.Value(contextKey)
	if v == nil {
		return nil, false
	}
	return v.(*Instrumentation), true
}
