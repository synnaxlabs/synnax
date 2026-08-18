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
	"context"

	"github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
)

// DeriveWASMCoordinates returns the per-module WASM import coordinate for a
// host-import reference. wasmModule is ref.module verbatim; wasmFuncName is
// ref.name, with a type suffix appended when the underlying symbol is
// polymorphic (or when an explicit suffix was supplied via
// ResolveImportWithSuffix). Panics on a local reference (ref.module == "")
// since locals are not WASM imports.
func DeriveWASMCoordinates(
	ctx context.Context,
	scope *symbol.Symbol,
	ref pendingRef,
) (wasmModule, wasmFuncName string) {
	if ref.module == "" {
		panic("DeriveWASMCoordinates called on local reference: " + ref.name)
	}
	wasmModule = ref.module
	wasmFuncName = ref.name
	var suffix string
	if ref.typeSuffix != "" {
		suffix = ref.typeSuffix
	} else if scope != nil {
		modSym, err := scope.Resolve(ctx, ref.module, symbol.IncludeInternal)
		if err == nil {
			original, err := modSym.Resolve(ctx, ref.name, symbol.IncludeInternal)
			if err == nil && original.Type.Kind == types.KindFunction {
				suffix = DeriveTypeSuffix(original.Type, ref.concreteType)
			}
		}
	}
	if suffix != "" {
		wasmFuncName = wasmFuncName + "_" + suffix
	}
	return wasmModule, wasmFuncName
}

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
