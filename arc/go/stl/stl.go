// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package stl defines the standard library module interfaces for Arc. A Module is the
// unit of STL organization: it provides symbols for the analyzer, node factories for
// the scheduler, and host function implementations for the WASM runtime.
package stl

import (
	"context"

	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/stl/channel"
	"github.com/synnaxlabs/arc/stl/constant"
	"github.com/synnaxlabs/arc/stl/control"
	stlerrors "github.com/synnaxlabs/arc/stl/errors"
	"github.com/synnaxlabs/arc/stl/math"
	"github.com/synnaxlabs/arc/stl/op"
	"github.com/synnaxlabs/arc/stl/selector"
	"github.com/synnaxlabs/arc/stl/series"
	"github.com/synnaxlabs/arc/stl/stable"
	"github.com/synnaxlabs/arc/stl/stage"
	"github.com/synnaxlabs/arc/stl/stat"
	"github.com/synnaxlabs/arc/stl/stateful"
	stringsstate "github.com/synnaxlabs/arc/stl/strings"
	"github.com/synnaxlabs/arc/stl/time"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

var SymbolResolver = symbol.CompoundResolver{
	channel.SymbolResolver,
	constant.SymbolResolver,
	control.SymbolResolver,
	stlerrors.SymbolResolver,
	math.SymbolResolver,
	op.SymbolResolver,
	selector.SymbolResolver,
	series.SymbolResolver,
	stable.SymbolResolver,
	stage.SymbolResolver,
	stat.SymbolResolver,
	stateful.SymbolResolver,
	stringsstate.SymbolResolver,
	time.SymbolResolver,
}

// CompoundFactory tries each factory in order until one succeeds. A factory that
// returns query.ErrNotFound is skipped; any other error is returned immediately.
type CompoundFactory []node.Factory

func (f CompoundFactory) Create(
	ctx context.Context,
	cfg node.Config,
) (node.Node, error) {
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
