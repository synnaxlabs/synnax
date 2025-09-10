// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package runtime

import (
	"context"
	"fmt"

	"github.com/synnaxlabs/arc/ir"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// Bindings provides a mock implementation of the arc runtime with configurable function implementations
type Bindings struct {
	// Channel operations
	ChannelRead         map[string]interface{}
	ChannelWrite        map[string]interface{}
	ChannelBlockingRead map[string]interface{}

	// State operations
	StateLoad  map[string]interface{}
	StateStore map[string]interface{}

	// Series operations
	SeriesCreateEmpty map[string]interface{}
	SeriesSetElement  map[string]interface{}
	SeriesIndex       map[string]interface{}
	SeriesElementAdd  map[string]interface{}
	SeriesElementSub  map[string]interface{}
	SeriesElementMul  map[string]interface{}
	SeriesElementDiv  map[string]interface{}
	SeriesSeriesAdd   map[string]interface{}
	SeriesSeriesSub   map[string]interface{}
	SeriesSeriesMul   map[string]interface{}
	SeriesSeriesDiv   map[string]interface{}
	SeriesCompareGT   map[string]interface{}
	SeriesCompareLT   map[string]interface{}
	SeriesCompareGE   map[string]interface{}
	SeriesCompareLE   map[string]interface{}
	SeriesCompareEQ   map[string]interface{}
	SeriesCompareNE   map[string]interface{}

	// Generic operations
	Now               func(context.Context) uint64
	Len               func(context.Context, uint32) uint64
	Panic             func(context.Context, uint32, uint32)
	MathPowF32        func(context.Context, float32, float32) float32
	MathPowF64        func(context.Context, float64, float64) float64
	SeriesLen         func(context.Context, uint32) uint64
	SeriesSlice       func(context.Context, uint32, uint32, uint32) uint32
	StringFromLiteral func(context.Context, uint32, uint32) uint32
	StringConcat      func(context.Context, uint32, uint32) uint32
	StringEqual       func(context.Context, uint32, uint32) uint32
	StringLen         func(context.Context, uint32) uint32
}

// NewBindings creates a new mock runtime with empty function maps
func NewBindings() *Bindings {
	return &Bindings{
		ChannelRead:         make(map[string]interface{}),
		ChannelWrite:        make(map[string]interface{}),
		ChannelBlockingRead: make(map[string]interface{}),
		StateLoad:           make(map[string]interface{}),
		StateStore:          make(map[string]interface{}),
		SeriesCreateEmpty:   make(map[string]interface{}),
		SeriesSetElement:    make(map[string]interface{}),
		SeriesIndex:         make(map[string]interface{}),
		SeriesElementAdd:    make(map[string]interface{}),
		SeriesElementSub:    make(map[string]interface{}),
		SeriesElementMul:    make(map[string]interface{}),
		SeriesElementDiv:    make(map[string]interface{}),
		SeriesSeriesAdd:     make(map[string]interface{}),
		SeriesSeriesSub:     make(map[string]interface{}),
		SeriesSeriesMul:     make(map[string]interface{}),
		SeriesSeriesDiv:     make(map[string]interface{}),
		SeriesCompareGT:     make(map[string]interface{}),
		SeriesCompareLT:     make(map[string]interface{}),
		SeriesCompareGE:     make(map[string]interface{}),
		SeriesCompareLE:     make(map[string]interface{}),
		SeriesCompareEQ:     make(map[string]interface{}),
		SeriesCompareNE:     make(map[string]interface{}),
	}
}

// Bind registers all runtime functions with the wazero runtime
func (b *Bindings) Bind(ctx context.Context, rt wazero.Runtime) error {
	hostBuilder := rt.NewHostModuleBuilder("env")

	// Bind channel operations for all types
	for _, typ := range ir.Numerics {
		if err := b.bindChannelOps(hostBuilder, typ); err != nil {
			return err
		}
	}
	if err := b.bindChannelOps(hostBuilder, ir.String{}); err != nil {
		return err
	}

	// Bind state operations for all types
	for _, typ := range ir.Numerics {
		if err := b.bindStateOps(hostBuilder, typ); err != nil {
			return err
		}
	}
	if err := b.bindStateOps(hostBuilder, ir.String{}); err != nil {
		return err
	}

	// Bind series operations for numeric types
	for _, typ := range ir.Numerics {
		if err := b.bindSeriesOps(hostBuilder, typ); err != nil {
			return err
		}
	}

	// Bind generic operations
	if err := b.bindGenericOps(hostBuilder); err != nil {
		return err
	}

	_, err := hostBuilder.Instantiate(ctx)
	return err
}

// bindChannelOps binds channel operations for a specific type
func (b *Bindings) bindChannelOps(builder wazero.HostModuleBuilder, t ir.Type) error {
	typeName := t.String()

	// Channel read
	funcName := fmt.Sprintf("channel_read_%s", typeName)
	if fn, ok := b.ChannelRead[typeName]; ok {
		builder.NewFunctionBuilder().WithFunc(fn).Export(funcName)
	} else {
		builder.NewFunctionBuilder().WithFunc(b.makeChannelReadStub(typeName)).Export(funcName)
	}

	// Channel write
	funcName = fmt.Sprintf("channel_write_%s", typeName)
	if fn, ok := b.ChannelWrite[typeName]; ok {
		builder.NewFunctionBuilder().WithFunc(fn).Export(funcName)
	} else {
		builder.NewFunctionBuilder().WithFunc(b.makeChannelWriteStub(typeName)).Export(funcName)
	}

	// Blocking channel read
	funcName = fmt.Sprintf("channel_blocking_read_%s", typeName)
	if fn, ok := b.ChannelBlockingRead[typeName]; ok {
		builder.NewFunctionBuilder().WithFunc(fn).Export(funcName)
	} else {
		builder.NewFunctionBuilder().WithFunc(b.makeChannelBlockingReadStub(typeName)).Export(funcName)
	}

	return nil
}

// bindStateOps binds state operations for a specific type
func (b *Bindings) bindStateOps(builder wazero.HostModuleBuilder, t ir.Type) error {
	typeName := t.String()

	// State load
	funcName := fmt.Sprintf("state_load_%s", typeName)
	if fn, ok := b.StateLoad[typeName]; ok {
		builder.NewFunctionBuilder().WithFunc(fn).Export(funcName)
	} else {
		builder.NewFunctionBuilder().WithFunc(b.makeStateLoadStub(typeName)).Export(funcName)
	}

	// State store
	funcName = fmt.Sprintf("state_store_%s", typeName)
	if fn, ok := b.StateStore[typeName]; ok {
		builder.NewFunctionBuilder().WithFunc(fn).Export(funcName)
	} else {
		builder.NewFunctionBuilder().WithFunc(b.makeStateStoreStub(typeName)).Export(funcName)
	}

	return nil
}

// bindSeriesOps binds series operations for a specific type
func (b *Bindings) bindSeriesOps(builder wazero.HostModuleBuilder, t ir.Type) error {
	typeName := t.String()

	// Series create empty
	funcName := fmt.Sprintf("series_create_empty_%s", typeName)
	if fn, ok := b.SeriesCreateEmpty[typeName]; ok {
		builder.NewFunctionBuilder().WithFunc(fn).Export(funcName)
	} else {
		builder.NewFunctionBuilder().WithFunc(b.makeSeriesCreateEmptyStub(typeName)).Export(funcName)
	}

	// Series set element
	funcName = fmt.Sprintf("series_set_element_%s", typeName)
	if fn, ok := b.SeriesSetElement[typeName]; ok {
		builder.NewFunctionBuilder().WithFunc(fn).Export(funcName)
	} else {
		builder.NewFunctionBuilder().WithFunc(b.makeSeriesSetElementStub(typeName)).Export(funcName)
	}

	// Series index
	funcName = fmt.Sprintf("series_index_%s", typeName)
	if fn, ok := b.SeriesIndex[typeName]; ok {
		builder.NewFunctionBuilder().WithFunc(fn).Export(funcName)
	} else {
		builder.NewFunctionBuilder().WithFunc(b.makeSeriesIndexStub(typeName)).Export(funcName)
	}

	// Series arithmetic operations
	b.bindSeriesArithmeticOps(builder, typeName)

	// Series comparison operations
	b.bindSeriesComparisonOps(builder, typeName)

	return nil
}

// bindSeriesArithmeticOps binds series arithmetic operations
func (b *Bindings) bindSeriesArithmeticOps(builder wazero.HostModuleBuilder, typeName string) {
	// Element operations
	ops := map[string]map[string]interface{}{
		"add": b.SeriesElementAdd,
		"sub": b.SeriesElementSub,
		"mul": b.SeriesElementMul,
		"div": b.SeriesElementDiv,
	}

	for op, fnMap := range ops {
		funcName := fmt.Sprintf("series_element_%s_%s", op, typeName)
		if fn, ok := fnMap[typeName]; ok {
			builder.NewFunctionBuilder().WithFunc(fn).Export(funcName)
		} else {
			builder.NewFunctionBuilder().WithFunc(b.makeSeriesArithmeticStub(typeName, op, true)).Export(funcName)
		}
	}

	// Series-series operations
	ops = map[string]map[string]interface{}{
		"add": b.SeriesSeriesAdd,
		"sub": b.SeriesSeriesSub,
		"mul": b.SeriesSeriesMul,
		"div": b.SeriesSeriesDiv,
	}

	for op, fnMap := range ops {
		funcName := fmt.Sprintf("series_series_%s_%s", op, typeName)
		if fn, ok := fnMap[typeName]; ok {
			builder.NewFunctionBuilder().WithFunc(fn).Export(funcName)
		} else {
			builder.NewFunctionBuilder().WithFunc(b.makeSeriesArithmeticStub(typeName, op, false)).Export(funcName)
		}
	}
}

// bindSeriesComparisonOps binds series comparison operations
func (b *Bindings) bindSeriesComparisonOps(builder wazero.HostModuleBuilder, typeName string) {
	ops := map[string]map[string]interface{}{
		"gt": b.SeriesCompareGT,
		"lt": b.SeriesCompareLT,
		"ge": b.SeriesCompareGE,
		"le": b.SeriesCompareLE,
		"eq": b.SeriesCompareEQ,
		"ne": b.SeriesCompareNE,
	}

	for op, fnMap := range ops {
		funcName := fmt.Sprintf("series_compare_%s_%s", op, typeName)
		if fn, ok := fnMap[typeName]; ok {
			builder.NewFunctionBuilder().WithFunc(fn).Export(funcName)
		} else {
			builder.NewFunctionBuilder().WithFunc(b.makeSeriesComparisonStub(typeName, op)).Export(funcName)
		}
	}
}

// bindGenericOps binds generic operations
func (b *Bindings) bindGenericOps(builder wazero.HostModuleBuilder) error {
	// Now function
	if b.Now != nil {
		builder.NewFunctionBuilder().WithFunc(b.Now).Export("now")
	} else {
		builder.NewFunctionBuilder().WithFunc(func(ctx context.Context) uint64 {
			panic("now() not implemented")
		}).Export("now")
	}

	// Len function
	if b.Len != nil {
		builder.NewFunctionBuilder().WithFunc(b.Len).Export("len")
	} else {
		builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, handle uint32) uint64 {
			panic("len() not implemented")
		}).Export("len")
	}

	// Panic function
	if b.Panic != nil {
		builder.NewFunctionBuilder().WithFunc(b.Panic).Export("panic")
	} else {
		builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, ptr uint32, len uint32) {
			panic("panic() called")
		}).Export("panic")
	}

	// Math power functions
	if b.MathPowF32 != nil {
		builder.NewFunctionBuilder().WithFunc(b.MathPowF32).Export("math_pow_f32")
	} else {
		builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, base, exp float32) float32 {
			panic("math_pow_f32() not implemented")
		}).Export("math_pow_f32")
	}

	if b.MathPowF64 != nil {
		builder.NewFunctionBuilder().WithFunc(b.MathPowF64).Export("math_pow_f64")
	} else {
		builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, base, exp float64) float64 {
			panic("math_pow_f64() not implemented")
		}).Export("math_pow_f64")
	}

	// Series len function
	if b.SeriesLen != nil {
		builder.NewFunctionBuilder().WithFunc(b.SeriesLen).Export("series_len")
	} else {
		builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, handle uint32) uint64 {
			panic("series_len() not implemented")
		}).Export("series_len")
	}

	// Series slice function
	if b.SeriesSlice != nil {
		builder.NewFunctionBuilder().WithFunc(b.SeriesSlice).Export("series_slice")
	} else {
		builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, handle uint32, start uint32, end uint32) uint32 {
			panic("series_slice() not implemented")
		}).Export("series_slice")
	}

	// String from literal function
	if b.StringFromLiteral != nil {
		builder.NewFunctionBuilder().WithFunc(b.StringFromLiteral).Export("string_from_literal")
	} else {
		builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, ptr uint32, len uint32) uint32 {
			panic("string_from_literal() not implemented")
		}).Export("string_from_literal")
	}

	// String concat function
	if b.StringConcat != nil {
		builder.NewFunctionBuilder().WithFunc(b.StringConcat).Export("string_concat")
	} else {
		builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, s1 uint32, s2 uint32) uint32 {
			panic("string_concat() not implemented")
		}).Export("string_concat")
	}

	// String equal function
	if b.StringEqual != nil {
		builder.NewFunctionBuilder().WithFunc(b.StringEqual).Export("string_equal")
	} else {
		builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, s1 uint32, s2 uint32) uint32 {
			panic("string_equal() not implemented")
		}).Export("string_equal")
	}

	// String len function
	if b.StringLen != nil {
		builder.NewFunctionBuilder().WithFunc(b.StringLen).Export("string_len")
	} else {
		builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, s uint32) uint32 {
			panic("string_len() not implemented")
		}).Export("string_len")
	}

	return nil
}

