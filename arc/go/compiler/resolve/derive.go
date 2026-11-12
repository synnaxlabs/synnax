// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package resolve

import (
	"github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/types"
)

// DeriveTypeSuffix returns the type suffix (e.g., "f64", "u8") for a concrete
// instantiation of a polymorphic symbol. Returns "" if the original type has no
// type variables.
func DeriveTypeSuffix(originalType, concreteType types.Type) string {
	if originalType.Kind != types.KindFunction ||
		concreteType.Kind != types.KindFunction {
		return ""
	}
	for i, inp := range originalType.Inputs {
		if inp.Type.Kind == types.KindVariable {
			if i < len(concreteType.Inputs) {
				return suffixForType(concreteType.Inputs[i].Type)
			}
		}
	}
	for i, out := range originalType.Outputs {
		if out.Type.Kind == types.KindVariable {
			if i < len(concreteType.Outputs) {
				return suffixForType(concreteType.Outputs[i].Type)
			}
		}
	}
	return ""
}

// suffixForType returns the WASM coordinate suffix for a concrete type. Units
// are stripped because WASM types (i32/i64/f32/f64) are unit-blind, so a
// timestamp (i64 ns) channel and an int64 channel both bind to the same
// write_i64 host function.
func suffixForType(t types.Type) string {
	t.Unit = nil
	return t.String()
}

// DeriveWASMFuncType converts an Arc function type to a WASM FunctionType.
// Each input param maps to a WASM param, each output maps to a WASM result.
func DeriveWASMFuncType(t types.Type) wasm.FunctionType {
	ft := wasm.FunctionType{}
	if t.Kind != types.KindFunction {
		return ft
	}
	for _, inp := range t.Inputs {
		ft.Params = append(ft.Params, wasm.ConvertType(inp.Type))
	}
	for _, out := range t.Outputs {
		ft.Results = append(ft.Results, wasm.ConvertType(out.Type))
	}
	return ft
}
