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
}