// Stub generation functions for each operation type

func (b *Bindings) makeChannelReadStub(typeName string) interface{} {
	switch typeName {
	case "i8", "u8":
		return func(ctx context.Context, channelID uint32) uint32 {
			panic(fmt.Sprintf("channel_read_%s not implemented", typeName))
		}
	case "i16", "u16", "i32", "u32":
		return func(ctx context.Context, channelID uint32) uint32 {
			panic(fmt.Sprintf("channel_read_%s not implemented", typeName))
		}
	case "i64", "u64":
		return func(ctx context.Context, channelID uint32) uint64 {
			panic(fmt.Sprintf("channel_read_%s not implemented", typeName))
		}
	case "f32":
		return func(ctx context.Context, channelID uint32) float32 {
			panic(fmt.Sprintf("channel_read_%s not implemented", typeName))
		}
	case "f64":
		return func(ctx context.Context, channelID uint32) float64 {
			panic(fmt.Sprintf("channel_read_%s not implemented", typeName))
		}
	case "string":
		return func(ctx context.Context, channelID uint32) uint32 {
			panic(fmt.Sprintf("channel_read_%s not implemented", typeName))
		}
	default:
		return func(ctx context.Context, channelID uint32) uint32 {
			panic(fmt.Sprintf("channel_read_%s not implemented", typeName))
		}
	}
}

