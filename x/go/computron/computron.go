// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package computron

import (
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	lua "github.com/yuin/gopher-lua"
)

type Calculator struct {
	l *lua.LState
	f *lua.LFunction
}

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

func SetLValueOnSeries(v lua.LValue, dataType telem.DataType, series telem.Series, index int64) telem.Series {
	switch v.Type() {
	case lua.LTNumber:
		num := float64(v.(lua.LNumber))
		switch dataType {
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
	// A callstack size of 20 is more than enough for all calculations.
	CallStackSize: 20,
	// Keep a fixed size stack to keep CPU overhead low.
	MinimizeStackMemory: false,
}

func Open(script string) (expr *Calculator, err error) {
	expr = &Calculator{l: lua.NewState(luaOptions)}
	expr.f, err = expr.l.LoadString(script)
	if err != nil {
		return nil, errors.Wrap(err, "invalid Lua syntax")
	}
	return
}

func (c *Calculator) Set(name string, value lua.LValue) { c.l.SetGlobal(name, value) }

func (c *Calculator) Run() (result lua.LValue, err error) {
	if err = c.l.CallByParam(lua.P{Fn: c.f, NRet: 1, Protect: true}); err != nil {
		return
	}
	result = c.l.Get(-1)
	c.l.Pop(1)
	return result, nil
}

// Close clears all calculation resources. Once Close is called, no other methods should
// be called on the calculator.
func (c *Calculator) Close() { c.l.Close() }
