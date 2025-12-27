// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:generate go run generate/main.go
package bindings

import (
	compilerbindings "github.com/synnaxlabs/arc/compiler/bindings"
)

// BindRuntime connects a Runtime implementation to the static compiler Bindings.
func BindRuntime(runtime *Runtime, b *compilerbindings.Bindings) {
	// Channel operations
	b.ChannelReadU8 = runtime.ChannelReadU8
	b.ChannelWriteU8 = runtime.ChannelWriteU8
	b.ChannelReadU16 = runtime.ChannelReadU16
	b.ChannelWriteU16 = runtime.ChannelWriteU16
	b.ChannelReadU32 = runtime.ChannelReadU32
	b.ChannelWriteU32 = runtime.ChannelWriteU32
	b.ChannelReadU64 = runtime.ChannelReadU64
	b.ChannelWriteU64 = runtime.ChannelWriteU64
	b.ChannelReadI8 = runtime.ChannelReadI8
	b.ChannelWriteI8 = runtime.ChannelWriteI8
	b.ChannelReadI16 = runtime.ChannelReadI16
	b.ChannelWriteI16 = runtime.ChannelWriteI16
	b.ChannelReadI32 = runtime.ChannelReadI32
	b.ChannelWriteI32 = runtime.ChannelWriteI32
	b.ChannelReadI64 = runtime.ChannelReadI64
	b.ChannelWriteI64 = runtime.ChannelWriteI64
	b.ChannelReadF32 = runtime.ChannelReadF32
	b.ChannelWriteF32 = runtime.ChannelWriteF32
	b.ChannelReadF64 = runtime.ChannelReadF64
	b.ChannelWriteF64 = runtime.ChannelWriteF64
	b.ChannelReadStr = runtime.ChannelReadStr
	b.ChannelWriteStr = runtime.ChannelWriteStr

	// State operations
	b.StateLoadU8 = runtime.StateLoadU8
	b.StateStoreU8 = runtime.StateStoreU8
	b.StateLoadU16 = runtime.StateLoadU16
	b.StateStoreU16 = runtime.StateStoreU16
	b.StateLoadU32 = runtime.StateLoadU32
	b.StateStoreU32 = runtime.StateStoreU32
	b.StateLoadU64 = runtime.StateLoadU64
	b.StateStoreU64 = runtime.StateStoreU64
	b.StateLoadI8 = runtime.StateLoadI8
	b.StateStoreI8 = runtime.StateStoreI8
	b.StateLoadI16 = runtime.StateLoadI16
	b.StateStoreI16 = runtime.StateStoreI16
	b.StateLoadI32 = runtime.StateLoadI32
	b.StateStoreI32 = runtime.StateStoreI32
	b.StateLoadI64 = runtime.StateLoadI64
	b.StateStoreI64 = runtime.StateStoreI64
	b.StateLoadF32 = runtime.StateLoadF32
	b.StateStoreF32 = runtime.StateStoreF32
	b.StateLoadF64 = runtime.StateLoadF64
	b.StateStoreF64 = runtime.StateStoreF64
	b.StateLoadStr = runtime.StateLoadStr
	b.StateStoreStr = runtime.StateStoreStr

	// String operations
	b.StringFromLiteral = runtime.StringFromLiteral
	b.StringConcat = runtime.StringConcat
	b.StringLen = runtime.StringLen
	b.StringEqual = runtime.StringEqual

	// Generic operations
	b.Now = runtime.Now
	b.Panic = runtime.Panic

	// Math operations
	b.MathPowF32 = runtime.MathPowF32
	b.MathPowF64 = runtime.MathPowF64
	b.MathPowU8 = runtime.MathPowU8
	b.MathPowU16 = runtime.MathPowU16
	b.MathPowU32 = runtime.MathPowU32
	b.MathPowU64 = runtime.MathPowU64
	b.MathPowI8 = runtime.MathPowI8
	b.MathPowI16 = runtime.MathPowI16
	b.MathPowI32 = runtime.MathPowI32
	b.MathPowI64 = runtime.MathPowI64

	// Series operations - U8
	b.SeriesCreateEmptyU8 = runtime.SeriesCreateEmptyU8
	b.SeriesSetElementU8 = runtime.SeriesSetElementU8
	b.SeriesIndexU8 = runtime.SeriesIndexU8
	b.SeriesElementAddU8 = runtime.SeriesElementAddU8
	b.SeriesElementSubU8 = runtime.SeriesElementSubU8
	b.SeriesElementMulU8 = runtime.SeriesElementMulU8
	b.SeriesElementDivU8 = runtime.SeriesElementDivU8
	b.SeriesElementModU8 = runtime.SeriesElementModU8
	b.SeriesElementRSubU8 = runtime.SeriesElementRSubU8
	b.SeriesElementRDivU8 = runtime.SeriesElementRDivU8
	b.SeriesSeriesAddU8 = runtime.SeriesSeriesAddU8
	b.SeriesSeriesSubU8 = runtime.SeriesSeriesSubU8
	b.SeriesSeriesMulU8 = runtime.SeriesSeriesMulU8
	b.SeriesSeriesDivU8 = runtime.SeriesSeriesDivU8
	b.SeriesSeriesModU8 = runtime.SeriesSeriesModU8
	b.SeriesCompareGTU8 = runtime.SeriesCompareGTU8
	b.SeriesCompareLTU8 = runtime.SeriesCompareLTU8
	b.SeriesCompareGEU8 = runtime.SeriesCompareGEU8
	b.SeriesCompareLEU8 = runtime.SeriesCompareLEU8
	b.SeriesCompareEQU8 = runtime.SeriesCompareEQU8
	b.SeriesCompareNEU8 = runtime.SeriesCompareNEU8
	b.SeriesCompareGTScalarU8 = runtime.SeriesCompareGTScalarU8
	b.SeriesCompareLTScalarU8 = runtime.SeriesCompareLTScalarU8
	b.SeriesCompareGEScalarU8 = runtime.SeriesCompareGEScalarU8
	b.SeriesCompareLEScalarU8 = runtime.SeriesCompareLEScalarU8
	b.SeriesCompareEQScalarU8 = runtime.SeriesCompareEQScalarU8
	b.SeriesCompareNEScalarU8 = runtime.SeriesCompareNEScalarU8
	b.StateLoadSeriesU8 = runtime.StateLoadSeriesU8
	b.StateStoreSeriesU8 = runtime.StateStoreSeriesU8

	// Series operations - U16
	b.SeriesCreateEmptyU16 = runtime.SeriesCreateEmptyU16
	b.SeriesSetElementU16 = runtime.SeriesSetElementU16
	b.SeriesIndexU16 = runtime.SeriesIndexU16
	b.SeriesElementAddU16 = runtime.SeriesElementAddU16
	b.SeriesElementSubU16 = runtime.SeriesElementSubU16
	b.SeriesElementMulU16 = runtime.SeriesElementMulU16
	b.SeriesElementDivU16 = runtime.SeriesElementDivU16
	b.SeriesElementModU16 = runtime.SeriesElementModU16
	b.SeriesElementRSubU16 = runtime.SeriesElementRSubU16
	b.SeriesElementRDivU16 = runtime.SeriesElementRDivU16
	b.SeriesSeriesAddU16 = runtime.SeriesSeriesAddU16
	b.SeriesSeriesSubU16 = runtime.SeriesSeriesSubU16
	b.SeriesSeriesMulU16 = runtime.SeriesSeriesMulU16
	b.SeriesSeriesDivU16 = runtime.SeriesSeriesDivU16
	b.SeriesSeriesModU16 = runtime.SeriesSeriesModU16
	b.SeriesCompareGTU16 = runtime.SeriesCompareGTU16
	b.SeriesCompareLTU16 = runtime.SeriesCompareLTU16
	b.SeriesCompareGEU16 = runtime.SeriesCompareGEU16
	b.SeriesCompareLEU16 = runtime.SeriesCompareLEU16
	b.SeriesCompareEQU16 = runtime.SeriesCompareEQU16
	b.SeriesCompareNEU16 = runtime.SeriesCompareNEU16
	b.SeriesCompareGTScalarU16 = runtime.SeriesCompareGTScalarU16
	b.SeriesCompareLTScalarU16 = runtime.SeriesCompareLTScalarU16
	b.SeriesCompareGEScalarU16 = runtime.SeriesCompareGEScalarU16
	b.SeriesCompareLEScalarU16 = runtime.SeriesCompareLEScalarU16
	b.SeriesCompareEQScalarU16 = runtime.SeriesCompareEQScalarU16
	b.SeriesCompareNEScalarU16 = runtime.SeriesCompareNEScalarU16
	b.StateLoadSeriesU16 = runtime.StateLoadSeriesU16
	b.StateStoreSeriesU16 = runtime.StateStoreSeriesU16

	// Series operations - U32
	b.SeriesCreateEmptyU32 = runtime.SeriesCreateEmptyU32
	b.SeriesSetElementU32 = runtime.SeriesSetElementU32
	b.SeriesIndexU32 = runtime.SeriesIndexU32
	b.SeriesElementAddU32 = runtime.SeriesElementAddU32
	b.SeriesElementSubU32 = runtime.SeriesElementSubU32
	b.SeriesElementMulU32 = runtime.SeriesElementMulU32
	b.SeriesElementDivU32 = runtime.SeriesElementDivU32
	b.SeriesElementModU32 = runtime.SeriesElementModU32
	b.SeriesElementRSubU32 = runtime.SeriesElementRSubU32
	b.SeriesElementRDivU32 = runtime.SeriesElementRDivU32
	b.SeriesSeriesAddU32 = runtime.SeriesSeriesAddU32
	b.SeriesSeriesSubU32 = runtime.SeriesSeriesSubU32
	b.SeriesSeriesMulU32 = runtime.SeriesSeriesMulU32
	b.SeriesSeriesDivU32 = runtime.SeriesSeriesDivU32
	b.SeriesSeriesModU32 = runtime.SeriesSeriesModU32
	b.SeriesCompareGTU32 = runtime.SeriesCompareGTU32
	b.SeriesCompareLTU32 = runtime.SeriesCompareLTU32
	b.SeriesCompareGEU32 = runtime.SeriesCompareGEU32
	b.SeriesCompareLEU32 = runtime.SeriesCompareLEU32
	b.SeriesCompareEQU32 = runtime.SeriesCompareEQU32
	b.SeriesCompareNEU32 = runtime.SeriesCompareNEU32
	b.SeriesCompareGTScalarU32 = runtime.SeriesCompareGTScalarU32
	b.SeriesCompareLTScalarU32 = runtime.SeriesCompareLTScalarU32
	b.SeriesCompareGEScalarU32 = runtime.SeriesCompareGEScalarU32
	b.SeriesCompareLEScalarU32 = runtime.SeriesCompareLEScalarU32
	b.SeriesCompareEQScalarU32 = runtime.SeriesCompareEQScalarU32
	b.SeriesCompareNEScalarU32 = runtime.SeriesCompareNEScalarU32
	b.StateLoadSeriesU32 = runtime.StateLoadSeriesU32
	b.StateStoreSeriesU32 = runtime.StateStoreSeriesU32

	// Series operations - U64
	b.SeriesCreateEmptyU64 = runtime.SeriesCreateEmptyU64
	b.SeriesSetElementU64 = runtime.SeriesSetElementU64
	b.SeriesIndexU64 = runtime.SeriesIndexU64
	b.SeriesElementAddU64 = runtime.SeriesElementAddU64
	b.SeriesElementSubU64 = runtime.SeriesElementSubU64
	b.SeriesElementMulU64 = runtime.SeriesElementMulU64
	b.SeriesElementDivU64 = runtime.SeriesElementDivU64
	b.SeriesElementModU64 = runtime.SeriesElementModU64
	b.SeriesElementRSubU64 = runtime.SeriesElementRSubU64
	b.SeriesElementRDivU64 = runtime.SeriesElementRDivU64
	b.SeriesSeriesAddU64 = runtime.SeriesSeriesAddU64
	b.SeriesSeriesSubU64 = runtime.SeriesSeriesSubU64
	b.SeriesSeriesMulU64 = runtime.SeriesSeriesMulU64
	b.SeriesSeriesDivU64 = runtime.SeriesSeriesDivU64
	b.SeriesSeriesModU64 = runtime.SeriesSeriesModU64
	b.SeriesCompareGTU64 = runtime.SeriesCompareGTU64
	b.SeriesCompareLTU64 = runtime.SeriesCompareLTU64
	b.SeriesCompareGEU64 = runtime.SeriesCompareGEU64
	b.SeriesCompareLEU64 = runtime.SeriesCompareLEU64
	b.SeriesCompareEQU64 = runtime.SeriesCompareEQU64
	b.SeriesCompareNEU64 = runtime.SeriesCompareNEU64
	b.SeriesCompareGTScalarU64 = runtime.SeriesCompareGTScalarU64
	b.SeriesCompareLTScalarU64 = runtime.SeriesCompareLTScalarU64
	b.SeriesCompareGEScalarU64 = runtime.SeriesCompareGEScalarU64
	b.SeriesCompareLEScalarU64 = runtime.SeriesCompareLEScalarU64
	b.SeriesCompareEQScalarU64 = runtime.SeriesCompareEQScalarU64
	b.SeriesCompareNEScalarU64 = runtime.SeriesCompareNEScalarU64
	b.StateLoadSeriesU64 = runtime.StateLoadSeriesU64
	b.StateStoreSeriesU64 = runtime.StateStoreSeriesU64

	// Series operations - I8
	b.SeriesCreateEmptyI8 = runtime.SeriesCreateEmptyI8
	b.SeriesSetElementI8 = runtime.SeriesSetElementI8
	b.SeriesIndexI8 = runtime.SeriesIndexI8
	b.SeriesElementAddI8 = runtime.SeriesElementAddI8
	b.SeriesElementSubI8 = runtime.SeriesElementSubI8
	b.SeriesElementMulI8 = runtime.SeriesElementMulI8
	b.SeriesElementDivI8 = runtime.SeriesElementDivI8
	b.SeriesElementModI8 = runtime.SeriesElementModI8
	b.SeriesElementRSubI8 = runtime.SeriesElementRSubI8
	b.SeriesElementRDivI8 = runtime.SeriesElementRDivI8
	b.SeriesSeriesAddI8 = runtime.SeriesSeriesAddI8
	b.SeriesSeriesSubI8 = runtime.SeriesSeriesSubI8
	b.SeriesSeriesMulI8 = runtime.SeriesSeriesMulI8
	b.SeriesSeriesDivI8 = runtime.SeriesSeriesDivI8
	b.SeriesSeriesModI8 = runtime.SeriesSeriesModI8
	b.SeriesCompareGTI8 = runtime.SeriesCompareGTI8
	b.SeriesCompareLTI8 = runtime.SeriesCompareLTI8
	b.SeriesCompareGEI8 = runtime.SeriesCompareGEI8
	b.SeriesCompareLEI8 = runtime.SeriesCompareLEI8
	b.SeriesCompareEQI8 = runtime.SeriesCompareEQI8
	b.SeriesCompareNEI8 = runtime.SeriesCompareNEI8
	b.SeriesCompareGTScalarI8 = runtime.SeriesCompareGTScalarI8
	b.SeriesCompareLTScalarI8 = runtime.SeriesCompareLTScalarI8
	b.SeriesCompareGEScalarI8 = runtime.SeriesCompareGEScalarI8
	b.SeriesCompareLEScalarI8 = runtime.SeriesCompareLEScalarI8
	b.SeriesCompareEQScalarI8 = runtime.SeriesCompareEQScalarI8
	b.SeriesCompareNEScalarI8 = runtime.SeriesCompareNEScalarI8
	b.StateLoadSeriesI8 = runtime.StateLoadSeriesI8
	b.StateStoreSeriesI8 = runtime.StateStoreSeriesI8

	// Series operations - I16
	b.SeriesCreateEmptyI16 = runtime.SeriesCreateEmptyI16
	b.SeriesSetElementI16 = runtime.SeriesSetElementI16
	b.SeriesIndexI16 = runtime.SeriesIndexI16
	b.SeriesElementAddI16 = runtime.SeriesElementAddI16
	b.SeriesElementSubI16 = runtime.SeriesElementSubI16
	b.SeriesElementMulI16 = runtime.SeriesElementMulI16
	b.SeriesElementDivI16 = runtime.SeriesElementDivI16
	b.SeriesElementModI16 = runtime.SeriesElementModI16
	b.SeriesElementRSubI16 = runtime.SeriesElementRSubI16
	b.SeriesElementRDivI16 = runtime.SeriesElementRDivI16
	b.SeriesSeriesAddI16 = runtime.SeriesSeriesAddI16
	b.SeriesSeriesSubI16 = runtime.SeriesSeriesSubI16
	b.SeriesSeriesMulI16 = runtime.SeriesSeriesMulI16
	b.SeriesSeriesDivI16 = runtime.SeriesSeriesDivI16
	b.SeriesSeriesModI16 = runtime.SeriesSeriesModI16
	b.SeriesCompareGTI16 = runtime.SeriesCompareGTI16
	b.SeriesCompareLTI16 = runtime.SeriesCompareLTI16
	b.SeriesCompareGEI16 = runtime.SeriesCompareGEI16
	b.SeriesCompareLEI16 = runtime.SeriesCompareLEI16
	b.SeriesCompareEQI16 = runtime.SeriesCompareEQI16
	b.SeriesCompareNEI16 = runtime.SeriesCompareNEI16
	b.SeriesCompareGTScalarI16 = runtime.SeriesCompareGTScalarI16
	b.SeriesCompareLTScalarI16 = runtime.SeriesCompareLTScalarI16
	b.SeriesCompareGEScalarI16 = runtime.SeriesCompareGEScalarI16
	b.SeriesCompareLEScalarI16 = runtime.SeriesCompareLEScalarI16
	b.SeriesCompareEQScalarI16 = runtime.SeriesCompareEQScalarI16
	b.SeriesCompareNEScalarI16 = runtime.SeriesCompareNEScalarI16
	b.StateLoadSeriesI16 = runtime.StateLoadSeriesI16
	b.StateStoreSeriesI16 = runtime.StateStoreSeriesI16

	// Series operations - I32
	b.SeriesCreateEmptyI32 = runtime.SeriesCreateEmptyI32
	b.SeriesSetElementI32 = runtime.SeriesSetElementI32
	b.SeriesIndexI32 = runtime.SeriesIndexI32
	b.SeriesElementAddI32 = runtime.SeriesElementAddI32
	b.SeriesElementSubI32 = runtime.SeriesElementSubI32
	b.SeriesElementMulI32 = runtime.SeriesElementMulI32
	b.SeriesElementDivI32 = runtime.SeriesElementDivI32
	b.SeriesElementModI32 = runtime.SeriesElementModI32
	b.SeriesElementRSubI32 = runtime.SeriesElementRSubI32
	b.SeriesElementRDivI32 = runtime.SeriesElementRDivI32
	b.SeriesSeriesAddI32 = runtime.SeriesSeriesAddI32
	b.SeriesSeriesSubI32 = runtime.SeriesSeriesSubI32
	b.SeriesSeriesMulI32 = runtime.SeriesSeriesMulI32
	b.SeriesSeriesDivI32 = runtime.SeriesSeriesDivI32
	b.SeriesSeriesModI32 = runtime.SeriesSeriesModI32
	b.SeriesCompareGTI32 = runtime.SeriesCompareGTI32
	b.SeriesCompareLTI32 = runtime.SeriesCompareLTI32
	b.SeriesCompareGEI32 = runtime.SeriesCompareGEI32
	b.SeriesCompareLEI32 = runtime.SeriesCompareLEI32
	b.SeriesCompareEQI32 = runtime.SeriesCompareEQI32
	b.SeriesCompareNEI32 = runtime.SeriesCompareNEI32
	b.SeriesCompareGTScalarI32 = runtime.SeriesCompareGTScalarI32
	b.SeriesCompareLTScalarI32 = runtime.SeriesCompareLTScalarI32
	b.SeriesCompareGEScalarI32 = runtime.SeriesCompareGEScalarI32
	b.SeriesCompareLEScalarI32 = runtime.SeriesCompareLEScalarI32
	b.SeriesCompareEQScalarI32 = runtime.SeriesCompareEQScalarI32
	b.SeriesCompareNEScalarI32 = runtime.SeriesCompareNEScalarI32
	b.StateLoadSeriesI32 = runtime.StateLoadSeriesI32
	b.StateStoreSeriesI32 = runtime.StateStoreSeriesI32

	// Series operations - I64
	b.SeriesCreateEmptyI64 = runtime.SeriesCreateEmptyI64
	b.SeriesSetElementI64 = runtime.SeriesSetElementI64
	b.SeriesIndexI64 = runtime.SeriesIndexI64
	b.SeriesElementAddI64 = runtime.SeriesElementAddI64
	b.SeriesElementSubI64 = runtime.SeriesElementSubI64
	b.SeriesElementMulI64 = runtime.SeriesElementMulI64
	b.SeriesElementDivI64 = runtime.SeriesElementDivI64
	b.SeriesElementModI64 = runtime.SeriesElementModI64
	b.SeriesElementRSubI64 = runtime.SeriesElementRSubI64
	b.SeriesElementRDivI64 = runtime.SeriesElementRDivI64
	b.SeriesSeriesAddI64 = runtime.SeriesSeriesAddI64
	b.SeriesSeriesSubI64 = runtime.SeriesSeriesSubI64
	b.SeriesSeriesMulI64 = runtime.SeriesSeriesMulI64
	b.SeriesSeriesDivI64 = runtime.SeriesSeriesDivI64
	b.SeriesSeriesModI64 = runtime.SeriesSeriesModI64
	b.SeriesCompareGTI64 = runtime.SeriesCompareGTI64
	b.SeriesCompareLTI64 = runtime.SeriesCompareLTI64
	b.SeriesCompareGEI64 = runtime.SeriesCompareGEI64
	b.SeriesCompareLEI64 = runtime.SeriesCompareLEI64
	b.SeriesCompareEQI64 = runtime.SeriesCompareEQI64
	b.SeriesCompareNEI64 = runtime.SeriesCompareNEI64
	b.SeriesCompareGTScalarI64 = runtime.SeriesCompareGTScalarI64
	b.SeriesCompareLTScalarI64 = runtime.SeriesCompareLTScalarI64
	b.SeriesCompareGEScalarI64 = runtime.SeriesCompareGEScalarI64
	b.SeriesCompareLEScalarI64 = runtime.SeriesCompareLEScalarI64
	b.SeriesCompareEQScalarI64 = runtime.SeriesCompareEQScalarI64
	b.SeriesCompareNEScalarI64 = runtime.SeriesCompareNEScalarI64
	b.StateLoadSeriesI64 = runtime.StateLoadSeriesI64
	b.StateStoreSeriesI64 = runtime.StateStoreSeriesI64

	// Series operations - F32
	b.SeriesCreateEmptyF32 = runtime.SeriesCreateEmptyF32
	b.SeriesSetElementF32 = runtime.SeriesSetElementF32
	b.SeriesIndexF32 = runtime.SeriesIndexF32
	b.SeriesElementAddF32 = runtime.SeriesElementAddF32
	b.SeriesElementSubF32 = runtime.SeriesElementSubF32
	b.SeriesElementMulF32 = runtime.SeriesElementMulF32
	b.SeriesElementDivF32 = runtime.SeriesElementDivF32
	b.SeriesElementModF32 = runtime.SeriesElementModF32
	b.SeriesElementRSubF32 = runtime.SeriesElementRSubF32
	b.SeriesElementRDivF32 = runtime.SeriesElementRDivF32
	b.SeriesSeriesAddF32 = runtime.SeriesSeriesAddF32
	b.SeriesSeriesSubF32 = runtime.SeriesSeriesSubF32
	b.SeriesSeriesMulF32 = runtime.SeriesSeriesMulF32
	b.SeriesSeriesDivF32 = runtime.SeriesSeriesDivF32
	b.SeriesSeriesModF32 = runtime.SeriesSeriesModF32
	b.SeriesCompareGTF32 = runtime.SeriesCompareGTF32
	b.SeriesCompareLTF32 = runtime.SeriesCompareLTF32
	b.SeriesCompareGEF32 = runtime.SeriesCompareGEF32
	b.SeriesCompareLEF32 = runtime.SeriesCompareLEF32
	b.SeriesCompareEQF32 = runtime.SeriesCompareEQF32
	b.SeriesCompareNEF32 = runtime.SeriesCompareNEF32
	b.SeriesCompareGTScalarF32 = runtime.SeriesCompareGTScalarF32
	b.SeriesCompareLTScalarF32 = runtime.SeriesCompareLTScalarF32
	b.SeriesCompareGEScalarF32 = runtime.SeriesCompareGEScalarF32
	b.SeriesCompareLEScalarF32 = runtime.SeriesCompareLEScalarF32
	b.SeriesCompareEQScalarF32 = runtime.SeriesCompareEQScalarF32
	b.SeriesCompareNEScalarF32 = runtime.SeriesCompareNEScalarF32
	b.StateLoadSeriesF32 = runtime.StateLoadSeriesF32
	b.StateStoreSeriesF32 = runtime.StateStoreSeriesF32

	// Series operations - F64
	b.SeriesCreateEmptyF64 = runtime.SeriesCreateEmptyF64
	b.SeriesSetElementF64 = runtime.SeriesSetElementF64
	b.SeriesIndexF64 = runtime.SeriesIndexF64
	b.SeriesElementAddF64 = runtime.SeriesElementAddF64
	b.SeriesElementSubF64 = runtime.SeriesElementSubF64
	b.SeriesElementMulF64 = runtime.SeriesElementMulF64
	b.SeriesElementDivF64 = runtime.SeriesElementDivF64
	b.SeriesElementModF64 = runtime.SeriesElementModF64
	b.SeriesElementRSubF64 = runtime.SeriesElementRSubF64
	b.SeriesElementRDivF64 = runtime.SeriesElementRDivF64
	b.SeriesSeriesAddF64 = runtime.SeriesSeriesAddF64
	b.SeriesSeriesSubF64 = runtime.SeriesSeriesSubF64
	b.SeriesSeriesMulF64 = runtime.SeriesSeriesMulF64
	b.SeriesSeriesDivF64 = runtime.SeriesSeriesDivF64
	b.SeriesSeriesModF64 = runtime.SeriesSeriesModF64
	b.SeriesCompareGTF64 = runtime.SeriesCompareGTF64
	b.SeriesCompareLTF64 = runtime.SeriesCompareLTF64
	b.SeriesCompareGEF64 = runtime.SeriesCompareGEF64
	b.SeriesCompareLEF64 = runtime.SeriesCompareLEF64
	b.SeriesCompareEQF64 = runtime.SeriesCompareEQF64
	b.SeriesCompareNEF64 = runtime.SeriesCompareNEF64
	b.SeriesCompareGTScalarF64 = runtime.SeriesCompareGTScalarF64
	b.SeriesCompareLTScalarF64 = runtime.SeriesCompareLTScalarF64
	b.SeriesCompareGEScalarF64 = runtime.SeriesCompareGEScalarF64
	b.SeriesCompareLEScalarF64 = runtime.SeriesCompareLEScalarF64
	b.SeriesCompareEQScalarF64 = runtime.SeriesCompareEQScalarF64
	b.SeriesCompareNEScalarF64 = runtime.SeriesCompareNEScalarF64
	b.StateLoadSeriesF64 = runtime.StateLoadSeriesF64
	b.StateStoreSeriesF64 = runtime.StateStoreSeriesF64

	// Generic series operations
	b.SeriesLen = runtime.SeriesLen
	b.SeriesSlice = runtime.SeriesSlice

	// Series unary operations
	b.SeriesNegateF64 = runtime.SeriesNegateF64
	b.SeriesNegateF32 = runtime.SeriesNegateF32
	b.SeriesNegateI64 = runtime.SeriesNegateI64
	b.SeriesNegateI32 = runtime.SeriesNegateI32
	b.SeriesNegateI16 = runtime.SeriesNegateI16
	b.SeriesNegateI8 = runtime.SeriesNegateI8
	b.SeriesNotU8 = runtime.SeriesNotU8
}
