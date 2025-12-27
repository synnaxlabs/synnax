// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package bindings_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/runtime/wasm/bindings"
)

// typeOps provides type-specific operations for testing.
type typeOps[T any] struct {
	name        string
	createEmpty func(ctx context.Context, length uint32) uint32
	setElement  func(ctx context.Context, handle, index uint32, value T) uint32
	index       func(ctx context.Context, handle, index uint32) T
	elementAdd  func(ctx context.Context, handle uint32, value T) uint32
	elementSub  func(ctx context.Context, handle uint32, value T) uint32
	elementMul  func(ctx context.Context, handle uint32, value T) uint32
	elementDiv  func(ctx context.Context, handle uint32, value T) uint32
	elementMod  func(ctx context.Context, handle uint32, value T) uint32
	elementRSub func(ctx context.Context, value T, handle uint32) uint32
	elementRDiv func(ctx context.Context, value T, handle uint32) uint32
	seriesAdd   func(ctx context.Context, a, b uint32) uint32
	seriesSub   func(ctx context.Context, a, b uint32) uint32
	seriesMul   func(ctx context.Context, a, b uint32) uint32
	seriesDiv   func(ctx context.Context, a, b uint32) uint32
	seriesMod   func(ctx context.Context, a, b uint32) uint32
	compareGT   func(ctx context.Context, a, b uint32) uint32
	compareLT   func(ctx context.Context, a, b uint32) uint32
	compareGE   func(ctx context.Context, a, b uint32) uint32
	compareLE   func(ctx context.Context, a, b uint32) uint32
	compareEQ   func(ctx context.Context, a, b uint32) uint32
	compareNE   func(ctx context.Context, a, b uint32) uint32
	scalarGT    func(ctx context.Context, handle uint32, value T) uint32
	scalarLT    func(ctx context.Context, handle uint32, value T) uint32
	scalarGE    func(ctx context.Context, handle uint32, value T) uint32
	scalarLE    func(ctx context.Context, handle uint32, value T) uint32
	scalarEQ    func(ctx context.Context, handle uint32, value T) uint32
	scalarNE    func(ctx context.Context, handle uint32, value T) uint32
	stateLoad   func(ctx context.Context, funcID, varID, initHandle uint32) uint32
	stateStore  func(ctx context.Context, funcID, varID, handle uint32)
	v1, v2, v3  T
	add12       T
	sub12       T
	mul12       T
	div21       T
	mod21       T
	scalar      T
	addScalar1  T
}

func buildU8Ops(rt *bindings.Runtime) typeOps[uint8] {
	return typeOps[uint8]{
		name: "U8", createEmpty: rt.SeriesCreateEmptyU8, setElement: rt.SeriesSetElementU8, index: rt.SeriesIndexU8,
		elementAdd: rt.SeriesElementAddU8, elementSub: rt.SeriesElementSubU8, elementMul: rt.SeriesElementMulU8,
		elementDiv: rt.SeriesElementDivU8, elementMod: rt.SeriesElementModU8,
		elementRSub: rt.SeriesElementRSubU8, elementRDiv: rt.SeriesElementRDivU8,
		seriesAdd: rt.SeriesSeriesAddU8, seriesSub: rt.SeriesSeriesSubU8, seriesMul: rt.SeriesSeriesMulU8,
		seriesDiv: rt.SeriesSeriesDivU8, seriesMod: rt.SeriesSeriesModU8,
		compareGT: rt.SeriesCompareGTU8, compareLT: rt.SeriesCompareLTU8, compareGE: rt.SeriesCompareGEU8,
		compareLE: rt.SeriesCompareLEU8, compareEQ: rt.SeriesCompareEQU8, compareNE: rt.SeriesCompareNEU8,
		scalarGT: rt.SeriesCompareGTScalarU8, scalarLT: rt.SeriesCompareLTScalarU8, scalarGE: rt.SeriesCompareGEScalarU8,
		scalarLE: rt.SeriesCompareLEScalarU8, scalarEQ: rt.SeriesCompareEQScalarU8, scalarNE: rt.SeriesCompareNEScalarU8,
		stateLoad: rt.StateLoadSeriesU8, stateStore: rt.StateStoreSeriesU8,
		v1: 10, v2: 20, v3: 30, add12: 30, sub12: 246, mul12: 200, div21: 2, mod21: 0, scalar: 5, addScalar1: 15,
	}
}

