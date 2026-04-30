// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package op

import (
	"github.com/synnaxlabs/x/telem"
	xunsafe "github.com/synnaxlabs/x/unsafe"
)

// NegateU8 negates uint8 values, promoting the output to int16.
func NegateU8(input telem.Series, output *telem.Series) {
	inputLen := input.Len()
	output.DataType = telem.Int16T
	output.Resize(inputLen)
	inData := xunsafe.CastSlice[uint8, uint8](input.Data)
	outData := xunsafe.CastSlice[uint8, int16](output.Data)
	for i := int64(0); i < inputLen; i++ {
		outData[i] = -int16(inData[i])
	}
}

// NegateU16 negates uint16 values, promoting the output to int32.
func NegateU16(input telem.Series, output *telem.Series) {
	inputLen := input.Len()
	output.DataType = telem.Int32T
	output.Resize(inputLen)
	inData := xunsafe.CastSlice[uint8, uint16](input.Data)
	outData := xunsafe.CastSlice[uint8, int32](output.Data)
	for i := int64(0); i < inputLen; i++ {
		outData[i] = -int32(inData[i])
	}
}

// NegateU32 negates uint32 values, promoting the output to int64.
func NegateU32(input telem.Series, output *telem.Series) {
	inputLen := input.Len()
	output.DataType = telem.Int64T
	output.Resize(inputLen)
	inData := xunsafe.CastSlice[uint8, uint32](input.Data)
	outData := xunsafe.CastSlice[uint8, int64](output.Data)
	for i := int64(0); i < inputLen; i++ {
		outData[i] = -int64(inData[i])
	}
}

// NegateU64 negates uint64 values, promoting the output to float64.
// Precision loss is possible for values > 2^53.
func NegateU64(input telem.Series, output *telem.Series) {
	inputLen := input.Len()
	output.DataType = telem.Float64T
	output.Resize(inputLen)
	inData := xunsafe.CastSlice[uint8, uint64](input.Data)
	outData := xunsafe.CastSlice[uint8, float64](output.Data)
	for i := int64(0); i < inputLen; i++ {
		outData[i] = -float64(inData[i])
	}
}
