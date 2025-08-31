// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types

import (
	"github.com/synnaxlabs/slate/analyzer/symbol"
)

type U8 struct{}

func (u U8) String() string { return "u8" }

type U16 struct{}

func (u U16) String() string { return "u16" }

type U32 struct{}

func (u U32) String() string { return "u32" }

type U64 struct{}

func (u U64) String() string { return "u64" }

type I8 struct{}

func (i I8) String() string { return "i8" }

type I16 struct{}

func (i I16) String() string { return "i16" }

type I32 struct{}

func (i I32) String() string { return "i32" }

type I64 struct{}

func (i I64) String() string { return "i64" }

type F32 struct{}

func (f F32) String() string { return "f32" }

type F64 struct{}

func (f F64) String() string { return "f64" }

type String struct{}

func (s String) String() string { return "string" }

type Series struct {
	ValueType symbol.Type
}

func (s Series) String() string { return "series " + s.ValueType.String() }

type Function struct {
	Params map[string]symbol.Type
	Return symbol.Type
}

func (f Function) String() string { return "function" }

type Task struct {
	Config map[string]symbol.Type
	Params map[string]symbol.Type
	Return symbol.Type
}

func (t Task) String() string { return "task" }

type Chan struct {
	ValueType symbol.Type
}

func (c Chan) String() string { return "chan " + c.ValueType.String() }

type TimeSpan struct{}

func (t TimeSpan) String() string { return "timespan" }

type TimeStamp struct{}

func (t TimeStamp) String() string { return "timestamp" }

func IsNumeric(t symbol.Type) bool {
	switch t {
	case U8{}, U16{}, U32{}, U64{}, I8{}, I16{}, I32{}, I64{}, F32{}, F64{}:
		return true
	default:
		return false
	}
}

func IsInteger(t symbol.Type) bool {
	switch t {
	case U8{}, U16{}, U32{}, U64{}, I8{}, I16{}, I32{}, I64{}:
		return true
	default:
		return false
	}
}

func IsSignedInteger(t symbol.Type) bool {
	switch t {
	case I8{}, I16{}, I32{}, I64{}:
		return true
	default:
		return false
	}
}

func IsUnsignedInteger(t symbol.Type) bool {
	switch t {
	case U8{}, U16{}, U32{}, U64{}:
		return true
	default:
		return false
	}
}

func IsFloat(t symbol.Type) bool {
	switch t {
	case F32{}, F64{}:
		return true
	default:
		return false
	}
}

func IsBool(t symbol.Type) bool {
	_, ok := t.(U8)
	return ok
}
