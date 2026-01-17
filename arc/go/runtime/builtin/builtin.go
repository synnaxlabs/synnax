// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package builtin provides symbol resolvers for built-in functions like now(), len(),
// and log().
package builtin

import (
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
)

const (
	nowSymbolName = "now"
	lenSymbolName = "len"
)

var (
	nowSymbol = symbol.Symbol{
		Name: nowSymbolName,
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.TimeStamp()}},
		}),
	}
	lenSymbol = symbol.Symbol{
		Name: lenSymbolName,
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Inputs:  types.Params{{Name: ir.DefaultInputParam, Type: types.Series(types.Variable("T", nil))}},
			Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I64()}},
		}),
	}
	SymbolResolver = symbol.MapResolver{
		nowSymbolName: nowSymbol,
		lenSymbolName: lenSymbol,
	}
)
