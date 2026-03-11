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
	"github.com/synnaxlabs/arc/program"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

// Config provides dependencies and context for creating node instances.
type Config struct {
	alamos.Instrumentation
	// Node is the IR definition for this node.
	Node ir.Node
	// State provides access to input/output data and channel I/O.
	State *State
	// Program contains the arc program for accessing global state and functions.
	Program program.Program
}

// Factory creates node instances from IR definitions.
// Implementations check the node type and return query.NotFound if they cannot
// handle the given type.
type Factory interface {
	// Create constructs a node from the given configuration.
	// Returns query.NotFound if this factory cannot handle cfg.Node.Type.
	Create(ctx context.Context, cfg Config) (Node, error)
}

// CompoundFactory tries each factory in order until one succeeds. A factory that
// returns query.ErrNotFound is skipped; any other error is returned immediately.
type CompoundFactory []Factory

func (f CompoundFactory) Create(ctx context.Context, cfg Config) (Node, error) {
	for _, factory := range f {
		n, err := factory.Create(ctx, cfg)
		if err == nil {
			return n, nil
		}
		if errors.Is(err, query.ErrNotFound) {
			continue
		}
		return nil, err
	}
	return nil, query.ErrNotFound
}