func buildU16Ops(rt *bindings.Runtime) typeOps[uint16] {
	return typeOps[uint16]{
		name: "U16", createEmpty: rt.SeriesCreateEmptyU16, setElement: rt.SeriesSetElementU16, index: rt.SeriesIndexU16,
		elementAdd: rt.SeriesElementAddU16, elementSub: rt.SeriesElementSubU16, elementMul: rt.SeriesElementMulU16,
		elementDiv: rt.SeriesElementDivU16, elementMod: rt.SeriesElementModU16,
		elementRSub: rt.SeriesElementRSubU16, elementRDiv: rt.SeriesElementRDivU16,
		seriesAdd: rt.SeriesSeriesAddU16, seriesSub: rt.SeriesSeriesSubU16, seriesMul: rt.SeriesSeriesMulU16,
		seriesDiv: rt.SeriesSeriesDivU16, seriesMod: rt.SeriesSeriesModU16,
		compareGT: rt.SeriesCompareGTU16, compareLT: rt.SeriesCompareLTU16, compareGE: rt.SeriesCompareGEU16,
		compareLE: rt.SeriesCompareLEU16, compareEQ: rt.SeriesCompareEQU16, compareNE: rt.SeriesCompareNEU16,
		scalarGT: rt.SeriesCompareGTScalarU16, scalarLT: rt.SeriesCompareLTScalarU16, scalarGE: rt.SeriesCompareGEScalarU16,
		scalarLE: rt.SeriesCompareLEScalarU16, scalarEQ: rt.SeriesCompareEQScalarU16, scalarNE: rt.SeriesCompareNEScalarU16,
		stateLoad: rt.StateLoadSeriesU16, stateStore: rt.StateStoreSeriesU16,
		v1: 10, v2: 20, v3: 30, add12: 30, sub12: 65526, mul12: 200, div21: 2, mod21: 0, scalar: 5, addScalar1: 15,
	}
}

func buildU32Ops(rt *bindings.Runtime) typeOps[uint32] {
	return typeOps[uint32]{
		name: "U32", createEmpty: rt.SeriesCreateEmptyU32, setElement: rt.SeriesSetElementU32, index: rt.SeriesIndexU32,
		elementAdd: rt.SeriesElementAddU32, elementSub: rt.SeriesElementSubU32, elementMul: rt.SeriesElementMulU32,
		elementDiv: rt.SeriesElementDivU32, elementMod: rt.SeriesElementModU32,
		elementRSub: rt.SeriesElementRSubU32, elementRDiv: rt.SeriesElementRDivU32,
		seriesAdd: rt.SeriesSeriesAddU32, seriesSub: rt.SeriesSeriesSubU32, seriesMul: rt.SeriesSeriesMulU32,
		seriesDiv: rt.SeriesSeriesDivU32, seriesMod: rt.SeriesSeriesModU32,
		compareGT: rt.SeriesCompareGTU32, compareLT: rt.SeriesCompareLTU32, compareGE: rt.SeriesCompareGEU32,
		compareLE: rt.SeriesCompareLEU32, compareEQ: rt.SeriesCompareEQU32, compareNE: rt.SeriesCompareNEU32,
		scalarGT: rt.SeriesCompareGTScalarU32, scalarLT: rt.SeriesCompareLTScalarU32, scalarGE: rt.SeriesCompareGEScalarU32,
		scalarLE: rt.SeriesCompareLEScalarU32, scalarEQ: rt.SeriesCompareEQScalarU32, scalarNE: rt.SeriesCompareNEScalarU32,
		stateLoad: rt.StateLoadSeriesU32, stateStore: rt.StateStoreSeriesU32,
		v1: 10, v2: 20, v3: 30, add12: 30, sub12: 4294967286, mul12: 200, div21: 2, mod21: 0, scalar: 5, addScalar1: 15,
	}
}

func buildU64Ops(rt *bindings.Runtime) typeOps[uint64] {
	return typeOps[uint64]{
		name: "U64", createEmpty: rt.SeriesCreateEmptyU64, setElement: rt.SeriesSetElementU64, index: rt.SeriesIndexU64,
		elementAdd: rt.SeriesElementAddU64, elementSub: rt.SeriesElementSubU64, elementMul: rt.SeriesElementMulU64,
		elementDiv: rt.SeriesElementDivU64, elementMod: rt.SeriesElementModU64,
		elementRSub: rt.SeriesElementRSubU64, elementRDiv: rt.SeriesElementRDivU64,
		seriesAdd: rt.SeriesSeriesAddU64, seriesSub: rt.SeriesSeriesSubU64, seriesMul: rt.SeriesSeriesMulU64,
		seriesDiv: rt.SeriesSeriesDivU64, seriesMod: rt.SeriesSeriesModU64,
		compareGT: rt.SeriesCompareGTU64, compareLT: rt.SeriesCompareLTU64, compareGE: rt.SeriesCompareGEU64,
		compareLE: rt.SeriesCompareLEU64, compareEQ: rt.SeriesCompareEQU64, compareNE: rt.SeriesCompareNEU64,
		scalarGT: rt.SeriesCompareGTScalarU64, scalarLT: rt.SeriesCompareLTScalarU64, scalarGE: rt.SeriesCompareGEScalarU64,
		scalarLE: rt.SeriesCompareLEScalarU64, scalarEQ: rt.SeriesCompareEQScalarU64, scalarNE: rt.SeriesCompareNEScalarU64,
		stateLoad: rt.StateLoadSeriesU64, stateStore: rt.StateStoreSeriesU64,
		v1: 10, v2: 20, v3: 30, add12: 30, sub12: 18446744073709551606, mul12: 200, div21: 2, mod21: 0, scalar: 5, addScalar1: 15,
	}
}