func (b *Bindings) makeChannelWriteStub(typeName string) interface{} {
	switch typeName {
	case "i8", "u8", "i16", "u16", "i32", "u32":
		return func(ctx context.Context, channelID uint32, value uint32) {
			panic(fmt.Sprintf("channel_write_%s not implemented", typeName))
		}
	case "i64", "u64":
		return func(ctx context.Context, channelID uint32, value uint64) {
			panic(fmt.Sprintf("channel_write_%s not implemented", typeName))
		}
	case "f32":
		return func(ctx context.Context, channelID uint32, value float32) {
			panic(fmt.Sprintf("channel_write_%s not implemented", typeName))
		}
	case "f64":
		return func(ctx context.Context, channelID uint32, value float64) {
			panic(fmt.Sprintf("channel_write_%s not implemented", typeName))
		}
	case "string":
		return func(ctx context.Context, channelID uint32, value uint32) {
			panic(fmt.Sprintf("channel_write_%s not implemented", typeName))
		}
	default:
		return func(ctx context.Context, channelID uint32, value uint32) {
			panic(fmt.Sprintf("channel_write_%s not implemented", typeName))
		}
	}
}

func (b *Bindings) makeChannelBlockingReadStub(typeName string) interface{} {
	// Same signatures as channel read
	return b.makeChannelReadStub(typeName)
}

