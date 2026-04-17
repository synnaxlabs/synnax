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

// AndBool computes the element-wise logical AND of two boolean series. Inputs
// are assumed canonical (each byte 0x00 or 0x01); the operation is bitwise,
// which coincides with logical AND under that invariant.
func AndBool(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, uint8](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, uint8](rhs.Data)
	outData := xunsafe.CastSlice[uint8, uint8](output.Data)

	var lhsLast, rhsLast uint8
	if lhsLen > 0 {
		lhsLast = lhsData[lhsLen-1]
	}
	if rhsLen > 0 {
		rhsLast = rhsData[rhsLen-1]
	}

	for i := int64(0); i < maxLen; i++ {
		lhsVal := lhsLast
		if i < lhsLen {
			lhsVal = lhsData[i]
			lhsLast = lhsVal
		}
		rhsVal := rhsLast
		if i < rhsLen {
			rhsVal = rhsData[i]
			rhsLast = rhsVal
		}
		outData[i] = lhsVal & rhsVal
	}
}

// OrBool computes the element-wise logical OR of two boolean series. See AndBool
// for the canonical-input assumption.
func OrBool(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, uint8](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, uint8](rhs.Data)
	outData := xunsafe.CastSlice[uint8, uint8](output.Data)

	var lhsLast, rhsLast uint8
	if lhsLen > 0 {
		lhsLast = lhsData[lhsLen-1]
	}
	if rhsLen > 0 {
		rhsLast = rhsData[rhsLen-1]
	}

	for i := int64(0); i < maxLen; i++ {
		lhsVal := lhsLast
		if i < lhsLen {
			lhsVal = lhsData[i]
			lhsLast = lhsVal
		}
		rhsVal := rhsLast
		if i < rhsLen {
			rhsVal = rhsData[i]
			rhsLast = rhsVal
		}
		outData[i] = lhsVal | rhsVal
	}
}

// NotBool computes the element-wise logical NOT of a boolean series. Any
// non-zero input maps to 0; zero maps to 1. The output is always canonical.
func NotBool(input telem.Series, output *telem.Series) {
	inputLen := input.Len()
	output.Resize(inputLen)

	inData := xunsafe.CastSlice[uint8, uint8](input.Data)
	outData := xunsafe.CastSlice[uint8, uint8](output.Data)

	for i := int64(0); i < inputLen; i++ {
		if inData[i] == 0 {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}
