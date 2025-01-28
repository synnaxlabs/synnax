// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package computron provides a lua based calculation system for transforming data.
package computron

import (
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	lua "github.com/yuin/gopher-lua"
	"github.com/yuin/gopher-lua/parse"
)

// Calculator is a lua function used to perform calculations on data.
type Calculator struct {
	// luaState is the lua state used to run the calculation.
	luaState *lua.LState
	// compiledExpr is the compiled lua function that performs the calculation.
	compiledExpr *lua.LFunction
}

// LValueFromSeries converts a numeric series value at an index to a lua value.
func LValueFromSeries(series telem.Series, index int64) lua.LValue {
	switch series.DataType {
	case telem.Int8T:
		return lua.LNumber(telem.ValueAt[int8](series, index))
	case telem.Int16T:
		return lua.LNumber(telem.ValueAt[int16](series, index))
	case telem.Int32T:
		return lua.LNumber(telem.ValueAt[int32](series, index))
	case telem.Int64T:
		return lua.LNumber(telem.ValueAt[int64](series, index))
	case telem.Uint8T:
		return lua.LNumber(telem.ValueAt[uint8](series, index))
	case telem.Uint16T:
		return lua.LNumber(telem.ValueAt[uint16](series, index))
	case telem.Uint32T:
		return lua.LNumber(telem.ValueAt[uint32](series, index))
	case telem.Uint64T:
		return lua.LNumber(telem.ValueAt[uint64](series, index))
	case telem.Float32T:
		return lua.LNumber(telem.ValueAt[float32](series, index))
	case telem.Float64T:
		return lua.LNumber(telem.ValueAt[float64](series, index))
	case telem.StringT:
		return lua.LString(series.Split()[index])
	default:
		return lua.LNil
	}
}

// SetLValueOnSeries sets the value of a series at an index to the given lua value. v must
// be a valid numeric lua value, series must have sufficient capacity to store the value,
// and index must be within the bounds of the series.
func SetLValueOnSeries(
	v lua.LValue,
	series telem.Series,
	index int64,
) telem.Series {
	switch v.Type() {
	case lua.LTNumber:
		num := float64(v.(lua.LNumber))
		switch series.DataType {
		case telem.Int8T:
			telem.SetValueAt[int8](series, index, int8(num))
		case telem.Int16T:
			telem.SetValueAt[int16](series, index, int16(num))
		case telem.Int32T:
			telem.SetValueAt[int32](series, index, int32(num))
		case telem.Int64T:
			telem.SetValueAt[int64](series, index, int64(num))
		case telem.Uint8T:
			telem.SetValueAt[uint8](series, index, uint8(num))
		case telem.Uint16T:
			telem.SetValueAt[uint16](series, index, uint16(num))
		case telem.Uint32T:
			telem.SetValueAt[uint32](series, index, uint32(num))
		case telem.Uint64T:
			telem.SetValueAt[uint64](series, index, uint64(num))
		case telem.Float32T:
			telem.SetValueAt[float32](series, index, float32(num))
		case telem.Float64T:
			telem.SetValueAt[float64](series, index, float64(num))
		}
	default:
		return series
	}
	return series
}

var luaOptions = lua.Options{
	// A callstack size of 10 is more than enough for all calculations.
	CallStackSize: 10,
	// Keep a fixed size stack to keep CPU overhead low.
	MinimizeStackMemory: false,
}

var _ error = &lua.ApiError{}

func parseSyntaxError(err error) error {
	if err == nil {
		return nil
	}
	apiErr, ok := err.(*lua.ApiError)
	if !ok {
		return err
	}
	pErr, ok := apiErr.Cause.(*parse.Error)
	if !ok {
		return err
	}
	// Return a wrapped error with the parse error message, position information, and token
	return errors.Wrapf(err,
		"syntax error at line %d column %d (token '%s'): %s",
		pErr.Pos.Line,
		pErr.Pos.Column,
		pErr.Token,
		pErr.Message,
	)
}

// Open creates a new calculator with the given expression as the calculation.
func Open(expr string) (calc *Calculator, err error) {
	calc = &Calculator{luaState: lua.NewState(luaOptions)}

	// Register the get function to access hyphenated variable names
	calc.luaState.SetGlobal("get", calc.luaState.NewFunction(func(L *lua.LState) int {
		name := L.ToString(1)
		value := L.GetGlobal(name)
		L.Push(value)
		return 1
	}))

	calc.compiledExpr, err = calc.luaState.LoadString(expr)
	return calc, parseSyntaxError(err)
}

// Set sets a variable in the calculator's lua state. This variable will be available
// the next time the expression is evaluated, and will override any previous variables
// set in the state.
func (c *Calculator) Set(name string, value lua.LValue) { c.luaState.SetGlobal(name, value) }

// Run evaluates the calculator's expression and returns the result. If an error occurs
// during evaluation, the error is returned.
func (c *Calculator) Run() (result lua.LValue, err error) {
	if err = c.luaState.CallByParam(lua.P{Fn: c.compiledExpr, NRet: 1, Protect: true}); err != nil {
		return
	}
	result = c.luaState.Get(-1)
	c.luaState.Pop(1)
	return result, nil
}

// Close clears all calculation resources. Once Close is called, no other methods
// should be called on the calculator.
func (c *Calculator) Close() { c.luaState.Close() }