func (b *Bindings) makeStateLoadStub(typeName string) interface{} {
	switch typeName {
	case "i8", "u8", "i16", "u16", "i32", "u32":
		return func(ctx context.Context, taskID uint32, key uint32) uint32 {
			panic(fmt.Sprintf("state_load_%s not implemented", typeName))
		}
	case "i64", "u64":
		return func(ctx context.Context, taskID uint32, key uint32) uint64 {
			panic(fmt.Sprintf("state_load_%s not implemented", typeName))
		}
	case "f32":
		return func(ctx context.Context, taskID uint32, key uint32) float32 {
			panic(fmt.Sprintf("state_load_%s not implemented", typeName))
		}
	case "f64":
		return func(ctx context.Context, taskID uint32, key uint32) float64 {
			panic(fmt.Sprintf("state_load_%s not implemented", typeName))
		}
	case "string":
		return func(ctx context.Context, taskID uint32, key uint32) uint32 {
			panic(fmt.Sprintf("state_load_%s not implemented", typeName))
		}
	default:
		return func(ctx context.Context, taskID uint32, key uint32) uint32 {
			panic(fmt.Sprintf("state_load_%s not implemented", typeName))
		}
	}
}

func (b *Bindings) makeStateStoreStub(typeName string) interface{} {
	switch typeName {
	case "i8", "u8", "i16", "u16", "i32", "u32":
		return func(ctx context.Context, taskID uint32, key uint32, value uint32) {
			panic(fmt.Sprintf("state_store_%s not implemented", typeName))
		}
	case "i64", "u64":
		return func(ctx context.Context, taskID uint32, key uint32, value uint64) {
			panic(fmt.Sprintf("state_store_%s not implemented", typeName))
		}
	case "f32":
		return func(ctx context.Context, taskID uint32, key uint32, value float32) {
			panic(fmt.Sprintf("state_store_%s not implemented", typeName))
		}
	case "f64":
		return func(ctx context.Context, taskID uint32, key uint32, value float64) {
			panic(fmt.Sprintf("state_store_%s not implemented", typeName))
		}
	case "string":
		return func(ctx context.Context, taskID uint32, key uint32, value uint32) {
			panic(fmt.Sprintf("state_store_%s not implemented", typeName))
		}
	default:
		return func(ctx context.Context, taskID uint32, key uint32, value uint32) {
			panic(fmt.Sprintf("state_store_%s not implemented", typeName))
		}
	}
}

