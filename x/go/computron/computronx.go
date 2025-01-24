package computron

import (
	"github.com/synnaxlabs/x/telem"

	"github.com/synnaxlabs/x/errors"
	lua "github.com/yuin/gopher-lua"
)

type Expression struct {
	l *lua.LState
	f *lua.LFunction
}

func SeriesToLua(series telem.Series) lua.LValue {
	switch series.DataType {
	case telem.Int8T:
		return lua.LNumber(telem.ValueAt[int8](series, -1))
	case telem.Int16T:
		return lua.LNumber(telem.ValueAt[int16](series, -1))
	case telem.Int32T:
		return lua.LNumber(telem.ValueAt[int32](series, -1))
	case telem.Int64T:
		return lua.LNumber(telem.ValueAt[int64](series, -1))
	case telem.Uint8T:
		return lua.LNumber(telem.ValueAt[uint8](series, -1))
	case telem.Uint16T:
		return lua.LNumber(telem.ValueAt[uint16](series, -1))
	case telem.Uint32T:
		return lua.LNumber(telem.ValueAt[uint32](series, -1))
	case telem.Uint64T:
		return lua.LNumber(telem.ValueAt[uint64](series, -1))
	case telem.Float32T:
		return lua.LNumber(telem.ValueAt[float32](series, -1))
	case telem.Float64T:
		return lua.LNumber(telem.ValueAt[float64](series, -1))
	case telem.StringT:
		return lua.LString(series.Split()[0])
	default:
		return lua.LNil
	}
}

func LuaToSeries(v lua.LValue, dataType telem.DataType) telem.Series {
	switch v.Type() {
	case lua.LTNumber:
		num := float64(v.(lua.LNumber))
		switch dataType {
		case telem.Int8T:
			return telem.NewSeriesV[int8](int8(num))
		case telem.Int16T:
			return telem.NewSeriesV[int16](int16(num))
		case telem.Int32T:
			return telem.NewSeriesV[int32](int32(num))
		case telem.Int64T:
			return telem.NewSeriesV[int64](int64(num))
		case telem.Uint8T:
			return telem.NewSeriesV[uint8](uint8(num))
		case telem.Uint16T:
			return telem.NewSeriesV[uint16](uint16(num))
		case telem.Uint32T:
			return telem.NewSeriesV[uint32](uint32(num))
		case telem.Uint64T:
			return telem.NewSeriesV[uint64](uint64(num))
		case telem.Float32T:
			return telem.NewSeriesV[float32](float32(num))
		case telem.Float64T:
			return telem.NewSeriesV[float64](num)
		}
	case lua.LTString:
		return telem.NewStringsV(string(v.(lua.LString)))
	}
	return telem.Series{}
}

func OpenExpression(script string) (expr *Expression, err error) {
	if script == "" {
		return nil, errors.New("empty expression")
	}
	expr = &Expression{
		l: lua.NewState(lua.Options{
			CallStackSize:       20,
			MinimizeStackMemory: false,
		}),
	}
	expr.f, err = expr.l.LoadString(script)
	if err != nil {
		return nil, errors.Wrap(err, "invalid Lua syntax")
	}
	if expr.f == nil {
		return nil, errors.New("expression did not compile to a valid Lua chunk")
	}
	return
}

func (c *Expression) Set(name string, value lua.LValue) { c.l.SetGlobal(name, value) }

func (c *Expression) Run() (result lua.LValue, err error) {
	if err = c.l.CallByParam(lua.P{
		Fn:      c.f,
		NRet:    1,
		Protect: true,
	}); err != nil {
		return
	}
	result = c.l.Get(-1)
	c.l.Pop(1)
	return result, nil
}

func (c *Expression) Close() {
	c.l.Close()
}
