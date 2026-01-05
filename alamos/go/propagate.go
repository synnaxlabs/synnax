// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package alamos

import "context"

// TraceCarrier is a propagation medium for traces, whether it be a protocol header, persisted
// storage, or something else. TraceCarrier is passed to the Propagate and Depropagate functions.
type TraceCarrier interface {
	// Set sets the given key to the given value.
	Set(key, value string)
	// Get gets the value for the given key.
	Get(key string) string
	// Keys lists the keys stored in this TraceCarrier.
	Keys() []string
}

// Propagate injects the current span into the given carrier, if it exists.
func (t *Tracer) Propagate(ctx context.Context, carrier TraceCarrier) {
	if t != nil {
		t.config.OtelPropagator.Inject(ctx, carrier)
	}
}

// Depropagate extracts a span from the given carrier and returns a new context with the
// span attached.
func (t *Tracer) Depropagate(ctx context.Context, carrier TraceCarrier) context.Context {
	if t == nil {
		return ctx
	}
	return t.config.OtelPropagator.Extract(ctx, carrier)
}
