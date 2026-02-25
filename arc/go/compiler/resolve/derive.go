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
	"strings"

	"github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
)

// DeriveWASMCoordinates maps a qualified Arc name to a per-module WASM import
// coordinate. The qualified name is split on the last dot to produce the WASM
// module name and function base name. If the original symbol has type variables,
// a type suffix is appended based on the concrete type.
func DeriveWASMCoordinates(
	symbols symbol.Resolver,
	ref pendingRef,
) (wasmModule string, wasmFuncName string) {
	if idx := strings.LastIndex(ref.qualifiedName, "."); idx >= 0 {
		wasmModule = ref.qualifiedName[:idx]
		wasmFuncName = ref.qualifiedName[idx+1:]
	} else {
		panic("unqualified host function name: " + ref.qualifiedName)
	}
	var suffix string
	if ref.typeSuffix != "" {
		suffix = ref.typeSuffix
	} else if symbols != nil {
		original, err := symbols.Resolve(context.Background(), ref.qualifiedName)
		if err == nil && original.Type.Kind == types.KindFunction {
			suffix = DeriveTypeSuffix(original.Type, ref.concreteType)
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
	if originalType.Kind != types.KindFunction || concreteType.Kind != types.KindFunction {
		return ""
	}
	for i, inp := range originalType.Inputs {
		if inp.Type.Kind == types.KindVariable {
			if i < len(concreteType.Inputs) {
				return concreteType.Inputs[i].Type.String()
			}
		}
	}
	for i, out := range originalType.Outputs {
		if out.Type.Kind == types.KindVariable {
			if i < len(concreteType.Outputs) {
				return concreteType.Outputs[i].Type.String()
			}
		}
	}
	return ""
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
