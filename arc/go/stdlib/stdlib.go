// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package stdlib provides standard library module definitions for Arc programs.
package stdlib

import (
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
)

// mathFunc creates a symbol for a unary math function (f64 -> f64).
func mathFunc(name string) symbol.Symbol {
	return symbol.Symbol{
		Name: name,
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Inputs:  types.Params{{Name: "x", Type: types.F64()}},
			Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.F64()}},
		}),
	}
}

// mathFunc2 creates a symbol for a binary math function (f64, f64 -> f64).
func mathFunc2(name string) symbol.Symbol {
	return symbol.Symbol{
		Name: name,
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Inputs: types.Params{
				{Name: "a", Type: types.F64()},
				{Name: "b", Type: types.F64()},
			},
			Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.F64()}},
		}),
	}
}

// Modules contains all standard library module definitions.
// Module names map to Resolvers that can look up their members.
var Modules = map[string]symbol.Resolver{
	"math": symbol.MapResolver{
		"sqrt":  mathFunc("sqrt"),
		"sin":   mathFunc("sin"),
		"cos":   mathFunc("cos"),
		"tan":   mathFunc("tan"),
		"asin":  mathFunc("asin"),
		"acos":  mathFunc("acos"),
		"atan":  mathFunc("atan"),
		"abs":   mathFunc("abs"),
		"floor": mathFunc("floor"),
		"ceil":  mathFunc("ceil"),
		"round": mathFunc("round"),
		"exp":   mathFunc("exp"),
		"log":   mathFunc("log"),
		"log10": mathFunc("log10"),
		"pow":   mathFunc2("pow"),
		"min":   mathFunc2("min"),
		"max":   mathFunc2("max"),
		"atan2": mathFunc2("atan2"),
	},
	"time": symbol.MapResolver{
		"now": {
			Name: "now",
			Kind: symbol.KindFunction,
			Type: types.Function(types.FunctionProperties{
				Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.TimeStamp()}},
			}),
		},
		"elapsed": {
			Name: "elapsed",
			Kind: symbol.KindFunction,
			Type: types.Function(types.FunctionProperties{
				Inputs:  types.Params{{Name: "since", Type: types.TimeStamp()}},
				Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.TimeSpan()}},
			}),
		},
	},
}
