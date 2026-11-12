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
	"strings"

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
	// Program contains the Arc program for accessing global state and functions.
	Program program.Program
}

// Factory creates node instances from IR definitions. Implementations check the node
// type and return query.NotFound if they cannot handle the given type.
type Factory interface {
	// Create constructs a node from the given configuration. Returns query.ErrNotFound
	// if this factory cannot handle cfg.Node.Type.
	Create(Config) (Node, error)
}

// moduleNamer is an optional interface that factories can implement to declare which
// module they belong to (e.g. "control", "time"). When the CompoundFactory encounters a
// qualified node type like "time.interval", it skips factories whose module name
// doesn't match the prefix. Factories that don't implement this interface are always
// considered.
type moduleNamer interface{ ModuleName() string }

// CompoundFactory tries each factory in order until one succeeds. A factory that
// returns query.ErrNotFound is skipped; any other error is returned immediately.
type CompoundFactory []Factory

func (f CompoundFactory) Create(cfg Config) (Node, error) {
	// A program function is keyed by its qualified name, so the full type is tried
	// first. Only then is the module prefix split off for the host factories.
	n, err := f.create(cfg, "")
	if !errors.Is(err, query.ErrNotFound) {
		return n, err
	}
	modulePrefix, bare, ok := strings.CutLast(cfg.Node.Type, ".")
	if !ok {
		return nil, query.ErrNotFound
	}
	cfg.Node.Type = bare
	return f.create(cfg, modulePrefix)
}

func (f CompoundFactory) create(cfg Config, modulePrefix string) (Node, error) {
	for _, factory := range f {
		// When the IR node has a module prefix (e.g. "control" from
		// "control.set_authority"), skip factories belonging to a different module.
		// Factories that don't implement moduleNamer are always considered.
		if modulePrefix != "" {
			if namer, ok := factory.(moduleNamer); ok {
				if name := namer.ModuleName(); name != "" && name != modulePrefix {
					continue
				}
			}
		}
		n, err := factory.Create(cfg)
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