func buildI8Ops(rt *bindings.Runtime) typeOps[int8] {
	return typeOps[int8]{
		name: "I8", createEmpty: rt.SeriesCreateEmptyI8, setElement: rt.SeriesSetElementI8, index: rt.SeriesIndexI8,
		elementAdd: rt.SeriesElementAddI8, elementSub: rt.SeriesElementSubI8, elementMul: rt.SeriesElementMulI8,
		elementDiv: rt.SeriesElementDivI8, elementMod: rt.SeriesElementModI8,
		elementRSub: rt.SeriesElementRSubI8, elementRDiv: rt.SeriesElementRDivI8,
		seriesAdd: rt.SeriesSeriesAddI8, seriesSub: rt.SeriesSeriesSubI8, seriesMul: rt.SeriesSeriesMulI8,
		seriesDiv: rt.SeriesSeriesDivI8, seriesMod: rt.SeriesSeriesModI8,
		compareGT: rt.SeriesCompareGTI8, compareLT: rt.SeriesCompareLTI8, compareGE: rt.SeriesCompareGEI8,
		compareLE: rt.SeriesCompareLEI8, compareEQ: rt.SeriesCompareEQI8, compareNE: rt.SeriesCompareNEI8,
		scalarGT: rt.SeriesCompareGTScalarI8, scalarLT: rt.SeriesCompareLTScalarI8, scalarGE: rt.SeriesCompareGEScalarI8,
		scalarLE: rt.SeriesCompareLEScalarI8, scalarEQ: rt.SeriesCompareEQScalarI8, scalarNE: rt.SeriesCompareNEScalarI8,
		stateLoad: rt.StateLoadSeriesI8, stateStore: rt.StateStoreSeriesI8,
		// Use small values to avoid int8 overflow (max 127)
		v1: 3, v2: 6, v3: 9, add12: 9, sub12: -3, mul12: 18, div21: 2, mod21: 0, scalar: 2, addScalar1: 5,
	}
}

func buildI16Ops(rt *bindings.Runtime) typeOps[int16] {
	return typeOps[int16]{
		name: "I16", createEmpty: rt.SeriesCreateEmptyI16, setElement: rt.SeriesSetElementI16, index: rt.SeriesIndexI16,
		elementAdd: rt.SeriesElementAddI16, elementSub: rt.SeriesElementSubI16, elementMul: rt.SeriesElementMulI16,
		elementDiv: rt.SeriesElementDivI16, elementMod: rt.SeriesElementModI16,
		elementRSub: rt.SeriesElementRSubI16, elementRDiv: rt.SeriesElementRDivI16,
		seriesAdd: rt.SeriesSeriesAddI16, seriesSub: rt.SeriesSeriesSubI16, seriesMul: rt.SeriesSeriesMulI16,
		seriesDiv: rt.SeriesSeriesDivI16, seriesMod: rt.SeriesSeriesModI16,
		compareGT: rt.SeriesCompareGTI16, compareLT: rt.SeriesCompareLTI16, compareGE: rt.SeriesCompareGEI16,
		compareLE: rt.SeriesCompareLEI16, compareEQ: rt.SeriesCompareEQI16, compareNE: rt.SeriesCompareNEI16,
		scalarGT: rt.SeriesCompareGTScalarI16, scalarLT: rt.SeriesCompareLTScalarI16, scalarGE: rt.SeriesCompareGEScalarI16,
		scalarLE: rt.SeriesCompareLEScalarI16, scalarEQ: rt.SeriesCompareEQScalarI16, scalarNE: rt.SeriesCompareNEScalarI16,
		stateLoad: rt.StateLoadSeriesI16, stateStore: rt.StateStoreSeriesI16,
		v1: 10, v2: 20, v3: 30, add12: 30, sub12: -10, mul12: 200, div21: 2, mod21: 0, scalar: 5, addScalar1: 15,
	}
}

