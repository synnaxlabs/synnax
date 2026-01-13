// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package constraint

import (
	"context"

	"github.com/synnaxlabs/x/telem"
)

// ComputeFunc is a function that computes a derived value from a source value. It
// returns the computed value and a boolean indicating success.
type ComputeFunc func(sourceValue any) (any, bool)

// computeRegistry maps property names to their computation functions.
var computeRegistry = make(map[string]ComputeFunc)

func init() {
	// Register built-in computation functions
	RegisterComputation("duration", func(v any) (any, bool) {
		tr, ok := v.(telem.TimeRange)
		if !ok {
			return nil, false
		}
		return tr.Span(), true
	})
	RegisterComputation("age", func(v any) (any, bool) {
		ts, ok := v.(telem.TimeStamp)
		if !ok {
			return nil, false
		}
		return ts.Span(telem.Now()), true
	})
	RegisterComputation("count", func(v any) (any, bool) {
		switch val := v.(type) {
		case []any:
			return len(val), true
		default:
			return nil, false
		}
	})
}

// RegisterComputation registers a computation function for a property name.
// This allows custom computed properties to be defined.
func RegisterComputation(property string, fn ComputeFunc) {
	computeRegistry[property] = fn
}

func computeValue(
	ctx context.Context,
	params EnforceParams,
	property string,
	source []string,
) (any, bool) {
	if len(source) < 1 {
		return nil, false
	}
	target := source[0]
	field := source[1:]
	sourceValue, ok := resolveFieldValue(ctx, params, target, field)
	if !ok {
		return nil, false
	}
	fn, ok := computeRegistry[property]
	if !ok {
		return nil, false
	}
	return fn(sourceValue)
}
