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
	"iter"

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
	Params OrderedMap[string, symbol.Type]
	Return symbol.Type
}

func NewFunction() Function {
	return Function{Params: OrderedMap[string, symbol.Type]{}}
}

func (f Function) String() string { return "function" }

type OrderedMap[K comparable, V any] struct {
	keys   []K
	values []V
}

func NewOrderedMap[K comparable, V any](keys []K, values []V) OrderedMap[K, V] {
	return OrderedMap[K, V]{keys, values}
}

func NewTask() Task {
	return Task{
		Params: OrderedMap[string, symbol.Type]{},
		Config: OrderedMap[string, symbol.Type]{},
	}
}

func (m *OrderedMap[K, V]) Count() int {
	return len(m.keys)
}

func (m *OrderedMap[K, V]) At(i int) V {
	return m.values[i]
}

func (m *OrderedMap[K, V]) Iter() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for i, k := range m.keys {
			if !yield(k, m.values[i]) {
				return
			}
		}
	}
}

func (m *OrderedMap[K, V]) Get(key K) (V, bool) {
	for i, k := range m.keys {
		if k == key {
			return m.values[i], true
		}
	}
	var res V
	return res, false
}

func (m *OrderedMap[K, V]) Put(key K, value V) bool {
	for _, k := range m.keys {
		if k == key {
			return false
		}
	}
	m.keys = append(m.keys, key)
	m.values = append(m.values, value)
	return true
}

type Task struct {
	Config OrderedMap[string, symbol.Type]
	Params OrderedMap[string, symbol.Type]
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
	if ch, isChan := t.(Chan); isChan {
		t = ch.ValueType
	}
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

func Equal(t symbol.Type, v symbol.Type) bool {
	return t == v
}