func buildI32Ops(rt *bindings.Runtime) typeOps[int32] {
	return typeOps[int32]{
		name: "I32", createEmpty: rt.SeriesCreateEmptyI32, setElement: rt.SeriesSetElementI32, index: rt.SeriesIndexI32,
		elementAdd: rt.SeriesElementAddI32, elementSub: rt.SeriesElementSubI32, elementMul: rt.SeriesElementMulI32,
		elementDiv: rt.SeriesElementDivI32, elementMod: rt.SeriesElementModI32,
		elementRSub: rt.SeriesElementRSubI32, elementRDiv: rt.SeriesElementRDivI32,
		seriesAdd: rt.SeriesSeriesAddI32, seriesSub: rt.SeriesSeriesSubI32, seriesMul: rt.SeriesSeriesMulI32,
		seriesDiv: rt.SeriesSeriesDivI32, seriesMod: rt.SeriesSeriesModI32,
		compareGT: rt.SeriesCompareGTI32, compareLT: rt.SeriesCompareLTI32, compareGE: rt.SeriesCompareGEI32,
		compareLE: rt.SeriesCompareLEI32, compareEQ: rt.SeriesCompareEQI32, compareNE: rt.SeriesCompareNEI32,
		scalarGT: rt.SeriesCompareGTScalarI32, scalarLT: rt.SeriesCompareLTScalarI32, scalarGE: rt.SeriesCompareGEScalarI32,
		scalarLE: rt.SeriesCompareLEScalarI32, scalarEQ: rt.SeriesCompareEQScalarI32, scalarNE: rt.SeriesCompareNEScalarI32,
		stateLoad: rt.StateLoadSeriesI32, stateStore: rt.StateStoreSeriesI32,
		v1: 10, v2: 20, v3: 30, add12: 30, sub12: -10, mul12: 200, div21: 2, mod21: 0, scalar: 5, addScalar1: 15,
	}
}

func buildI64Ops(rt *bindings.Runtime) typeOps[int64] {
	return typeOps[int64]{
		name: "I64", createEmpty: rt.SeriesCreateEmptyI64, setElement: rt.SeriesSetElementI64, index: rt.SeriesIndexI64,
		elementAdd: rt.SeriesElementAddI64, elementSub: rt.SeriesElementSubI64, elementMul: rt.SeriesElementMulI64,
		elementDiv: rt.SeriesElementDivI64, elementMod: rt.SeriesElementModI64,
		elementRSub: rt.SeriesElementRSubI64, elementRDiv: rt.SeriesElementRDivI64,
		seriesAdd: rt.SeriesSeriesAddI64, seriesSub: rt.SeriesSeriesSubI64, seriesMul: rt.SeriesSeriesMulI64,
		seriesDiv: rt.SeriesSeriesDivI64, seriesMod: rt.SeriesSeriesModI64,
		compareGT: rt.SeriesCompareGTI64, compareLT: rt.SeriesCompareLTI64, compareGE: rt.SeriesCompareGEI64,
		compareLE: rt.SeriesCompareLEI64, compareEQ: rt.SeriesCompareEQI64, compareNE: rt.SeriesCompareNEI64,
		scalarGT: rt.SeriesCompareGTScalarI64, scalarLT: rt.SeriesCompareLTScalarI64, scalarGE: rt.SeriesCompareGEScalarI64,
		scalarLE: rt.SeriesCompareLEScalarI64, scalarEQ: rt.SeriesCompareEQScalarI64, scalarNE: rt.SeriesCompareNEScalarI64,
		stateLoad: rt.StateLoadSeriesI64, stateStore: rt.StateStoreSeriesI64,
		v1: 10, v2: 20, v3: 30, add12: 30, sub12: -10, mul12: 200, div21: 2, mod21: 0, scalar: 5, addScalar1: 15,
	}
}