func (b *Bindings) makeSeriesCreateEmptyStub(typeName string) interface{} {
	return func(ctx context.Context, length uint32) uint32 {
		panic(fmt.Sprintf("series_create_empty_%s not implemented", typeName))
	}
}

func (b *Bindings) makeSeriesSetElementStub(typeName string) interface{} {
	switch typeName {
	case "i8", "u8", "i16", "u16", "i32", "u32":
		return func(ctx context.Context, handle uint32, index uint32, value uint32) {
			panic(fmt.Sprintf("series_set_element_%s not implemented", typeName))
		}
	case "i64", "u64":
		return func(ctx context.Context, handle uint32, index uint32, value uint64) {
			panic(fmt.Sprintf("series_set_element_%s not implemented", typeName))
		}
	case "f32":
		return func(ctx context.Context, handle uint32, index uint32, value float32) {
			panic(fmt.Sprintf("series_set_element_%s not implemented", typeName))
		}
	case "f64":
		return func(ctx context.Context, handle uint32, index uint32, value float64) {
			panic(fmt.Sprintf("series_set_element_%s not implemented", typeName))
		}
	default:
		return func(ctx context.Context, handle uint32, index uint32, value uint32) {
			panic(fmt.Sprintf("series_set_element_%s not implemented", typeName))
		}
	}
}

