// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package expression

import (
	"strconv"

	"github.com/synnaxlabs/slate/parser"
	"github.com/synnaxlabs/slate/types"
	"github.com/synnaxlabs/x/errors"
)

// compileLiteral compiles a literal value
func (e *Compiler) compileLiteral(lit parser.ILiteralContext) (types.Type, error) {
	if num := lit.NumericLiteral(); num != nil {
		return e.compileNumericLiteral(num)
	}
	if temp := lit.TemporalLiteral(); temp != nil {
		return types.TimeSpan{}, nil
	}
	if str := lit.STRING_LITERAL(); str != nil {
		return nil, errors.New("string literals are not yet supported")
	}
	if series := lit.SeriesLiteral(); series != nil {
		return nil, errors.New("series literals not yet implemented")
	}
	return nil, errors.New("unknown literal type")
}

func (e *Compiler) compileNumericLiteral(num parser.INumericLiteralContext) (types.Type, error) {
	if intLit := num.INTEGER_LITERAL(); intLit != nil {
		text := intLit.GetText()
		value, err := strconv.ParseInt(text, 10, 64)
		if err != nil {
			return nil, errors.Newf("invalid integer literal: %s", text)
		}
		e.encoder.WriteI64Const(value)
		return types.I64{}, nil
	}
	if floatLit := num.FLOAT_LITERAL(); floatLit != nil {
		text := floatLit.GetText()
		value, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return nil, errors.Newf("invalid float literal: %s", text)
		}
		e.encoder.WriteF64Const(value)
		return types.F64{}, nil
	}
	return nil, errors.New("unknown numeric literal")
}
