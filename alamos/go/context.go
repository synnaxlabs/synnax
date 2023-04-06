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

// attach attaches the given Instrumentation to the given context.
func attach(ctx context.Context, ins Instrumentation) context.Context {
	return context.WithValue(ctx, contextKey, ins)
}

const contextKey = "alamos-instrumentation"

// extract extracts the Instrumentation from the given context.
// If the Instrumentation is not present, ok will be false.
func extract(ctx context.Context) Instrumentation {
	v := ctx.Value(contextKey)
	if v == nil {
		return Instrumentation{}
	}
	return v.(Instrumentation)
}
