// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package wasm

import (
	"math"

	"github.com/synnaxlabs/arc/runtime/node"
	stlstrings "github.com/synnaxlabs/arc/stl/strings"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/tetratelabs/wazero/api"
	"go.uber.org/zap"
)

type Module struct {
	Module        api.Module
	Memory        api.Memory
	Strings       *stlstrings.ProgramState
	NodeKeySetter NodeKeySetter
	arena         *arena
}

func (w *Module) Create(cfg node.Config) (node.Node, error) {
	irFn, ok := cfg.Program.Functions.Find(cfg.Node.Type)
	if !ok {
		return nil, query.ErrNotFound
	}
	fn := w.Module.ExportedFunction(cfg.Node.Type)
	if fn == nil {
		return nil, query.ErrNotFound
	}
	def := fn.Definition()
	if len(irFn.Inputs) != len(def.ParamTypes()) {
		return nil, errors.Newf(
			"node %s declares %d inputs, but %q takes %d params",
			cfg.Node.Key, len(irFn.Inputs), cfg.Node.Type, len(def.ParamTypes()),
		)
	}
	params, varInputs, err := w.literalParams(cfg.Node.Inputs, len(irFn.Inputs))
	if err != nil {
		return nil, err
	}
	base := cfg.Program.OutputMemoryBases[cfg.Node.Type]
	memOffsets, err := outputMemOffsets(cfg.Node.Key, irFn.Outputs, base)
	if err != nil {
		return nil, err
	}

	// The reused call stack is separate from params because CallWithStack overwrites it
	// with the results, and params holds literal values that persist across calls.
	stack := make([]uint64, max(len(def.ParamTypes()), len(def.ResultTypes())))

	selIdx := -1
	if idx, err := cfg.State.ResolveInput("$sel"); err == nil {
		selIdx = idx
	}

	n := &nodeImpl{
		State:         cfg.State,
		ir:            cfg.Node,
		fn:            fn,
		mem:           w.Memory,
		fnOutputs:     irFn.Outputs,
		memOffsets:    memOffsets,
		outputValues:  make([]result, len(irFn.Outputs)),
		memBase:       base,
		params:        params,
		stack:         stack,
		offsets:       make([]int, len(irFn.Outputs)),
		selIdx:        selIdx,
		nodeKeySetter: w.NodeKeySetter,
		stringInputs:  kindFlags(irFn.Inputs, types.KindString),
		chanInputs:    kindFlags(irFn.Inputs, types.KindChan),
		varInputs:     varInputs,
		stringOutputs: kindFlags(irFn.Outputs, types.KindString),
		strings:       w.Strings,
		batch:         w.newBatchCall(cfg, irFn, base),
	}
	return n, nil
}

// literalParams fills the WASM param slot of each literal input with its constant
// value. Edge-fed inputs (nil Value) are streamed per sample in Next, and variable
// reference inputs are flagged in the returned varInputs.
func (w *Module) literalParams(
	inputs types.Params,
	count int,
) (params []uint64, varInputs []bool, err error) {
	params = make([]uint64, count)
	varInputs = make([]bool, len(inputs))
	for i, param := range inputs {
		if param.Type.Kind == types.KindVarRef {
			varInputs[i] = true
			continue
		}
		if param.Value == nil {
			continue
		}
		value := types.DecodeValue(param.Value, param.Type)
		if s, ok := value.(string); ok {
			// A literal string gets a stable handle that persists across Flush calls.
			params[i] = uint64(w.Strings.CreateLiteral(s))
			continue
		}
		if params[i], err = ConvertLiteralValue(value); err != nil {
			return nil, nil, err
		}
	}
	return params, varInputs, nil
}

// outputMemOffsets returns where in guest memory each output's value lands, after the
// 8-byte dirty flags at base.
func outputMemOffsets(
	nodeKey string,
	outputs types.Params,
	base uint32,
) ([]uint32, error) {
	memOffsets := make([]uint32, len(outputs))
	offset := base + 8
	for i, t := range outputs {
		if !t.Type.IsValid() {
			return nil, errors.Newf(
				"node %s has output %q with unresolved type",
				nodeKey, t.Name,
			)
		}
		memOffsets[i] = offset
		if t.Type.Kind == types.KindString {
			offset += 4 // strings are i32 handles in WASM
		} else {
			offset += uint32(t.Type.Density())
		}
	}
	return memOffsets, nil
}

func kindFlags(params types.Params, kind types.Kind) []bool {
	flags := make([]bool, len(params))
	for i, p := range params {
		flags[i] = p.Type.Kind == kind
	}
	return flags
}

// ConvertLiteralValue converts a literal value to uint64 for WASM function calls.
func ConvertLiteralValue(v any) (uint64, error) {
	switch val := v.(type) {
	case bool:
		if val {
			return 1, nil
		}
		return 0, nil
	case int8:
		return uint64(val), nil
	case int16:
		return uint64(val), nil
	case int32:
		return uint64(val), nil
	case int64:
		return uint64(val), nil
	case uint8:
		return uint64(val), nil
	case uint16:
		return uint64(val), nil
	case uint32:
		return uint64(val), nil
	case uint64:
		return val, nil
	case float32:
		return uint64(math.Float32bits(val)), nil
	case float64:
		return math.Float64bits(val), nil
	case telem.TimeStamp:
		return uint64(val), nil
	case telem.TimeSpan:
		return uint64(val), nil
	default:
		err := errors.Newf("unsupported literal value type: %T", v)
		zap.S().DPanic(err.Error())
		return 0, err
	}
}