func buildF32Ops(rt *bindings.Runtime) typeOps[float32] {
	return typeOps[float32]{
		name: "F32", createEmpty: rt.SeriesCreateEmptyF32, setElement: rt.SeriesSetElementF32, index: rt.SeriesIndexF32,
		elementAdd: rt.SeriesElementAddF32, elementSub: rt.SeriesElementSubF32, elementMul: rt.SeriesElementMulF32,
		elementDiv: rt.SeriesElementDivF32, elementMod: rt.SeriesElementModF32,
		elementRSub: rt.SeriesElementRSubF32, elementRDiv: rt.SeriesElementRDivF32,
		seriesAdd: rt.SeriesSeriesAddF32, seriesSub: rt.SeriesSeriesSubF32, seriesMul: rt.SeriesSeriesMulF32,
		seriesDiv: rt.SeriesSeriesDivF32, seriesMod: rt.SeriesSeriesModF32,
		compareGT: rt.SeriesCompareGTF32, compareLT: rt.SeriesCompareLTF32, compareGE: rt.SeriesCompareGEF32,
		compareLE: rt.SeriesCompareLEF32, compareEQ: rt.SeriesCompareEQF32, compareNE: rt.SeriesCompareNEF32,
		scalarGT: rt.SeriesCompareGTScalarF32, scalarLT: rt.SeriesCompareLTScalarF32, scalarGE: rt.SeriesCompareGEScalarF32,
		scalarLE: rt.SeriesCompareLEScalarF32, scalarEQ: rt.SeriesCompareEQScalarF32, scalarNE: rt.SeriesCompareNEScalarF32,
		stateLoad: rt.StateLoadSeriesF32, stateStore: rt.StateStoreSeriesF32,
		v1: 10.0, v2: 20.0, v3: 30.0, add12: 30.0, sub12: -10.0, mul12: 200.0, div21: 2.0, mod21: 0, scalar: 5.0, addScalar1: 15.0,
	}
}

func buildF64Ops(rt *bindings.Runtime) typeOps[float64] {
	return typeOps[float64]{
		name: "F64", createEmpty: rt.SeriesCreateEmptyF64, setElement: rt.SeriesSetElementF64, index: rt.SeriesIndexF64,
		elementAdd: rt.SeriesElementAddF64, elementSub: rt.SeriesElementSubF64, elementMul: rt.SeriesElementMulF64,
		elementDiv: rt.SeriesElementDivF64, elementMod: rt.SeriesElementModF64,
		elementRSub: rt.SeriesElementRSubF64, elementRDiv: rt.SeriesElementRDivF64,
		seriesAdd: rt.SeriesSeriesAddF64, seriesSub: rt.SeriesSeriesSubF64, seriesMul: rt.SeriesSeriesMulF64,
		seriesDiv: rt.SeriesSeriesDivF64, seriesMod: rt.SeriesSeriesModF64,
		compareGT: rt.SeriesCompareGTF64, compareLT: rt.SeriesCompareLTF64, compareGE: rt.SeriesCompareGEF64,
		compareLE: rt.SeriesCompareLEF64, compareEQ: rt.SeriesCompareEQF64, compareNE: rt.SeriesCompareNEF64,
		scalarGT: rt.SeriesCompareGTScalarF64, scalarLT: rt.SeriesCompareLTScalarF64, scalarGE: rt.SeriesCompareGEScalarF64,
		scalarLE: rt.SeriesCompareLEScalarF64, scalarEQ: rt.SeriesCompareEQScalarF64, scalarNE: rt.SeriesCompareNEScalarF64,
		stateLoad: rt.StateLoadSeriesF64, stateStore: rt.StateStoreSeriesF64,
		v1: 10.0, v2: 20.0, v3: 30.0, add12: 30.0, sub12: -10.0, mul12: 200.0, div21: 2.0, mod21: 0, scalar: 5.0, addScalar1: 15.0,
	}
}

