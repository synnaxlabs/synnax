// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package expression

import (
	"math"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/compiler/context"
	"github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/types"
)

// floatCheck tests |x| against +Inf with op. NaN fails every comparison, so a check
// that must pass for NaN negates a comparison that passes for all other values.
type floatCheck struct {
	op      string
	negated bool
}

// floatChecks compile inline, so they need no host function in either runtime.
var floatChecks = map[string]floatCheck{
	"math.is_nan":    {op: "<=", negated: true},
	"math.is_inf":    {op: "=="},
	"math.is_finite": {op: "<"},
}

func emitFloatCheck[T antlr.ParserRuleContext](
	ctx context.Context[T],
	check floatCheck,
	t types.Type,
) (types.Type, error) {
	if t.Kind == types.KindF32 {
		ctx.Writer.WriteOpcode(wasm.OpF32Abs)
		ctx.Writer.WriteF32Const(float32(math.Inf(1)))
	} else {
		ctx.Writer.WriteOpcode(wasm.OpF64Abs)
		ctx.Writer.WriteF64Const(math.Inf(1))
	}
	if err := ctx.Writer.WriteBinaryOpInferred(check.op, t); err != nil {
		return types.Type{}, err
	}
	if check.negated {
		ctx.Writer.WriteI32Eqz()
	}
	return types.Bool(), nil
}
