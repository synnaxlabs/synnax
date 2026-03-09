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
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/stl/channel"
	"github.com/synnaxlabs/arc/stl/constant"
	"github.com/synnaxlabs/arc/stl/control"
	"github.com/synnaxlabs/arc/stl/errors"
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
	"github.com/synnaxlabs/arc/types"
)

var SymbolResolver = symbol.CompoundResolver{
	topLevelSymbolResolver,
	channel.CompilerSymbolResolver,
	constant.SymbolResolver,
	control.SymbolResolver,
	errors.SymbolResolver,
	math.SymbolResolver,
	op.SymbolResolver,
	selector.SymbolResolver,
	series.CompilerSymbolResolver,
	stable.SymbolResolver,
	stage.SymbolResolver,
	stat.SymbolResolver,
	stateful.SymbolResolver,
	stringsstate.SymbolResolver,
	stringsstate.SymbolResolver,
	time.SymbolResolver,
}

var topLevelSymbolResolver = symbol.MapResolver{
	"now": {
		Name: "now",
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.TimeStamp()}},
		}),
	},
	"len": {
		Name: "len",
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Inputs:  types.Params{{Name: ir.DefaultInputParam, Type: types.Variable("T", nil)}},
			Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I64()}},
		}),
	},
}