// testAllOps runs all operations for a type within a single It block
func testAllOps[T comparable](ops typeOps[T], rt *bindings.Runtime, ctx context.Context) {
	// Creation and Access
	h := ops.createEmpty(ctx, 3)
	Expect(h).ToNot(Equal(uint32(0)))
	Expect(rt.SeriesLen(ctx, h)).To(Equal(uint64(3)))
	ops.setElement(ctx, h, 0, ops.v1)
	ops.setElement(ctx, h, 1, ops.v2)
	ops.setElement(ctx, h, 2, ops.v3)
	Expect(ops.index(ctx, h, 0)).To(Equal(ops.v1))
	Expect(ops.index(ctx, h, 1)).To(Equal(ops.v2))
	Expect(ops.index(ctx, h, 2)).To(Equal(ops.v3))

	// Element scalar operations
	h1 := ops.createEmpty(ctx, 1)
	ops.setElement(ctx, h1, 0, ops.v1)
	Expect(ops.index(ctx, ops.elementAdd(ctx, h1, ops.scalar), 0)).To(Equal(ops.addScalar1))
	Expect(ops.elementSub(ctx, h1, ops.scalar)).ToNot(Equal(uint32(0)))
	Expect(ops.elementMul(ctx, h1, ops.scalar)).ToNot(Equal(uint32(0)))
	Expect(ops.elementDiv(ctx, h1, ops.scalar)).ToNot(Equal(uint32(0)))
	Expect(ops.elementMod(ctx, h1, ops.scalar)).ToNot(Equal(uint32(0)))
	Expect(ops.elementRSub(ctx, ops.scalar, h1)).ToNot(Equal(uint32(0)))
	Expect(ops.elementRDiv(ctx, ops.scalar, h1)).ToNot(Equal(uint32(0)))

	// Series-series operations
	h2 := ops.createEmpty(ctx, 1)
	ops.setElement(ctx, h2, 0, ops.v2)
	Expect(ops.index(ctx, ops.seriesAdd(ctx, h1, h2), 0)).To(Equal(ops.add12))
	Expect(ops.index(ctx, ops.seriesSub(ctx, h1, h2), 0)).To(Equal(ops.sub12))
	Expect(ops.index(ctx, ops.seriesMul(ctx, h1, h2), 0)).To(Equal(ops.mul12))
	Expect(ops.index(ctx, ops.seriesDiv(ctx, h2, h1), 0)).To(Equal(ops.div21))
	Expect(ops.index(ctx, ops.seriesMod(ctx, h2, h1), 0)).To(Equal(ops.mod21))

	// Comparison operations
	h3 := ops.createEmpty(ctx, 1)
	ops.setElement(ctx, h3, 0, ops.v1) // same as h1
	Expect(rt.SeriesIndexU8(ctx, ops.compareGT(ctx, h2, h1), 0)).To(Equal(uint8(1)))
	Expect(rt.SeriesIndexU8(ctx, ops.compareGT(ctx, h1, h2), 0)).To(Equal(uint8(0)))
	Expect(rt.SeriesIndexU8(ctx, ops.compareLT(ctx, h1, h2), 0)).To(Equal(uint8(1)))
	Expect(rt.SeriesIndexU8(ctx, ops.compareGE(ctx, h1, h3), 0)).To(Equal(uint8(1)))
	Expect(rt.SeriesIndexU8(ctx, ops.compareLE(ctx, h1, h3), 0)).To(Equal(uint8(1)))
	Expect(rt.SeriesIndexU8(ctx, ops.compareEQ(ctx, h1, h3), 0)).To(Equal(uint8(1)))
	Expect(rt.SeriesIndexU8(ctx, ops.compareEQ(ctx, h1, h2), 0)).To(Equal(uint8(0)))
	Expect(rt.SeriesIndexU8(ctx, ops.compareNE(ctx, h1, h2), 0)).To(Equal(uint8(1)))

	// Scalar comparison operations
	Expect(rt.SeriesIndexU8(ctx, ops.scalarGT(ctx, h2, ops.v1), 0)).To(Equal(uint8(1)))
	Expect(rt.SeriesIndexU8(ctx, ops.scalarLT(ctx, h2, ops.v3), 0)).To(Equal(uint8(1)))
	Expect(rt.SeriesIndexU8(ctx, ops.scalarGE(ctx, h2, ops.v2), 0)).To(Equal(uint8(1)))
	Expect(rt.SeriesIndexU8(ctx, ops.scalarLE(ctx, h2, ops.v2), 0)).To(Equal(uint8(1)))
	Expect(rt.SeriesIndexU8(ctx, ops.scalarEQ(ctx, h2, ops.v2), 0)).To(Equal(uint8(1)))
	Expect(rt.SeriesIndexU8(ctx, ops.scalarNE(ctx, h2, ops.v1), 0)).To(Equal(uint8(1)))

	// State operations
	hs := ops.createEmpty(ctx, 2)
	ops.setElement(ctx, hs, 0, ops.v1)
	ops.setElement(ctx, hs, 1, ops.v2)
	loaded1 := ops.stateLoad(ctx, 1, 1, hs)
	Expect(loaded1).To(Equal(hs))
	ops.stateStore(ctx, 1, 1, hs)
	hInit := ops.createEmpty(ctx, 1)
	ops.setElement(ctx, hInit, 0, ops.v3)
	loaded2 := ops.stateLoad(ctx, 1, 1, hInit)
	Expect(loaded2).ToNot(Equal(hInit))
	Expect(rt.SeriesLen(ctx, loaded2)).To(Equal(uint64(2)))
}