func (b *Bindings) makeSeriesIndexStub(typeName string) interface{} {
	switch typeName {
	case "i8", "u8", "i16", "u16", "i32", "u32":
		return func(ctx context.Context, handle uint32, index uint32) uint32 {
			panic(fmt.Sprintf("series_index_%s not implemented", typeName))
		}
	case "i64", "u64":
		return func(ctx context.Context, handle uint32, index uint32) uint64 {
			panic(fmt.Sprintf("series_index_%s not implemented", typeName))
		}
	case "f32":
		return func(ctx context.Context, handle uint32, index uint32) float32 {
			panic(fmt.Sprintf("series_index_%s not implemented", typeName))
		}
	case "f64":
		return func(ctx context.Context, handle uint32, index uint32) float64 {
			panic(fmt.Sprintf("series_index_%s not implemented", typeName))
		}
	default:
		return func(ctx context.Context, handle uint32, index uint32) uint32 {
			panic(fmt.Sprintf("series_index_%s not implemented", typeName))
		}
	}
}

func (b *Bindings) makeSeriesArithmeticStub(typeName string, op string, isElement bool) interface{} {
	prefix := "series_element"
	if !isElement {
		prefix = "series_series"
	}

	if isElement {
		switch typeName {
		case "i8", "u8", "i16", "u16", "i32", "u32":
			return func(ctx context.Context, handle uint32, value uint32) uint32 {
				panic(fmt.Sprintf("%s_%s_%s not implemented", prefix, op, typeName))
			}
		case "i64", "u64":
			return func(ctx context.Context, handle uint32, value uint64) uint32 {
				panic(fmt.Sprintf("%s_%s_%s not implemented", prefix, op, typeName))
			}
		case "f32":
			return func(ctx context.Context, handle uint32, value float32) uint32 {
				panic(fmt.Sprintf("%s_%s_%s not implemented", prefix, op, typeName))
			}
		case "f64":
			return func(ctx context.Context, handle uint32, value float64) uint32 {
				panic(fmt.Sprintf("%s_%s_%s not implemented", prefix, op, typeName))
			}
		default:
			return func(ctx context.Context, handle uint32, value uint32) uint32 {
				panic(fmt.Sprintf("%s_%s_%s not implemented", prefix, op, typeName))
			}
		}
	} else {
		return func(ctx context.Context, handle1 uint32, handle2 uint32) uint32 {
			panic(fmt.Sprintf("%s_%s_%s not implemented", prefix, op, typeName))
		}
	}
}

func (b *Bindings) makeSeriesComparisonStub(typeName string, op string) interface{} {
	return func(ctx context.Context, handle1 uint32, handle2 uint32) uint32 {
		panic(fmt.Sprintf("series_compare_%s_%s not implemented", op, typeName))
	}
}

// Helper function to wrap api.Module calls (if needed)
func wrapModuleFunc(fn interface{}) api.GoModuleFunc {
	return api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
		// This would need proper implementation based on function signature
		panic("not implemented")
	})
}
