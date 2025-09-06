// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mock

import (
	"context"
	"fmt"

	"github.com/synnaxlabs/slate/types"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// Runtime provides a mock implementation of the Slate runtime with configurable function implementations
type Runtime struct {
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
	Now                func(context.Context) uint64
	Len                func(context.Context, uint32) uint64
	Panic              func(context.Context, uint32, uint32)
	MathPowF32         func(context.Context, float32, float32) float32
	MathPowF64         func(context.Context, float64, float64) float64
	SeriesLen          func(context.Context, uint32) uint64
	SeriesSlice        func(context.Context, uint32, uint32, uint32) uint32
	StringFromLiteral  func(context.Context, uint32, uint32) uint32
	StringConcat       func(context.Context, uint32, uint32) uint32
	StringEqual        func(context.Context, uint32, uint32) uint32
	StringLen          func(context.Context, uint32) uint32
}

// New creates a new mock runtime with empty function maps
func New() *Runtime {
	return &Runtime{
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
func (r *Runtime) Bind(ctx context.Context, rt wazero.Runtime) error {
	hostBuilder := rt.NewHostModuleBuilder("env")

	// Bind channel operations for all types
	for _, typ := range types.Numerics {
		if err := r.bindChannelOps(hostBuilder, typ); err != nil {
			return err
		}
	}
	if err := r.bindChannelOps(hostBuilder, types.String{}); err != nil {
		return err
	}

	// Bind state operations for all types
	for _, typ := range types.Numerics {
		if err := r.bindStateOps(hostBuilder, typ); err != nil {
			return err
		}
	}
	if err := r.bindStateOps(hostBuilder, types.String{}); err != nil {
		return err
	}

	// Bind series operations for numeric types
	for _, typ := range types.Numerics {
		if err := r.bindSeriesOps(hostBuilder, typ); err != nil {
			return err
		}
	}

	// Bind generic operations
	if err := r.bindGenericOps(hostBuilder); err != nil {
		return err
	}

	_, err := hostBuilder.Instantiate(ctx)
	return err
}

// bindChannelOps binds channel operations for a specific type
func (r *Runtime) bindChannelOps(builder wazero.HostModuleBuilder, t types.Type) error {
	typeName := t.String()

	// Channel read
	funcName := fmt.Sprintf("channel_read_%s", typeName)
	if fn, ok := r.ChannelRead[typeName]; ok {
		builder.NewFunctionBuilder().WithFunc(fn).Export(funcName)
	} else {
		builder.NewFunctionBuilder().WithFunc(r.makeChannelReadStub(typeName)).Export(funcName)
	}

	// Channel write
	funcName = fmt.Sprintf("channel_write_%s", typeName)
	if fn, ok := r.ChannelWrite[typeName]; ok {
		builder.NewFunctionBuilder().WithFunc(fn).Export(funcName)
	} else {
		builder.NewFunctionBuilder().WithFunc(r.makeChannelWriteStub(typeName)).Export(funcName)
	}

	// Blocking channel read
	funcName = fmt.Sprintf("channel_blocking_read_%s", typeName)
	if fn, ok := r.ChannelBlockingRead[typeName]; ok {
		builder.NewFunctionBuilder().WithFunc(fn).Export(funcName)
	} else {
		builder.NewFunctionBuilder().WithFunc(r.makeChannelBlockingReadStub(typeName)).Export(funcName)
	}

	return nil
}

// bindStateOps binds state operations for a specific type
func (r *Runtime) bindStateOps(builder wazero.HostModuleBuilder, t types.Type) error {
	typeName := t.String()

	// State load
	funcName := fmt.Sprintf("state_load_%s", typeName)
	if fn, ok := r.StateLoad[typeName]; ok {
		builder.NewFunctionBuilder().WithFunc(fn).Export(funcName)
	} else {
		builder.NewFunctionBuilder().WithFunc(r.makeStateLoadStub(typeName)).Export(funcName)
	}

	// State store
	funcName = fmt.Sprintf("state_store_%s", typeName)
	if fn, ok := r.StateStore[typeName]; ok {
		builder.NewFunctionBuilder().WithFunc(fn).Export(funcName)
	} else {
		builder.NewFunctionBuilder().WithFunc(r.makeStateStoreStub(typeName)).Export(funcName)
	}

	return nil
}

// bindSeriesOps binds series operations for a specific type
func (r *Runtime) bindSeriesOps(builder wazero.HostModuleBuilder, t types.Type) error {
	typeName := t.String()

	// Series create empty
	funcName := fmt.Sprintf("series_create_empty_%s", typeName)
	if fn, ok := r.SeriesCreateEmpty[typeName]; ok {
		builder.NewFunctionBuilder().WithFunc(fn).Export(funcName)
	} else {
		builder.NewFunctionBuilder().WithFunc(r.makeSeriesCreateEmptyStub(typeName)).Export(funcName)
	}

	// Series set element
	funcName = fmt.Sprintf("series_set_element_%s", typeName)
	if fn, ok := r.SeriesSetElement[typeName]; ok {
		builder.NewFunctionBuilder().WithFunc(fn).Export(funcName)
	} else {
		builder.NewFunctionBuilder().WithFunc(r.makeSeriesSetElementStub(typeName)).Export(funcName)
	}

	// Series index
	funcName = fmt.Sprintf("series_index_%s", typeName)
	if fn, ok := r.SeriesIndex[typeName]; ok {
		builder.NewFunctionBuilder().WithFunc(fn).Export(funcName)
	} else {
		builder.NewFunctionBuilder().WithFunc(r.makeSeriesIndexStub(typeName)).Export(funcName)
	}

	// Series arithmetic operations
	r.bindSeriesArithmeticOps(builder, typeName)

	// Series comparison operations
	r.bindSeriesComparisonOps(builder, typeName)

	return nil
}

// bindSeriesArithmeticOps binds series arithmetic operations
func (r *Runtime) bindSeriesArithmeticOps(builder wazero.HostModuleBuilder, typeName string) {
	// Element operations
	ops := map[string]map[string]interface{}{
		"add": r.SeriesElementAdd,
		"sub": r.SeriesElementSub,
		"mul": r.SeriesElementMul,
		"div": r.SeriesElementDiv,
	}

	for op, fnMap := range ops {
		funcName := fmt.Sprintf("series_element_%s_%s", op, typeName)
		if fn, ok := fnMap[typeName]; ok {
			builder.NewFunctionBuilder().WithFunc(fn).Export(funcName)
		} else {
			builder.NewFunctionBuilder().WithFunc(r.makeSeriesArithmeticStub(typeName, op, true)).Export(funcName)
		}
	}

	// Series-series operations
	ops = map[string]map[string]interface{}{
		"add": r.SeriesSeriesAdd,
		"sub": r.SeriesSeriesSub,
		"mul": r.SeriesSeriesMul,
		"div": r.SeriesSeriesDiv,
	}

	for op, fnMap := range ops {
		funcName := fmt.Sprintf("series_series_%s_%s", op, typeName)
		if fn, ok := fnMap[typeName]; ok {
			builder.NewFunctionBuilder().WithFunc(fn).Export(funcName)
		} else {
			builder.NewFunctionBuilder().WithFunc(r.makeSeriesArithmeticStub(typeName, op, false)).Export(funcName)
		}
	}
}

// bindSeriesComparisonOps binds series comparison operations
func (r *Runtime) bindSeriesComparisonOps(builder wazero.HostModuleBuilder, typeName string) {
	ops := map[string]map[string]interface{}{
		"gt": r.SeriesCompareGT,
		"lt": r.SeriesCompareLT,
		"ge": r.SeriesCompareGE,
		"le": r.SeriesCompareLE,
		"eq": r.SeriesCompareEQ,
		"ne": r.SeriesCompareNE,
	}

	for op, fnMap := range ops {
		funcName := fmt.Sprintf("series_compare_%s_%s", op, typeName)
		if fn, ok := fnMap[typeName]; ok {
			builder.NewFunctionBuilder().WithFunc(fn).Export(funcName)
		} else {
			builder.NewFunctionBuilder().WithFunc(r.makeSeriesComparisonStub(typeName, op)).Export(funcName)
		}
	}
}

// bindGenericOps binds generic operations
func (r *Runtime) bindGenericOps(builder wazero.HostModuleBuilder) error {
	// Now function
	if r.Now != nil {
		builder.NewFunctionBuilder().WithFunc(r.Now).Export("now")
	} else {
		builder.NewFunctionBuilder().WithFunc(func(ctx context.Context) uint64 {
			panic("now() not implemented")
		}).Export("now")
	}

	// Len function
	if r.Len != nil {
		builder.NewFunctionBuilder().WithFunc(r.Len).Export("len")
	} else {
		builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, handle uint32) uint64 {
			panic("len() not implemented")
		}).Export("len")
	}

	// Panic function
	if r.Panic != nil {
		builder.NewFunctionBuilder().WithFunc(r.Panic).Export("panic")
	} else {
		builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, ptr uint32, len uint32) {
			panic("panic() called")
		}).Export("panic")
	}

	// Math power functions
	if r.MathPowF32 != nil {
		builder.NewFunctionBuilder().WithFunc(r.MathPowF32).Export("math_pow_f32")
	} else {
		builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, base, exp float32) float32 {
			panic("math_pow_f32() not implemented")
		}).Export("math_pow_f32")
	}

	if r.MathPowF64 != nil {
		builder.NewFunctionBuilder().WithFunc(r.MathPowF64).Export("math_pow_f64")
	} else {
		builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, base, exp float64) float64 {
			panic("math_pow_f64() not implemented")
		}).Export("math_pow_f64")
	}

	// Series len function
	if r.SeriesLen != nil {
		builder.NewFunctionBuilder().WithFunc(r.SeriesLen).Export("series_len")
	} else {
		builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, handle uint32) uint64 {
			panic("series_len() not implemented")
		}).Export("series_len")
	}

	// Series slice function
	if r.SeriesSlice != nil {
		builder.NewFunctionBuilder().WithFunc(r.SeriesSlice).Export("series_slice")
	} else {
		builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, handle uint32, start uint32, end uint32) uint32 {
			panic("series_slice() not implemented")
		}).Export("series_slice")
	}

	// String from literal function
	if r.StringFromLiteral != nil {
		builder.NewFunctionBuilder().WithFunc(r.StringFromLiteral).Export("string_from_literal")
	} else {
		builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, ptr uint32, len uint32) uint32 {
			panic("string_from_literal() not implemented")
		}).Export("string_from_literal")
	}

	// String concat function
	if r.StringConcat != nil {
		builder.NewFunctionBuilder().WithFunc(r.StringConcat).Export("string_concat")
	} else {
		builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, s1 uint32, s2 uint32) uint32 {
			panic("string_concat() not implemented")
		}).Export("string_concat")
	}

	// String equal function
	if r.StringEqual != nil {
		builder.NewFunctionBuilder().WithFunc(r.StringEqual).Export("string_equal")
	} else {
		builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, s1 uint32, s2 uint32) uint32 {
			panic("string_equal() not implemented")
		}).Export("string_equal")
	}

	// String len function
	if r.StringLen != nil {
		builder.NewFunctionBuilder().WithFunc(r.StringLen).Export("string_len")
	} else {
		builder.NewFunctionBuilder().WithFunc(func(ctx context.Context, s uint32) uint32 {
			panic("string_len() not implemented")
		}).Export("string_len")
	}

	return nil
}

// Stub generation functions for each operation type

func (r *Runtime) makeChannelReadStub(typeName string) interface{} {
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

func (r *Runtime) makeChannelWriteStub(typeName string) interface{} {
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

func (r *Runtime) makeChannelBlockingReadStub(typeName string) interface{} {
	// Same signatures as channel read
	return r.makeChannelReadStub(typeName)
}

func (r *Runtime) makeStateLoadStub(typeName string) interface{} {
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

func (r *Runtime) makeStateStoreStub(typeName string) interface{} {
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

func (r *Runtime) makeSeriesCreateEmptyStub(typeName string) interface{} {
	return func(ctx context.Context, length uint32) uint32 {
		panic(fmt.Sprintf("series_create_empty_%s not implemented", typeName))
	}
}

func (r *Runtime) makeSeriesSetElementStub(typeName string) interface{} {
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

func (r *Runtime) makeSeriesIndexStub(typeName string) interface{} {
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

func (r *Runtime) makeSeriesArithmeticStub(typeName string, op string, isElement bool) interface{} {
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

func (r *Runtime) makeSeriesComparisonStub(typeName string, op string) interface{} {
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