var _ = Describe("Runtime Series Operations", func() {
	var rt *bindings.Runtime
	var ctx context.Context

	BeforeEach(func() {
		rt = bindings.NewRuntime(nil, nil)
		ctx = context.Background()
	})

	// Test all 10 types - each type tests all 28 operations
	Describe("U8", func() {
		It("Should support all operations", func() { testAllOps(buildU8Ops(rt), rt, ctx) })
	})
	Describe("U16", func() {
		It("Should support all operations", func() { testAllOps(buildU16Ops(rt), rt, ctx) })
	})
	Describe("U32", func() {
		It("Should support all operations", func() { testAllOps(buildU32Ops(rt), rt, ctx) })
	})
	Describe("U64", func() {
		It("Should support all operations", func() { testAllOps(buildU64Ops(rt), rt, ctx) })
	})
	Describe("I8", func() {
		It("Should support all operations", func() { testAllOps(buildI8Ops(rt), rt, ctx) })
	})
	Describe("I16", func() {
		It("Should support all operations", func() { testAllOps(buildI16Ops(rt), rt, ctx) })
	})
	Describe("I32", func() {
		It("Should support all operations", func() { testAllOps(buildI32Ops(rt), rt, ctx) })
	})
	Describe("I64", func() {
		It("Should support all operations", func() { testAllOps(buildI64Ops(rt), rt, ctx) })
	})
	Describe("F32", func() {
		It("Should support all operations", func() { testAllOps(buildF32Ops(rt), rt, ctx) })
	})
	Describe("F64", func() {
		It("Should support all operations", func() { testAllOps(buildF64Ops(rt), rt, ctx) })
	})

	Describe("Unary Operations", func() {
		Describe("Negate", func() {
			It("Should negate I8", func() {
				h := rt.SeriesCreateEmptyI8(ctx, 3)
				rt.SeriesSetElementI8(ctx, h, 0, 5)
				rt.SeriesSetElementI8(ctx, h, 1, -3)
				rt.SeriesSetElementI8(ctx, h, 2, 0)
				result := rt.SeriesNegateI8(ctx, h)
				Expect(rt.SeriesIndexI8(ctx, result, 0)).To(Equal(int8(-5)))
				Expect(rt.SeriesIndexI8(ctx, result, 1)).To(Equal(int8(3)))
				Expect(rt.SeriesIndexI8(ctx, result, 2)).To(Equal(int8(0)))
			})
			It("Should negate I16", func() {
				h := rt.SeriesCreateEmptyI16(ctx, 2)
				rt.SeriesSetElementI16(ctx, h, 0, 1000)
				rt.SeriesSetElementI16(ctx, h, 1, -500)
				result := rt.SeriesNegateI16(ctx, h)
				Expect(rt.SeriesIndexI16(ctx, result, 0)).To(Equal(int16(-1000)))
				Expect(rt.SeriesIndexI16(ctx, result, 1)).To(Equal(int16(500)))
			})
			It("Should negate I32", func() {
				h := rt.SeriesCreateEmptyI32(ctx, 2)
				rt.SeriesSetElementI32(ctx, h, 0, 100000)
				rt.SeriesSetElementI32(ctx, h, 1, -50000)
				result := rt.SeriesNegateI32(ctx, h)
				Expect(rt.SeriesIndexI32(ctx, result, 0)).To(Equal(int32(-100000)))
				Expect(rt.SeriesIndexI32(ctx, result, 1)).To(Equal(int32(50000)))
			})
			It("Should negate I64", func() {
				h := rt.SeriesCreateEmptyI64(ctx, 2)
				rt.SeriesSetElementI64(ctx, h, 0, 10000000000)
				rt.SeriesSetElementI64(ctx, h, 1, -5000000000)
				result := rt.SeriesNegateI64(ctx, h)
				Expect(rt.SeriesIndexI64(ctx, result, 0)).To(Equal(int64(-10000000000)))
				Expect(rt.SeriesIndexI64(ctx, result, 1)).To(Equal(int64(5000000000)))
			})
			It("Should negate F32", func() {
				h := rt.SeriesCreateEmptyF32(ctx, 2)
				rt.SeriesSetElementF32(ctx, h, 0, 3.14)
				rt.SeriesSetElementF32(ctx, h, 1, -2.71)
				result := rt.SeriesNegateF32(ctx, h)
				Expect(rt.SeriesIndexF32(ctx, result, 0)).To(BeNumerically("~", -3.14, 0.01))
				Expect(rt.SeriesIndexF32(ctx, result, 1)).To(BeNumerically("~", 2.71, 0.01))
			})
			It("Should negate F64", func() {
				h := rt.SeriesCreateEmptyF64(ctx, 2)
				rt.SeriesSetElementF64(ctx, h, 0, 3.14159265)
				rt.SeriesSetElementF64(ctx, h, 1, -2.71828182)
				result := rt.SeriesNegateF64(ctx, h)
				Expect(rt.SeriesIndexF64(ctx, result, 0)).To(BeNumerically("~", -3.14159265, 0.0000001))
				Expect(rt.SeriesIndexF64(ctx, result, 1)).To(BeNumerically("~", 2.71828182, 0.0000001))
			})
			It("Should return 0 for invalid handle", func() {
				Expect(rt.SeriesNegateI32(ctx, 999)).To(Equal(uint32(0)))
				Expect(rt.SeriesNegateF64(ctx, 999)).To(Equal(uint32(0)))
			})
		})

		Describe("NOT", func() {
			It("Should perform bitwise NOT on U8", func() {
				h := rt.SeriesCreateEmptyU8(ctx, 4)
				rt.SeriesSetElementU8(ctx, h, 0, 0x00)
				rt.SeriesSetElementU8(ctx, h, 1, 0xFF)
				rt.SeriesSetElementU8(ctx, h, 2, 0x0F)
				rt.SeriesSetElementU8(ctx, h, 3, 0xF0)
				result := rt.SeriesNotU8(ctx, h)
				Expect(rt.SeriesIndexU8(ctx, result, 0)).To(Equal(uint8(0xFF)))
				Expect(rt.SeriesIndexU8(ctx, result, 1)).To(Equal(uint8(0x00)))
				Expect(rt.SeriesIndexU8(ctx, result, 2)).To(Equal(uint8(0xF0)))
				Expect(rt.SeriesIndexU8(ctx, result, 3)).To(Equal(uint8(0x0F)))
			})
			It("Should return 0 for invalid handle", func() {
				Expect(rt.SeriesNotU8(ctx, 999)).To(Equal(uint32(0)))
			})
		})
	})

	Describe("Utility Operations", func() {
		It("Should return correct length", func() {
			h := rt.SeriesCreateEmptyF64(ctx, 7)
			Expect(rt.SeriesLen(ctx, h)).To(Equal(uint64(7)))
		})
		It("Should return 0 for invalid handle", func() {
			Expect(rt.SeriesLen(ctx, 999)).To(Equal(uint64(0)))
		})
		It("Should slice series", func() {
			h := rt.SeriesCreateEmptyF64(ctx, 5)
			for i := uint32(0); i < 5; i++ {
				rt.SeriesSetElementF64(ctx, h, i, float64(i*10))
			}
			sliced := rt.SeriesSlice(ctx, h, 1, 4)
			Expect(rt.SeriesLen(ctx, sliced)).To(Equal(uint64(3)))
			Expect(rt.SeriesIndexF64(ctx, sliced, 0)).To(Equal(float64(10)))
			Expect(rt.SeriesIndexF64(ctx, sliced, 1)).To(Equal(float64(20)))
			Expect(rt.SeriesIndexF64(ctx, sliced, 2)).To(Equal(float64(30)))
		})
	})

	Describe("Edge Cases", func() {
		It("Should handle empty series", func() {
			h1 := rt.SeriesCreateEmptyF64(ctx, 0)
			h2 := rt.SeriesCreateEmptyF64(ctx, 0)
			result := rt.SeriesSeriesAddF64(ctx, h1, h2)
			Expect(rt.SeriesLen(ctx, result)).To(Equal(uint64(0)))
		})
		It("Should return 0 for operations on invalid handles", func() {
			Expect(rt.SeriesIndexF64(ctx, 999, 0)).To(Equal(float64(0)))
			Expect(rt.SeriesElementAddF64(ctx, 999, 1.0)).To(Equal(uint32(0)))
			Expect(rt.SeriesSeriesAddF64(ctx, 999, 888)).To(Equal(uint32(0)))
			Expect(rt.SeriesCompareGTF64(ctx, 999, 888)).To(Equal(uint32(0)))
		})
	})

	Describe("String Operations", func() {
		Describe("StringLen", func() {
			It("Should return 0 for invalid handle", func() {
				Expect(rt.StringLen(ctx, 999)).To(Equal(uint32(0)))
			})
		})

		Describe("StringEqual", func() {
			It("Should return 0 for invalid handles", func() {
				Expect(rt.StringEqual(ctx, 999, 998)).To(Equal(uint32(0)))
			})
		})

		Describe("StringConcat", func() {
			It("Should return 0 for invalid handles", func() {
				Expect(rt.StringConcat(ctx, 999, 998)).To(Equal(uint32(0)))
			})

			It("Should return 0 when first handle is invalid", func() {
				Expect(rt.StringConcat(ctx, 999, 0)).To(Equal(uint32(0)))
			})

			It("Should return 0 when second handle is invalid", func() {
				Expect(rt.StringConcat(ctx, 0, 999)).To(Equal(uint32(0)))
			})
		})
	})
})
