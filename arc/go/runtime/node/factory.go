// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package node

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/module"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

// Config provides dependencies and context for creating node instances.
type Config struct {
	alamos.Instrumentation
	// Node is the IR definition for this node.
	Node ir.Node
	// State provides access to input/output data and channel I/O.
	State *state.Node
	// Module contains the arc module for accessing global state and functions.
	Module module.Module
}

// Factory creates node instances from IR definitions.
// Implementations check the node type and return query.ErrNotFound if they
// cannot handle the given type, allowing MultiFactory to try other factories.
type Factory interface {
	// Create constructs a node from the given configuration.
	// Returns query.ErrNotFound if this factory cannot handle cfg.Node.Type.
	Create(ctx context.Context, cfg Config) (Node, error)
}

// MultiFactory composes multiple factories with fallback behavior.
// It tries each factory in order until one succeeds or all return NotFound.
type MultiFactory []Factory

// Create attempts to create a node using each factory in sequence.
// Returns the first successful node, or query.ErrNotFound if no factory matches.
// Non-NotFound errors stop the search and are returned immediately.
func (mf MultiFactory) Create(ctx context.Context, cfg Config) (Node, error) {
	for _, f := range mf {
		n, err := f.Create(ctx, cfg)
		if err != nil {
			if errors.Is(err, query.ErrNotFound) {
				continue
			}
			return nil, err
		}
		return n, nil
	}
	return nil, query.ErrNotFound
}
