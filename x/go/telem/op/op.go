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
	"math"

	"github.com/synnaxlabs/x/telem"
	xunsafe "github.com/synnaxlabs/x/unsafe"
)

var _ = math.Mod // Ensure math is used

func GreaterThanF64(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, float64](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, float64](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast float64
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
		if lhsVal > rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func GreaterThanOrEqualF64(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, float64](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, float64](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast float64
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
		if lhsVal >= rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func LessThanF64(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, float64](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, float64](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast float64
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
		if lhsVal < rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func LessThanOrEqualF64(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, float64](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, float64](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast float64
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
		if lhsVal <= rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func EqualF64(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, float64](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, float64](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast float64
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
		if lhsVal == rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func NotEqualF64(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, float64](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, float64](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast float64
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
		if lhsVal != rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func AddF64(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, float64](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, float64](rhs.Data)
	outData := xunsafe.CastSlice[uint8, float64](output.Data)

	var lhsLast, rhsLast float64
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
		outData[i] = lhsVal + rhsVal
	}
}

func SubtractF64(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, float64](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, float64](rhs.Data)
	outData := xunsafe.CastSlice[uint8, float64](output.Data)

	var lhsLast, rhsLast float64
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
		outData[i] = lhsVal - rhsVal
	}
}

func MultiplyF64(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, float64](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, float64](rhs.Data)
	outData := xunsafe.CastSlice[uint8, float64](output.Data)

	var lhsLast, rhsLast float64
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
		outData[i] = lhsVal * rhsVal
	}
}

func DivideF64(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, float64](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, float64](rhs.Data)
	outData := xunsafe.CastSlice[uint8, float64](output.Data)

	var lhsLast, rhsLast float64
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
		outData[i] = lhsVal / rhsVal
	}
}

func GreaterThanF32(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, float32](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, float32](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast float32
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
		if lhsVal > rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func GreaterThanOrEqualF32(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, float32](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, float32](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast float32
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
		if lhsVal >= rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func LessThanF32(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, float32](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, float32](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast float32
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
		if lhsVal < rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func LessThanOrEqualF32(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, float32](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, float32](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast float32
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
		if lhsVal <= rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func EqualF32(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, float32](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, float32](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast float32
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
		if lhsVal == rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func NotEqualF32(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, float32](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, float32](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast float32
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
		if lhsVal != rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func AddF32(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, float32](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, float32](rhs.Data)
	outData := xunsafe.CastSlice[uint8, float32](output.Data)

	var lhsLast, rhsLast float32
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
		outData[i] = lhsVal + rhsVal
	}
}

func SubtractF32(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, float32](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, float32](rhs.Data)
	outData := xunsafe.CastSlice[uint8, float32](output.Data)

	var lhsLast, rhsLast float32
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
		outData[i] = lhsVal - rhsVal
	}
}

func MultiplyF32(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, float32](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, float32](rhs.Data)
	outData := xunsafe.CastSlice[uint8, float32](output.Data)

	var lhsLast, rhsLast float32
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
		outData[i] = lhsVal * rhsVal
	}
}

func DivideF32(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, float32](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, float32](rhs.Data)
	outData := xunsafe.CastSlice[uint8, float32](output.Data)

	var lhsLast, rhsLast float32
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
		outData[i] = lhsVal / rhsVal
	}
}

func GreaterThanI64(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, int64](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, int64](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast int64
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
		if lhsVal > rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func GreaterThanOrEqualI64(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, int64](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, int64](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast int64
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
		if lhsVal >= rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func LessThanI64(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, int64](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, int64](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast int64
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
		if lhsVal < rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func LessThanOrEqualI64(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, int64](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, int64](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast int64
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
		if lhsVal <= rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func EqualI64(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, int64](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, int64](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast int64
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
		if lhsVal == rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func NotEqualI64(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, int64](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, int64](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast int64
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
		if lhsVal != rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func AddI64(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, int64](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, int64](rhs.Data)
	outData := xunsafe.CastSlice[uint8, int64](output.Data)

	var lhsLast, rhsLast int64
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
		outData[i] = lhsVal + rhsVal
	}
}

func SubtractI64(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, int64](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, int64](rhs.Data)
	outData := xunsafe.CastSlice[uint8, int64](output.Data)

	var lhsLast, rhsLast int64
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
		outData[i] = lhsVal - rhsVal
	}
}

func MultiplyI64(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, int64](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, int64](rhs.Data)
	outData := xunsafe.CastSlice[uint8, int64](output.Data)

	var lhsLast, rhsLast int64
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
		outData[i] = lhsVal * rhsVal
	}
}

func DivideI64(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, int64](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, int64](rhs.Data)
	outData := xunsafe.CastSlice[uint8, int64](output.Data)

	var lhsLast, rhsLast int64
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
		outData[i] = lhsVal / rhsVal
	}
}

func GreaterThanI32(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, int32](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, int32](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast int32
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
		if lhsVal > rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func GreaterThanOrEqualI32(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, int32](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, int32](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast int32
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
		if lhsVal >= rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func LessThanI32(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, int32](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, int32](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast int32
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
		if lhsVal < rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func LessThanOrEqualI32(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, int32](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, int32](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast int32
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
		if lhsVal <= rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func EqualI32(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, int32](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, int32](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast int32
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
		if lhsVal == rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func NotEqualI32(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, int32](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, int32](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast int32
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
		if lhsVal != rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func AddI32(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, int32](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, int32](rhs.Data)
	outData := xunsafe.CastSlice[uint8, int32](output.Data)

	var lhsLast, rhsLast int32
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
		outData[i] = lhsVal + rhsVal
	}
}

func SubtractI32(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, int32](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, int32](rhs.Data)
	outData := xunsafe.CastSlice[uint8, int32](output.Data)

	var lhsLast, rhsLast int32
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
		outData[i] = lhsVal - rhsVal
	}
}

func MultiplyI32(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, int32](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, int32](rhs.Data)
	outData := xunsafe.CastSlice[uint8, int32](output.Data)

	var lhsLast, rhsLast int32
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
		outData[i] = lhsVal * rhsVal
	}
}

func DivideI32(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, int32](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, int32](rhs.Data)
	outData := xunsafe.CastSlice[uint8, int32](output.Data)

	var lhsLast, rhsLast int32
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
		outData[i] = lhsVal / rhsVal
	}
}

func GreaterThanI16(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, int16](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, int16](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast int16
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
		if lhsVal > rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func GreaterThanOrEqualI16(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, int16](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, int16](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast int16
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
		if lhsVal >= rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func LessThanI16(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, int16](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, int16](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast int16
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
		if lhsVal < rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func LessThanOrEqualI16(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, int16](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, int16](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast int16
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
		if lhsVal <= rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func EqualI16(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, int16](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, int16](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast int16
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
		if lhsVal == rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func NotEqualI16(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, int16](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, int16](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast int16
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
		if lhsVal != rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func AddI16(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, int16](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, int16](rhs.Data)
	outData := xunsafe.CastSlice[uint8, int16](output.Data)

	var lhsLast, rhsLast int16
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
		outData[i] = lhsVal + rhsVal
	}
}

func SubtractI16(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, int16](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, int16](rhs.Data)
	outData := xunsafe.CastSlice[uint8, int16](output.Data)

	var lhsLast, rhsLast int16
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
		outData[i] = lhsVal - rhsVal
	}
}

func MultiplyI16(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, int16](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, int16](rhs.Data)
	outData := xunsafe.CastSlice[uint8, int16](output.Data)

	var lhsLast, rhsLast int16
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
		outData[i] = lhsVal * rhsVal
	}
}

func DivideI16(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, int16](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, int16](rhs.Data)
	outData := xunsafe.CastSlice[uint8, int16](output.Data)

	var lhsLast, rhsLast int16
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
		outData[i] = lhsVal / rhsVal
	}
}

func GreaterThanI8(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, int8](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, int8](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast int8
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
		if lhsVal > rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func GreaterThanOrEqualI8(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, int8](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, int8](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast int8
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
		if lhsVal >= rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func LessThanI8(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, int8](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, int8](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast int8
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
		if lhsVal < rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func LessThanOrEqualI8(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, int8](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, int8](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast int8
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
		if lhsVal <= rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func EqualI8(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, int8](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, int8](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast int8
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
		if lhsVal == rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func NotEqualI8(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, int8](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, int8](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast int8
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
		if lhsVal != rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func AddI8(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, int8](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, int8](rhs.Data)
	outData := xunsafe.CastSlice[uint8, int8](output.Data)

	var lhsLast, rhsLast int8
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
		outData[i] = lhsVal + rhsVal
	}
}

func SubtractI8(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, int8](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, int8](rhs.Data)
	outData := xunsafe.CastSlice[uint8, int8](output.Data)

	var lhsLast, rhsLast int8
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
		outData[i] = lhsVal - rhsVal
	}
}

func MultiplyI8(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, int8](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, int8](rhs.Data)
	outData := xunsafe.CastSlice[uint8, int8](output.Data)

	var lhsLast, rhsLast int8
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
		outData[i] = lhsVal * rhsVal
	}
}

func DivideI8(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, int8](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, int8](rhs.Data)
	outData := xunsafe.CastSlice[uint8, int8](output.Data)

	var lhsLast, rhsLast int8
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
		outData[i] = lhsVal / rhsVal
	}
}

func GreaterThanU64(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, uint64](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, uint64](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast uint64
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
		if lhsVal > rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func GreaterThanOrEqualU64(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, uint64](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, uint64](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast uint64
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
		if lhsVal >= rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func LessThanU64(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, uint64](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, uint64](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast uint64
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
		if lhsVal < rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func LessThanOrEqualU64(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, uint64](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, uint64](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast uint64
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
		if lhsVal <= rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func EqualU64(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, uint64](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, uint64](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast uint64
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
		if lhsVal == rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func NotEqualU64(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, uint64](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, uint64](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast uint64
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
		if lhsVal != rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func AddU64(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, uint64](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, uint64](rhs.Data)
	outData := xunsafe.CastSlice[uint8, uint64](output.Data)

	var lhsLast, rhsLast uint64
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
		outData[i] = lhsVal + rhsVal
	}
}

func SubtractU64(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, uint64](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, uint64](rhs.Data)
	outData := xunsafe.CastSlice[uint8, uint64](output.Data)

	var lhsLast, rhsLast uint64
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
		outData[i] = lhsVal - rhsVal
	}
}

func MultiplyU64(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, uint64](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, uint64](rhs.Data)
	outData := xunsafe.CastSlice[uint8, uint64](output.Data)

	var lhsLast, rhsLast uint64
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
		outData[i] = lhsVal * rhsVal
	}
}

func DivideU64(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, uint64](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, uint64](rhs.Data)
	outData := xunsafe.CastSlice[uint8, uint64](output.Data)

	var lhsLast, rhsLast uint64
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
		outData[i] = lhsVal / rhsVal
	}
}

func GreaterThanU32(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, uint32](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, uint32](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast uint32
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
		if lhsVal > rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func GreaterThanOrEqualU32(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, uint32](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, uint32](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast uint32
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
		if lhsVal >= rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func LessThanU32(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, uint32](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, uint32](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast uint32
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
		if lhsVal < rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func LessThanOrEqualU32(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, uint32](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, uint32](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast uint32
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
		if lhsVal <= rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func EqualU32(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, uint32](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, uint32](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast uint32
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
		if lhsVal == rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func NotEqualU32(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, uint32](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, uint32](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast uint32
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
		if lhsVal != rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func AddU32(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, uint32](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, uint32](rhs.Data)
	outData := xunsafe.CastSlice[uint8, uint32](output.Data)

	var lhsLast, rhsLast uint32
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
		outData[i] = lhsVal + rhsVal
	}
}

func SubtractU32(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, uint32](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, uint32](rhs.Data)
	outData := xunsafe.CastSlice[uint8, uint32](output.Data)

	var lhsLast, rhsLast uint32
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
		outData[i] = lhsVal - rhsVal
	}
}

func MultiplyU32(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, uint32](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, uint32](rhs.Data)
	outData := xunsafe.CastSlice[uint8, uint32](output.Data)

	var lhsLast, rhsLast uint32
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
		outData[i] = lhsVal * rhsVal
	}
}

func DivideU32(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, uint32](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, uint32](rhs.Data)
	outData := xunsafe.CastSlice[uint8, uint32](output.Data)

	var lhsLast, rhsLast uint32
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
		outData[i] = lhsVal / rhsVal
	}
}

func GreaterThanU16(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, uint16](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, uint16](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast uint16
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
		if lhsVal > rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func GreaterThanOrEqualU16(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, uint16](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, uint16](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast uint16
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
		if lhsVal >= rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func LessThanU16(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, uint16](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, uint16](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast uint16
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
		if lhsVal < rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func LessThanOrEqualU16(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, uint16](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, uint16](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast uint16
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
		if lhsVal <= rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func EqualU16(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, uint16](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, uint16](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast uint16
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
		if lhsVal == rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func NotEqualU16(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, uint16](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, uint16](rhs.Data)
	outData := output.Data

	var lhsLast, rhsLast uint16
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
		if lhsVal != rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func AddU16(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, uint16](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, uint16](rhs.Data)
	outData := xunsafe.CastSlice[uint8, uint16](output.Data)

	var lhsLast, rhsLast uint16
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
		outData[i] = lhsVal + rhsVal
	}
}

func SubtractU16(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, uint16](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, uint16](rhs.Data)
	outData := xunsafe.CastSlice[uint8, uint16](output.Data)

	var lhsLast, rhsLast uint16
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
		outData[i] = lhsVal - rhsVal
	}
}

func MultiplyU16(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, uint16](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, uint16](rhs.Data)
	outData := xunsafe.CastSlice[uint8, uint16](output.Data)

	var lhsLast, rhsLast uint16
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
		outData[i] = lhsVal * rhsVal
	}
}

func DivideU16(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, uint16](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, uint16](rhs.Data)
	outData := xunsafe.CastSlice[uint8, uint16](output.Data)

	var lhsLast, rhsLast uint16
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
		outData[i] = lhsVal / rhsVal
	}
}

func GreaterThanU8(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, uint8](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, uint8](rhs.Data)
	outData := output.Data

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
		if lhsVal > rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func GreaterThanOrEqualU8(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, uint8](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, uint8](rhs.Data)
	outData := output.Data

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
		if lhsVal >= rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func LessThanU8(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, uint8](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, uint8](rhs.Data)
	outData := output.Data

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
		if lhsVal < rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func LessThanOrEqualU8(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, uint8](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, uint8](rhs.Data)
	outData := output.Data

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
		if lhsVal <= rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func EqualU8(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, uint8](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, uint8](rhs.Data)
	outData := output.Data

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
		if lhsVal == rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func NotEqualU8(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, uint8](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, uint8](rhs.Data)
	outData := output.Data

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
		if lhsVal != rhsVal {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func AddU8(lhs, rhs telem.Series, output *telem.Series) {
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
		outData[i] = lhsVal + rhsVal
	}
}

func SubtractU8(lhs, rhs telem.Series, output *telem.Series) {
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
		outData[i] = lhsVal - rhsVal
	}
}

func MultiplyU8(lhs, rhs telem.Series, output *telem.Series) {
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
		outData[i] = lhsVal * rhsVal
	}
}

func DivideU8(lhs, rhs telem.Series, output *telem.Series) {
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
		outData[i] = lhsVal / rhsVal
	}
}

func ModuloF64(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, float64](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, float64](rhs.Data)
	outData := xunsafe.CastSlice[uint8, float64](output.Data)

	var lhsLast, rhsLast float64
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
		outData[i] = float64(math.Mod(float64(lhsVal), float64(rhsVal)))
	}
}

func ModuloF32(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, float32](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, float32](rhs.Data)
	outData := xunsafe.CastSlice[uint8, float32](output.Data)

	var lhsLast, rhsLast float32
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
		outData[i] = float32(math.Mod(float64(lhsVal), float64(rhsVal)))
	}
}

func ModuloI64(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, int64](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, int64](rhs.Data)
	outData := xunsafe.CastSlice[uint8, int64](output.Data)

	var lhsLast, rhsLast int64
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
		outData[i] = lhsVal % rhsVal
	}
}

func ModuloI32(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, int32](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, int32](rhs.Data)
	outData := xunsafe.CastSlice[uint8, int32](output.Data)

	var lhsLast, rhsLast int32
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
		outData[i] = lhsVal % rhsVal
	}
}

func ModuloI16(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, int16](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, int16](rhs.Data)
	outData := xunsafe.CastSlice[uint8, int16](output.Data)

	var lhsLast, rhsLast int16
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
		outData[i] = lhsVal % rhsVal
	}
}

func ModuloI8(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, int8](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, int8](rhs.Data)
	outData := xunsafe.CastSlice[uint8, int8](output.Data)

	var lhsLast, rhsLast int8
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
		outData[i] = lhsVal % rhsVal
	}
}

func ModuloU64(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, uint64](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, uint64](rhs.Data)
	outData := xunsafe.CastSlice[uint8, uint64](output.Data)

	var lhsLast, rhsLast uint64
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
		outData[i] = lhsVal % rhsVal
	}
}

func ModuloU32(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, uint32](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, uint32](rhs.Data)
	outData := xunsafe.CastSlice[uint8, uint32](output.Data)

	var lhsLast, rhsLast uint32
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
		outData[i] = lhsVal % rhsVal
	}
}

func ModuloU16(lhs, rhs telem.Series, output *telem.Series) {
	lhsLen := lhs.Len()
	rhsLen := rhs.Len()
	maxLen := lhsLen
	if rhsLen > maxLen {
		maxLen = rhsLen
	}
	output.Resize(maxLen)

	lhsData := xunsafe.CastSlice[uint8, uint16](lhs.Data)
	rhsData := xunsafe.CastSlice[uint8, uint16](rhs.Data)
	outData := xunsafe.CastSlice[uint8, uint16](output.Data)

	var lhsLast, rhsLast uint16
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
		outData[i] = lhsVal % rhsVal
	}
}

func ModuloU8(lhs, rhs telem.Series, output *telem.Series) {
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
		outData[i] = lhsVal % rhsVal
	}
}

func AndU8(lhs, rhs telem.Series, output *telem.Series) {
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

func OrU8(lhs, rhs telem.Series, output *telem.Series) {
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

func NotU8(input telem.Series, output *telem.Series) {
	inputLen := input.Len()
	output.Resize(inputLen)

	inData := xunsafe.CastSlice[uint8, uint8](input.Data)
	outData := xunsafe.CastSlice[uint8, uint8](output.Data)

	for i := int64(0); i < inputLen; i++ {
		outData[i] = ^inData[i]
	}
}

func NegateF64(input telem.Series, output *telem.Series) {
	inputLen := input.Len()
	output.Resize(inputLen)

	inData := xunsafe.CastSlice[uint8, float64](input.Data)
	outData := xunsafe.CastSlice[uint8, float64](output.Data)

	for i := int64(0); i < inputLen; i++ {
		outData[i] = -inData[i]
	}
}

func NegateF32(input telem.Series, output *telem.Series) {
	inputLen := input.Len()
	output.Resize(inputLen)

	inData := xunsafe.CastSlice[uint8, float32](input.Data)
	outData := xunsafe.CastSlice[uint8, float32](output.Data)

	for i := int64(0); i < inputLen; i++ {
		outData[i] = -inData[i]
	}
}

func NegateI64(input telem.Series, output *telem.Series) {
	inputLen := input.Len()
	output.Resize(inputLen)

	inData := xunsafe.CastSlice[uint8, int64](input.Data)
	outData := xunsafe.CastSlice[uint8, int64](output.Data)

	for i := int64(0); i < inputLen; i++ {
		outData[i] = -inData[i]
	}
}

func NegateI32(input telem.Series, output *telem.Series) {
	inputLen := input.Len()
	output.Resize(inputLen)

	inData := xunsafe.CastSlice[uint8, int32](input.Data)
	outData := xunsafe.CastSlice[uint8, int32](output.Data)

	for i := int64(0); i < inputLen; i++ {
		outData[i] = -inData[i]
	}
}

func NegateI16(input telem.Series, output *telem.Series) {
	inputLen := input.Len()
	output.Resize(inputLen)

	inData := xunsafe.CastSlice[uint8, int16](input.Data)
	outData := xunsafe.CastSlice[uint8, int16](output.Data)

	for i := int64(0); i < inputLen; i++ {
		outData[i] = -inData[i]
	}
}

func NegateI8(input telem.Series, output *telem.Series) {
	inputLen := input.Len()
	output.Resize(inputLen)

	inData := xunsafe.CastSlice[uint8, int8](input.Data)
	outData := xunsafe.CastSlice[uint8, int8](output.Data)

	for i := int64(0); i < inputLen; i++ {
		outData[i] = -inData[i]
	}
}

func AvgF64(input telem.Series, prevCount int64, output *telem.Series) int64 {
	inputLen := input.Len()
	if inputLen == 0 {
		return prevCount
	}

	inData := xunsafe.CastSlice[uint8, float64](input.Data)

	// Compute sum of new input samples
	var newSum float64
	for i := int64(0); i < inputLen; i++ {
		newSum += inData[i]
	}

	// Check if we're starting fresh (either no previous samples or output was reset)
	outputLen := output.Len()
	freshStart := prevCount == 0 || outputLen == 0
	output.Resize(1)
	outData := xunsafe.CastSlice[uint8, float64](output.Data)

	if freshStart {
		// Fresh start: compute average of input samples
		outData[0] = newSum / float64(inputLen)
	} else {
		// Weighted average: combine previous average with new samples
		prevAvg := outData[0]
		totalCount := prevCount + inputLen
		outData[0] = (prevAvg*float64(prevCount) + newSum) / float64(totalCount)
	}

	return prevCount + inputLen

}

func MinF64(input telem.Series, prevCount int64, output *telem.Series) int64 {
	inputLen := input.Len()
	if inputLen == 0 {
		return prevCount
	}

	inData := xunsafe.CastSlice[uint8, float64](input.Data)

	// Check if we're starting fresh (either no previous samples or output was reset)
	outputLen := output.Len()
	freshStart := prevCount == 0 || outputLen == 0
	output.Resize(1)
	outData := xunsafe.CastSlice[uint8, float64](output.Data)

	// Find minimum in new input samples
	newMin := inData[0]
	for i := int64(1); i < inputLen; i++ {
		if inData[i] < newMin {
			newMin = inData[i]
		}
	}

	if freshStart {
		// Fresh start
		outData[0] = newMin
	} else {
		// Compare with previous minimum
		if newMin < outData[0] {
			outData[0] = newMin
		}
	}

	return prevCount + inputLen

}

func MaxF64(input telem.Series, prevCount int64, output *telem.Series) int64 {
	inputLen := input.Len()
	if inputLen == 0 {
		return prevCount
	}

	inData := xunsafe.CastSlice[uint8, float64](input.Data)

	// Check if we're starting fresh (either no previous samples or output was reset)
	outputLen := output.Len()
	freshStart := prevCount == 0 || outputLen == 0
	output.Resize(1)
	outData := xunsafe.CastSlice[uint8, float64](output.Data)

	// Find maximum in new input samples
	newMax := inData[0]
	for i := int64(1); i < inputLen; i++ {
		if inData[i] > newMax {
			newMax = inData[i]
		}
	}

	if freshStart {
		// Fresh start
		outData[0] = newMax
	} else {
		// Compare with previous maximum
		if newMax > outData[0] {
			outData[0] = newMax
		}
	}

	return prevCount + inputLen

}

func AvgF32(input telem.Series, prevCount int64, output *telem.Series) int64 {
	inputLen := input.Len()
	if inputLen == 0 {
		return prevCount
	}

	inData := xunsafe.CastSlice[uint8, float32](input.Data)

	// Compute sum of new input samples
	var newSum float32
	for i := int64(0); i < inputLen; i++ {
		newSum += inData[i]
	}

	// Check if we're starting fresh (either no previous samples or output was reset)
	outputLen := output.Len()
	freshStart := prevCount == 0 || outputLen == 0
	output.Resize(1)
	outData := xunsafe.CastSlice[uint8, float32](output.Data)

	if freshStart {
		// Fresh start: compute average of input samples
		outData[0] = newSum / float32(inputLen)
	} else {
		// Weighted average: combine previous average with new samples
		prevAvg := outData[0]
		totalCount := prevCount + inputLen
		outData[0] = (prevAvg*float32(prevCount) + newSum) / float32(totalCount)
	}

	return prevCount + inputLen

}

func MinF32(input telem.Series, prevCount int64, output *telem.Series) int64 {
	inputLen := input.Len()
	if inputLen == 0 {
		return prevCount
	}

	inData := xunsafe.CastSlice[uint8, float32](input.Data)

	// Check if we're starting fresh (either no previous samples or output was reset)
	outputLen := output.Len()
	freshStart := prevCount == 0 || outputLen == 0
	output.Resize(1)
	outData := xunsafe.CastSlice[uint8, float32](output.Data)

	// Find minimum in new input samples
	newMin := inData[0]
	for i := int64(1); i < inputLen; i++ {
		if inData[i] < newMin {
			newMin = inData[i]
		}
	}

	if freshStart {
		// Fresh start
		outData[0] = newMin
	} else {
		// Compare with previous minimum
		if newMin < outData[0] {
			outData[0] = newMin
		}
	}

	return prevCount + inputLen

}

func MaxF32(input telem.Series, prevCount int64, output *telem.Series) int64 {
	inputLen := input.Len()
	if inputLen == 0 {
		return prevCount
	}

	inData := xunsafe.CastSlice[uint8, float32](input.Data)

	// Check if we're starting fresh (either no previous samples or output was reset)
	outputLen := output.Len()
	freshStart := prevCount == 0 || outputLen == 0
	output.Resize(1)
	outData := xunsafe.CastSlice[uint8, float32](output.Data)

	// Find maximum in new input samples
	newMax := inData[0]
	for i := int64(1); i < inputLen; i++ {
		if inData[i] > newMax {
			newMax = inData[i]
		}
	}

	if freshStart {
		// Fresh start
		outData[0] = newMax
	} else {
		// Compare with previous maximum
		if newMax > outData[0] {
			outData[0] = newMax
		}
	}

	return prevCount + inputLen

}

func AvgI64(input telem.Series, prevCount int64, output *telem.Series) int64 {
	inputLen := input.Len()
	if inputLen == 0 {
		return prevCount
	}

	inData := xunsafe.CastSlice[uint8, int64](input.Data)

	// Compute sum of new input samples
	var newSum int64
	for i := int64(0); i < inputLen; i++ {
		newSum += inData[i]
	}

	// Check if we're starting fresh (either no previous samples or output was reset)
	outputLen := output.Len()
	freshStart := prevCount == 0 || outputLen == 0
	output.Resize(1)
	outData := xunsafe.CastSlice[uint8, int64](output.Data)

	if freshStart {
		// Fresh start: compute average of input samples
		outData[0] = newSum / int64(inputLen)
	} else {
		// Weighted average: combine previous average with new samples
		prevAvg := outData[0]
		totalCount := prevCount + inputLen
		outData[0] = (prevAvg*int64(prevCount) + newSum) / int64(totalCount)
	}

	return prevCount + inputLen

}

func MinI64(input telem.Series, prevCount int64, output *telem.Series) int64 {
	inputLen := input.Len()
	if inputLen == 0 {
		return prevCount
	}

	inData := xunsafe.CastSlice[uint8, int64](input.Data)

	// Check if we're starting fresh (either no previous samples or output was reset)
	outputLen := output.Len()
	freshStart := prevCount == 0 || outputLen == 0
	output.Resize(1)
	outData := xunsafe.CastSlice[uint8, int64](output.Data)

	// Find minimum in new input samples
	newMin := inData[0]
	for i := int64(1); i < inputLen; i++ {
		if inData[i] < newMin {
			newMin = inData[i]
		}
	}

	if freshStart {
		// Fresh start
		outData[0] = newMin
	} else {
		// Compare with previous minimum
		if newMin < outData[0] {
			outData[0] = newMin
		}
	}

	return prevCount + inputLen

}

func MaxI64(input telem.Series, prevCount int64, output *telem.Series) int64 {
	inputLen := input.Len()
	if inputLen == 0 {
		return prevCount
	}

	inData := xunsafe.CastSlice[uint8, int64](input.Data)

	// Check if we're starting fresh (either no previous samples or output was reset)
	outputLen := output.Len()
	freshStart := prevCount == 0 || outputLen == 0
	output.Resize(1)
	outData := xunsafe.CastSlice[uint8, int64](output.Data)

	// Find maximum in new input samples
	newMax := inData[0]
	for i := int64(1); i < inputLen; i++ {
		if inData[i] > newMax {
			newMax = inData[i]
		}
	}

	if freshStart {
		// Fresh start
		outData[0] = newMax
	} else {
		// Compare with previous maximum
		if newMax > outData[0] {
			outData[0] = newMax
		}
	}

	return prevCount + inputLen

}

func AvgI32(input telem.Series, prevCount int64, output *telem.Series) int64 {
	inputLen := input.Len()
	if inputLen == 0 {
		return prevCount
	}

	inData := xunsafe.CastSlice[uint8, int32](input.Data)

	// Compute sum of new input samples
	var newSum int32
	for i := int64(0); i < inputLen; i++ {
		newSum += inData[i]
	}

	// Check if we're starting fresh (either no previous samples or output was reset)
	outputLen := output.Len()
	freshStart := prevCount == 0 || outputLen == 0
	output.Resize(1)
	outData := xunsafe.CastSlice[uint8, int32](output.Data)

	if freshStart {
		// Fresh start: compute average of input samples
		outData[0] = newSum / int32(inputLen)
	} else {
		// Weighted average: combine previous average with new samples
		prevAvg := outData[0]
		totalCount := prevCount + inputLen
		outData[0] = (prevAvg*int32(prevCount) + newSum) / int32(totalCount)
	}

	return prevCount + inputLen

}

func MinI32(input telem.Series, prevCount int64, output *telem.Series) int64 {
	inputLen := input.Len()
	if inputLen == 0 {
		return prevCount
	}

	inData := xunsafe.CastSlice[uint8, int32](input.Data)

	// Check if we're starting fresh (either no previous samples or output was reset)
	outputLen := output.Len()
	freshStart := prevCount == 0 || outputLen == 0
	output.Resize(1)
	outData := xunsafe.CastSlice[uint8, int32](output.Data)

	// Find minimum in new input samples
	newMin := inData[0]
	for i := int64(1); i < inputLen; i++ {
		if inData[i] < newMin {
			newMin = inData[i]
		}
	}

	if freshStart {
		// Fresh start
		outData[0] = newMin
	} else {
		// Compare with previous minimum
		if newMin < outData[0] {
			outData[0] = newMin
		}
	}

	return prevCount + inputLen

}

func MaxI32(input telem.Series, prevCount int64, output *telem.Series) int64 {
	inputLen := input.Len()
	if inputLen == 0 {
		return prevCount
	}

	inData := xunsafe.CastSlice[uint8, int32](input.Data)

	// Check if we're starting fresh (either no previous samples or output was reset)
	outputLen := output.Len()
	freshStart := prevCount == 0 || outputLen == 0
	output.Resize(1)
	outData := xunsafe.CastSlice[uint8, int32](output.Data)

	// Find maximum in new input samples
	newMax := inData[0]
	for i := int64(1); i < inputLen; i++ {
		if inData[i] > newMax {
			newMax = inData[i]
		}
	}

	if freshStart {
		// Fresh start
		outData[0] = newMax
	} else {
		// Compare with previous maximum
		if newMax > outData[0] {
			outData[0] = newMax
		}
	}

	return prevCount + inputLen

}

func AvgI16(input telem.Series, prevCount int64, output *telem.Series) int64 {
	inputLen := input.Len()
	if inputLen == 0 {
		return prevCount
	}

	inData := xunsafe.CastSlice[uint8, int16](input.Data)

	// Compute sum of new input samples
	var newSum int16
	for i := int64(0); i < inputLen; i++ {
		newSum += inData[i]
	}

	// Check if we're starting fresh (either no previous samples or output was reset)
	outputLen := output.Len()
	freshStart := prevCount == 0 || outputLen == 0
	output.Resize(1)
	outData := xunsafe.CastSlice[uint8, int16](output.Data)

	if freshStart {
		// Fresh start: compute average of input samples
		outData[0] = newSum / int16(inputLen)
	} else {
		// Weighted average: combine previous average with new samples
		prevAvg := outData[0]
		totalCount := prevCount + inputLen
		outData[0] = (prevAvg*int16(prevCount) + newSum) / int16(totalCount)
	}

	return prevCount + inputLen

}

func MinI16(input telem.Series, prevCount int64, output *telem.Series) int64 {
	inputLen := input.Len()
	if inputLen == 0 {
		return prevCount
	}

	inData := xunsafe.CastSlice[uint8, int16](input.Data)

	// Check if we're starting fresh (either no previous samples or output was reset)
	outputLen := output.Len()
	freshStart := prevCount == 0 || outputLen == 0
	output.Resize(1)
	outData := xunsafe.CastSlice[uint8, int16](output.Data)

	// Find minimum in new input samples
	newMin := inData[0]
	for i := int64(1); i < inputLen; i++ {
		if inData[i] < newMin {
			newMin = inData[i]
		}
	}

	if freshStart {
		// Fresh start
		outData[0] = newMin
	} else {
		// Compare with previous minimum
		if newMin < outData[0] {
			outData[0] = newMin
		}
	}

	return prevCount + inputLen

}

func MaxI16(input telem.Series, prevCount int64, output *telem.Series) int64 {
	inputLen := input.Len()
	if inputLen == 0 {
		return prevCount
	}

	inData := xunsafe.CastSlice[uint8, int16](input.Data)

	// Check if we're starting fresh (either no previous samples or output was reset)
	outputLen := output.Len()
	freshStart := prevCount == 0 || outputLen == 0
	output.Resize(1)
	outData := xunsafe.CastSlice[uint8, int16](output.Data)

	// Find maximum in new input samples
	newMax := inData[0]
	for i := int64(1); i < inputLen; i++ {
		if inData[i] > newMax {
			newMax = inData[i]
		}
	}

	if freshStart {
		// Fresh start
		outData[0] = newMax
	} else {
		// Compare with previous maximum
		if newMax > outData[0] {
			outData[0] = newMax
		}
	}

	return prevCount + inputLen

}

func AvgI8(input telem.Series, prevCount int64, output *telem.Series) int64 {
	inputLen := input.Len()
	if inputLen == 0 {
		return prevCount
	}

	inData := xunsafe.CastSlice[uint8, int8](input.Data)

	// Compute sum of new input samples
	var newSum int8
	for i := int64(0); i < inputLen; i++ {
		newSum += inData[i]
	}

	// Check if we're starting fresh (either no previous samples or output was reset)
	outputLen := output.Len()
	freshStart := prevCount == 0 || outputLen == 0
	output.Resize(1)
	outData := xunsafe.CastSlice[uint8, int8](output.Data)

	if freshStart {
		// Fresh start: compute average of input samples
		outData[0] = newSum / int8(inputLen)
	} else {
		// Weighted average: combine previous average with new samples
		prevAvg := outData[0]
		totalCount := prevCount + inputLen
		outData[0] = (prevAvg*int8(prevCount) + newSum) / int8(totalCount)
	}

	return prevCount + inputLen

}

func MinI8(input telem.Series, prevCount int64, output *telem.Series) int64 {
	inputLen := input.Len()
	if inputLen == 0 {
		return prevCount
	}

	inData := xunsafe.CastSlice[uint8, int8](input.Data)

	// Check if we're starting fresh (either no previous samples or output was reset)
	outputLen := output.Len()
	freshStart := prevCount == 0 || outputLen == 0
	output.Resize(1)
	outData := xunsafe.CastSlice[uint8, int8](output.Data)

	// Find minimum in new input samples
	newMin := inData[0]
	for i := int64(1); i < inputLen; i++ {
		if inData[i] < newMin {
			newMin = inData[i]
		}
	}

	if freshStart {
		// Fresh start
		outData[0] = newMin
	} else {
		// Compare with previous minimum
		if newMin < outData[0] {
			outData[0] = newMin
		}
	}

	return prevCount + inputLen

}

func MaxI8(input telem.Series, prevCount int64, output *telem.Series) int64 {
	inputLen := input.Len()
	if inputLen == 0 {
		return prevCount
	}

	inData := xunsafe.CastSlice[uint8, int8](input.Data)

	// Check if we're starting fresh (either no previous samples or output was reset)
	outputLen := output.Len()
	freshStart := prevCount == 0 || outputLen == 0
	output.Resize(1)
	outData := xunsafe.CastSlice[uint8, int8](output.Data)

	// Find maximum in new input samples
	newMax := inData[0]
	for i := int64(1); i < inputLen; i++ {
		if inData[i] > newMax {
			newMax = inData[i]
		}
	}

	if freshStart {
		// Fresh start
		outData[0] = newMax
	} else {
		// Compare with previous maximum
		if newMax > outData[0] {
			outData[0] = newMax
		}
	}

	return prevCount + inputLen

}

func AvgU64(input telem.Series, prevCount int64, output *telem.Series) int64 {
	inputLen := input.Len()
	if inputLen == 0 {
		return prevCount
	}

	inData := xunsafe.CastSlice[uint8, uint64](input.Data)

	// Compute sum of new input samples
	var newSum uint64
	for i := int64(0); i < inputLen; i++ {
		newSum += inData[i]
	}

	// Check if we're starting fresh (either no previous samples or output was reset)
	outputLen := output.Len()
	freshStart := prevCount == 0 || outputLen == 0
	output.Resize(1)
	outData := xunsafe.CastSlice[uint8, uint64](output.Data)

	if freshStart {
		// Fresh start: compute average of input samples
		outData[0] = newSum / uint64(inputLen)
	} else {
		// Weighted average: combine previous average with new samples
		prevAvg := outData[0]
		totalCount := prevCount + inputLen
		outData[0] = (prevAvg*uint64(prevCount) + newSum) / uint64(totalCount)
	}

	return prevCount + inputLen

}

func MinU64(input telem.Series, prevCount int64, output *telem.Series) int64 {
	inputLen := input.Len()
	if inputLen == 0 {
		return prevCount
	}

	inData := xunsafe.CastSlice[uint8, uint64](input.Data)

	// Check if we're starting fresh (either no previous samples or output was reset)
	outputLen := output.Len()
	freshStart := prevCount == 0 || outputLen == 0
	output.Resize(1)
	outData := xunsafe.CastSlice[uint8, uint64](output.Data)

	// Find minimum in new input samples
	newMin := inData[0]
	for i := int64(1); i < inputLen; i++ {
		if inData[i] < newMin {
			newMin = inData[i]
		}
	}

	if freshStart {
		// Fresh start
		outData[0] = newMin
	} else {
		// Compare with previous minimum
		if newMin < outData[0] {
			outData[0] = newMin
		}
	}

	return prevCount + inputLen

}

func MaxU64(input telem.Series, prevCount int64, output *telem.Series) int64 {
	inputLen := input.Len()
	if inputLen == 0 {
		return prevCount
	}

	inData := xunsafe.CastSlice[uint8, uint64](input.Data)

	// Check if we're starting fresh (either no previous samples or output was reset)
	outputLen := output.Len()
	freshStart := prevCount == 0 || outputLen == 0
	output.Resize(1)
	outData := xunsafe.CastSlice[uint8, uint64](output.Data)

	// Find maximum in new input samples
	newMax := inData[0]
	for i := int64(1); i < inputLen; i++ {
		if inData[i] > newMax {
			newMax = inData[i]
		}
	}

	if freshStart {
		// Fresh start
		outData[0] = newMax
	} else {
		// Compare with previous maximum
		if newMax > outData[0] {
			outData[0] = newMax
		}
	}

	return prevCount + inputLen

}

func AvgU32(input telem.Series, prevCount int64, output *telem.Series) int64 {
	inputLen := input.Len()
	if inputLen == 0 {
		return prevCount
	}

	inData := xunsafe.CastSlice[uint8, uint32](input.Data)

	// Compute sum of new input samples
	var newSum uint32
	for i := int64(0); i < inputLen; i++ {
		newSum += inData[i]
	}

	// Check if we're starting fresh (either no previous samples or output was reset)
	outputLen := output.Len()
	freshStart := prevCount == 0 || outputLen == 0
	output.Resize(1)
	outData := xunsafe.CastSlice[uint8, uint32](output.Data)

	if freshStart {
		// Fresh start: compute average of input samples
		outData[0] = newSum / uint32(inputLen)
	} else {
		// Weighted average: combine previous average with new samples
		prevAvg := outData[0]
		totalCount := prevCount + inputLen
		outData[0] = (prevAvg*uint32(prevCount) + newSum) / uint32(totalCount)
	}

	return prevCount + inputLen

}

func MinU32(input telem.Series, prevCount int64, output *telem.Series) int64 {
	inputLen := input.Len()
	if inputLen == 0 {
		return prevCount
	}

	inData := xunsafe.CastSlice[uint8, uint32](input.Data)

	// Check if we're starting fresh (either no previous samples or output was reset)
	outputLen := output.Len()
	freshStart := prevCount == 0 || outputLen == 0
	output.Resize(1)
	outData := xunsafe.CastSlice[uint8, uint32](output.Data)

	// Find minimum in new input samples
	newMin := inData[0]
	for i := int64(1); i < inputLen; i++ {
		if inData[i] < newMin {
			newMin = inData[i]
		}
	}

	if freshStart {
		// Fresh start
		outData[0] = newMin
	} else {
		// Compare with previous minimum
		if newMin < outData[0] {
			outData[0] = newMin
		}
	}

	return prevCount + inputLen

}

func MaxU32(input telem.Series, prevCount int64, output *telem.Series) int64 {
	inputLen := input.Len()
	if inputLen == 0 {
		return prevCount
	}

	inData := xunsafe.CastSlice[uint8, uint32](input.Data)

	// Check if we're starting fresh (either no previous samples or output was reset)
	outputLen := output.Len()
	freshStart := prevCount == 0 || outputLen == 0
	output.Resize(1)
	outData := xunsafe.CastSlice[uint8, uint32](output.Data)

	// Find maximum in new input samples
	newMax := inData[0]
	for i := int64(1); i < inputLen; i++ {
		if inData[i] > newMax {
			newMax = inData[i]
		}
	}

	if freshStart {
		// Fresh start
		outData[0] = newMax
	} else {
		// Compare with previous maximum
		if newMax > outData[0] {
			outData[0] = newMax
		}
	}

	return prevCount + inputLen

}

func AvgU16(input telem.Series, prevCount int64, output *telem.Series) int64 {
	inputLen := input.Len()
	if inputLen == 0 {
		return prevCount
	}

	inData := xunsafe.CastSlice[uint8, uint16](input.Data)

	// Compute sum of new input samples
	var newSum uint16
	for i := int64(0); i < inputLen; i++ {
		newSum += inData[i]
	}

	// Check if we're starting fresh (either no previous samples or output was reset)
	outputLen := output.Len()
	freshStart := prevCount == 0 || outputLen == 0
	output.Resize(1)
	outData := xunsafe.CastSlice[uint8, uint16](output.Data)

	if freshStart {
		// Fresh start: compute average of input samples
		outData[0] = newSum / uint16(inputLen)
	} else {
		// Weighted average: combine previous average with new samples
		prevAvg := outData[0]
		totalCount := prevCount + inputLen
		outData[0] = (prevAvg*uint16(prevCount) + newSum) / uint16(totalCount)
	}

	return prevCount + inputLen

}

func MinU16(input telem.Series, prevCount int64, output *telem.Series) int64 {
	inputLen := input.Len()
	if inputLen == 0 {
		return prevCount
	}

	inData := xunsafe.CastSlice[uint8, uint16](input.Data)

	// Check if we're starting fresh (either no previous samples or output was reset)
	outputLen := output.Len()
	freshStart := prevCount == 0 || outputLen == 0
	output.Resize(1)
	outData := xunsafe.CastSlice[uint8, uint16](output.Data)

	// Find minimum in new input samples
	newMin := inData[0]
	for i := int64(1); i < inputLen; i++ {
		if inData[i] < newMin {
			newMin = inData[i]
		}
	}

	if freshStart {
		// Fresh start
		outData[0] = newMin
	} else {
		// Compare with previous minimum
		if newMin < outData[0] {
			outData[0] = newMin
		}
	}

	return prevCount + inputLen

}

func MaxU16(input telem.Series, prevCount int64, output *telem.Series) int64 {
	inputLen := input.Len()
	if inputLen == 0 {
		return prevCount
	}

	inData := xunsafe.CastSlice[uint8, uint16](input.Data)

	// Check if we're starting fresh (either no previous samples or output was reset)
	outputLen := output.Len()
	freshStart := prevCount == 0 || outputLen == 0
	output.Resize(1)
	outData := xunsafe.CastSlice[uint8, uint16](output.Data)

	// Find maximum in new input samples
	newMax := inData[0]
	for i := int64(1); i < inputLen; i++ {
		if inData[i] > newMax {
			newMax = inData[i]
		}
	}

	if freshStart {
		// Fresh start
		outData[0] = newMax
	} else {
		// Compare with previous maximum
		if newMax > outData[0] {
			outData[0] = newMax
		}
	}

	return prevCount + inputLen

}

func AvgU8(input telem.Series, prevCount int64, output *telem.Series) int64 {
	inputLen := input.Len()
	if inputLen == 0 {
		return prevCount
	}

	inData := xunsafe.CastSlice[uint8, uint8](input.Data)

	// Compute sum of new input samples
	var newSum uint8
	for i := int64(0); i < inputLen; i++ {
		newSum += inData[i]
	}

	// Check if we're starting fresh (either no previous samples or output was reset)
	outputLen := output.Len()
	freshStart := prevCount == 0 || outputLen == 0
	output.Resize(1)
	outData := xunsafe.CastSlice[uint8, uint8](output.Data)

	if freshStart {
		// Fresh start: compute average of input samples
		outData[0] = newSum / uint8(inputLen)
	} else {
		// Weighted average: combine previous average with new samples
		prevAvg := outData[0]
		totalCount := prevCount + inputLen
		outData[0] = (prevAvg*uint8(prevCount) + newSum) / uint8(totalCount)
	}

	return prevCount + inputLen

}

func MinU8(input telem.Series, prevCount int64, output *telem.Series) int64 {
	inputLen := input.Len()
	if inputLen == 0 {
		return prevCount
	}

	inData := xunsafe.CastSlice[uint8, uint8](input.Data)

	// Check if we're starting fresh (either no previous samples or output was reset)
	outputLen := output.Len()
	freshStart := prevCount == 0 || outputLen == 0
	output.Resize(1)
	outData := xunsafe.CastSlice[uint8, uint8](output.Data)

	// Find minimum in new input samples
	newMin := inData[0]
	for i := int64(1); i < inputLen; i++ {
		if inData[i] < newMin {
			newMin = inData[i]
		}
	}

	if freshStart {
		// Fresh start
		outData[0] = newMin
	} else {
		// Compare with previous minimum
		if newMin < outData[0] {
			outData[0] = newMin
		}
	}

	return prevCount + inputLen

}

func MaxU8(input telem.Series, prevCount int64, output *telem.Series) int64 {
	inputLen := input.Len()
	if inputLen == 0 {
		return prevCount
	}

	inData := xunsafe.CastSlice[uint8, uint8](input.Data)

	// Check if we're starting fresh (either no previous samples or output was reset)
	outputLen := output.Len()
	freshStart := prevCount == 0 || outputLen == 0
	output.Resize(1)
	outData := xunsafe.CastSlice[uint8, uint8](output.Data)

	// Find maximum in new input samples
	newMax := inData[0]
	for i := int64(1); i < inputLen; i++ {
		if inData[i] > newMax {
			newMax = inData[i]
		}
	}

	if freshStart {
		// Fresh start
		outData[0] = newMax
	} else {
		// Compare with previous maximum
		if newMax > outData[0] {
			outData[0] = newMax
		}
	}

	return prevCount + inputLen

}

func AddScalarF64(series telem.Series, scalar float64, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, float64](series.Data)
	outData := xunsafe.CastSlice[uint8, float64](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] + scalar
	}
}

func SubtractScalarF64(series telem.Series, scalar float64, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, float64](series.Data)
	outData := xunsafe.CastSlice[uint8, float64](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] - scalar
	}
}

func MultiplyScalarF64(series telem.Series, scalar float64, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, float64](series.Data)
	outData := xunsafe.CastSlice[uint8, float64](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] * scalar
	}
}

func DivideScalarF64(series telem.Series, scalar float64, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, float64](series.Data)
	outData := xunsafe.CastSlice[uint8, float64](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] / scalar
	}
}

func AddScalarF32(series telem.Series, scalar float32, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, float32](series.Data)
	outData := xunsafe.CastSlice[uint8, float32](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] + scalar
	}
}

func SubtractScalarF32(series telem.Series, scalar float32, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, float32](series.Data)
	outData := xunsafe.CastSlice[uint8, float32](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] - scalar
	}
}

func MultiplyScalarF32(series telem.Series, scalar float32, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, float32](series.Data)
	outData := xunsafe.CastSlice[uint8, float32](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] * scalar
	}
}

func DivideScalarF32(series telem.Series, scalar float32, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, float32](series.Data)
	outData := xunsafe.CastSlice[uint8, float32](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] / scalar
	}
}

func AddScalarI64(series telem.Series, scalar int64, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int64](series.Data)
	outData := xunsafe.CastSlice[uint8, int64](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] + scalar
	}
}

func SubtractScalarI64(series telem.Series, scalar int64, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int64](series.Data)
	outData := xunsafe.CastSlice[uint8, int64](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] - scalar
	}
}

func MultiplyScalarI64(series telem.Series, scalar int64, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int64](series.Data)
	outData := xunsafe.CastSlice[uint8, int64](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] * scalar
	}
}

func DivideScalarI64(series telem.Series, scalar int64, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int64](series.Data)
	outData := xunsafe.CastSlice[uint8, int64](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] / scalar
	}
}

func AddScalarI32(series telem.Series, scalar int32, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int32](series.Data)
	outData := xunsafe.CastSlice[uint8, int32](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] + scalar
	}
}

func SubtractScalarI32(series telem.Series, scalar int32, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int32](series.Data)
	outData := xunsafe.CastSlice[uint8, int32](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] - scalar
	}
}

func MultiplyScalarI32(series telem.Series, scalar int32, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int32](series.Data)
	outData := xunsafe.CastSlice[uint8, int32](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] * scalar
	}
}

func DivideScalarI32(series telem.Series, scalar int32, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int32](series.Data)
	outData := xunsafe.CastSlice[uint8, int32](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] / scalar
	}
}

func AddScalarI16(series telem.Series, scalar int16, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int16](series.Data)
	outData := xunsafe.CastSlice[uint8, int16](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] + scalar
	}
}

func SubtractScalarI16(series telem.Series, scalar int16, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int16](series.Data)
	outData := xunsafe.CastSlice[uint8, int16](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] - scalar
	}
}

func MultiplyScalarI16(series telem.Series, scalar int16, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int16](series.Data)
	outData := xunsafe.CastSlice[uint8, int16](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] * scalar
	}
}

func DivideScalarI16(series telem.Series, scalar int16, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int16](series.Data)
	outData := xunsafe.CastSlice[uint8, int16](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] / scalar
	}
}

func AddScalarI8(series telem.Series, scalar int8, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int8](series.Data)
	outData := xunsafe.CastSlice[uint8, int8](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] + scalar
	}
}

func SubtractScalarI8(series telem.Series, scalar int8, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int8](series.Data)
	outData := xunsafe.CastSlice[uint8, int8](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] - scalar
	}
}

func MultiplyScalarI8(series telem.Series, scalar int8, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int8](series.Data)
	outData := xunsafe.CastSlice[uint8, int8](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] * scalar
	}
}

func DivideScalarI8(series telem.Series, scalar int8, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int8](series.Data)
	outData := xunsafe.CastSlice[uint8, int8](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] / scalar
	}
}

func AddScalarU64(series telem.Series, scalar uint64, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint64](series.Data)
	outData := xunsafe.CastSlice[uint8, uint64](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] + scalar
	}
}

func SubtractScalarU64(series telem.Series, scalar uint64, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint64](series.Data)
	outData := xunsafe.CastSlice[uint8, uint64](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] - scalar
	}
}

func MultiplyScalarU64(series telem.Series, scalar uint64, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint64](series.Data)
	outData := xunsafe.CastSlice[uint8, uint64](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] * scalar
	}
}

func DivideScalarU64(series telem.Series, scalar uint64, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint64](series.Data)
	outData := xunsafe.CastSlice[uint8, uint64](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] / scalar
	}
}

func AddScalarU32(series telem.Series, scalar uint32, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint32](series.Data)
	outData := xunsafe.CastSlice[uint8, uint32](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] + scalar
	}
}

func SubtractScalarU32(series telem.Series, scalar uint32, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint32](series.Data)
	outData := xunsafe.CastSlice[uint8, uint32](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] - scalar
	}
}

func MultiplyScalarU32(series telem.Series, scalar uint32, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint32](series.Data)
	outData := xunsafe.CastSlice[uint8, uint32](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] * scalar
	}
}

func DivideScalarU32(series telem.Series, scalar uint32, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint32](series.Data)
	outData := xunsafe.CastSlice[uint8, uint32](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] / scalar
	}
}

func AddScalarU16(series telem.Series, scalar uint16, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint16](series.Data)
	outData := xunsafe.CastSlice[uint8, uint16](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] + scalar
	}
}

func SubtractScalarU16(series telem.Series, scalar uint16, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint16](series.Data)
	outData := xunsafe.CastSlice[uint8, uint16](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] - scalar
	}
}

func MultiplyScalarU16(series telem.Series, scalar uint16, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint16](series.Data)
	outData := xunsafe.CastSlice[uint8, uint16](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] * scalar
	}
}

func DivideScalarU16(series telem.Series, scalar uint16, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint16](series.Data)
	outData := xunsafe.CastSlice[uint8, uint16](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] / scalar
	}
}

func AddScalarU8(series telem.Series, scalar uint8, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint8](series.Data)
	outData := xunsafe.CastSlice[uint8, uint8](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] + scalar
	}
}

func SubtractScalarU8(series telem.Series, scalar uint8, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint8](series.Data)
	outData := xunsafe.CastSlice[uint8, uint8](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] - scalar
	}
}

func MultiplyScalarU8(series telem.Series, scalar uint8, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint8](series.Data)
	outData := xunsafe.CastSlice[uint8, uint8](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] * scalar
	}
}

func DivideScalarU8(series telem.Series, scalar uint8, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint8](series.Data)
	outData := xunsafe.CastSlice[uint8, uint8](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] / scalar
	}
}

func ReverseSubtractScalarF64(series telem.Series, scalar float64, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, float64](series.Data)
	outData := xunsafe.CastSlice[uint8, float64](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = scalar - inData[i]
	}
}

func ReverseDivideScalarF64(series telem.Series, scalar float64, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, float64](series.Data)
	outData := xunsafe.CastSlice[uint8, float64](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = scalar / inData[i]
	}
}

func ReverseSubtractScalarF32(series telem.Series, scalar float32, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, float32](series.Data)
	outData := xunsafe.CastSlice[uint8, float32](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = scalar - inData[i]
	}
}

func ReverseDivideScalarF32(series telem.Series, scalar float32, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, float32](series.Data)
	outData := xunsafe.CastSlice[uint8, float32](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = scalar / inData[i]
	}
}

func ReverseSubtractScalarI64(series telem.Series, scalar int64, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int64](series.Data)
	outData := xunsafe.CastSlice[uint8, int64](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = scalar - inData[i]
	}
}

func ReverseDivideScalarI64(series telem.Series, scalar int64, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int64](series.Data)
	outData := xunsafe.CastSlice[uint8, int64](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = scalar / inData[i]
	}
}

func ReverseSubtractScalarI32(series telem.Series, scalar int32, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int32](series.Data)
	outData := xunsafe.CastSlice[uint8, int32](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = scalar - inData[i]
	}
}

func ReverseDivideScalarI32(series telem.Series, scalar int32, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int32](series.Data)
	outData := xunsafe.CastSlice[uint8, int32](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = scalar / inData[i]
	}
}

func ReverseSubtractScalarI16(series telem.Series, scalar int16, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int16](series.Data)
	outData := xunsafe.CastSlice[uint8, int16](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = scalar - inData[i]
	}
}

func ReverseDivideScalarI16(series telem.Series, scalar int16, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int16](series.Data)
	outData := xunsafe.CastSlice[uint8, int16](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = scalar / inData[i]
	}
}

func ReverseSubtractScalarI8(series telem.Series, scalar int8, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int8](series.Data)
	outData := xunsafe.CastSlice[uint8, int8](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = scalar - inData[i]
	}
}

func ReverseDivideScalarI8(series telem.Series, scalar int8, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int8](series.Data)
	outData := xunsafe.CastSlice[uint8, int8](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = scalar / inData[i]
	}
}

func ReverseSubtractScalarU64(series telem.Series, scalar uint64, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint64](series.Data)
	outData := xunsafe.CastSlice[uint8, uint64](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = scalar - inData[i]
	}
}

func ReverseDivideScalarU64(series telem.Series, scalar uint64, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint64](series.Data)
	outData := xunsafe.CastSlice[uint8, uint64](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = scalar / inData[i]
	}
}

func ReverseSubtractScalarU32(series telem.Series, scalar uint32, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint32](series.Data)
	outData := xunsafe.CastSlice[uint8, uint32](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = scalar - inData[i]
	}
}

func ReverseDivideScalarU32(series telem.Series, scalar uint32, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint32](series.Data)
	outData := xunsafe.CastSlice[uint8, uint32](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = scalar / inData[i]
	}
}

func ReverseSubtractScalarU16(series telem.Series, scalar uint16, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint16](series.Data)
	outData := xunsafe.CastSlice[uint8, uint16](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = scalar - inData[i]
	}
}

func ReverseDivideScalarU16(series telem.Series, scalar uint16, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint16](series.Data)
	outData := xunsafe.CastSlice[uint8, uint16](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = scalar / inData[i]
	}
}

func ReverseSubtractScalarU8(series telem.Series, scalar uint8, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint8](series.Data)
	outData := xunsafe.CastSlice[uint8, uint8](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = scalar - inData[i]
	}
}

func ReverseDivideScalarU8(series telem.Series, scalar uint8, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint8](series.Data)
	outData := xunsafe.CastSlice[uint8, uint8](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = scalar / inData[i]
	}
}

func ModuloScalarF64(series telem.Series, scalar float64, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, float64](series.Data)
	outData := xunsafe.CastSlice[uint8, float64](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = float64(math.Mod(float64(inData[i]), float64(scalar)))
	}
}

func ModuloScalarF32(series telem.Series, scalar float32, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, float32](series.Data)
	outData := xunsafe.CastSlice[uint8, float32](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = float32(math.Mod(float64(inData[i]), float64(scalar)))
	}
}

func ModuloScalarI64(series telem.Series, scalar int64, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int64](series.Data)
	outData := xunsafe.CastSlice[uint8, int64](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] % scalar
	}
}

func ModuloScalarI32(series telem.Series, scalar int32, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int32](series.Data)
	outData := xunsafe.CastSlice[uint8, int32](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] % scalar
	}
}

func ModuloScalarI16(series telem.Series, scalar int16, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int16](series.Data)
	outData := xunsafe.CastSlice[uint8, int16](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] % scalar
	}
}

func ModuloScalarI8(series telem.Series, scalar int8, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int8](series.Data)
	outData := xunsafe.CastSlice[uint8, int8](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] % scalar
	}
}

func ModuloScalarU64(series telem.Series, scalar uint64, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint64](series.Data)
	outData := xunsafe.CastSlice[uint8, uint64](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] % scalar
	}
}

func ModuloScalarU32(series telem.Series, scalar uint32, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint32](series.Data)
	outData := xunsafe.CastSlice[uint8, uint32](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] % scalar
	}
}

func ModuloScalarU16(series telem.Series, scalar uint16, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint16](series.Data)
	outData := xunsafe.CastSlice[uint8, uint16](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] % scalar
	}
}

func ModuloScalarU8(series telem.Series, scalar uint8, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint8](series.Data)
	outData := xunsafe.CastSlice[uint8, uint8](output.Data)

	for i := int64(0); i < length; i++ {
		outData[i] = inData[i] % scalar
	}
}

func GreaterThanScalarF64(series telem.Series, scalar float64, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, float64](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] > scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func GreaterThanOrEqualScalarF64(series telem.Series, scalar float64, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, float64](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] >= scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func LessThanScalarF64(series telem.Series, scalar float64, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, float64](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] < scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func LessThanOrEqualScalarF64(series telem.Series, scalar float64, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, float64](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] <= scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func EqualScalarF64(series telem.Series, scalar float64, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, float64](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] == scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func NotEqualScalarF64(series telem.Series, scalar float64, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, float64](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] != scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func GreaterThanScalarF32(series telem.Series, scalar float32, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, float32](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] > scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func GreaterThanOrEqualScalarF32(series telem.Series, scalar float32, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, float32](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] >= scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func LessThanScalarF32(series telem.Series, scalar float32, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, float32](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] < scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func LessThanOrEqualScalarF32(series telem.Series, scalar float32, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, float32](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] <= scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func EqualScalarF32(series telem.Series, scalar float32, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, float32](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] == scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func NotEqualScalarF32(series telem.Series, scalar float32, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, float32](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] != scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func GreaterThanScalarI64(series telem.Series, scalar int64, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int64](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] > scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func GreaterThanOrEqualScalarI64(series telem.Series, scalar int64, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int64](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] >= scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func LessThanScalarI64(series telem.Series, scalar int64, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int64](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] < scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func LessThanOrEqualScalarI64(series telem.Series, scalar int64, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int64](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] <= scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func EqualScalarI64(series telem.Series, scalar int64, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int64](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] == scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func NotEqualScalarI64(series telem.Series, scalar int64, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int64](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] != scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func GreaterThanScalarI32(series telem.Series, scalar int32, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int32](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] > scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func GreaterThanOrEqualScalarI32(series telem.Series, scalar int32, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int32](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] >= scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func LessThanScalarI32(series telem.Series, scalar int32, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int32](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] < scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func LessThanOrEqualScalarI32(series telem.Series, scalar int32, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int32](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] <= scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func EqualScalarI32(series telem.Series, scalar int32, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int32](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] == scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func NotEqualScalarI32(series telem.Series, scalar int32, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int32](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] != scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func GreaterThanScalarI16(series telem.Series, scalar int16, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int16](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] > scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func GreaterThanOrEqualScalarI16(series telem.Series, scalar int16, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int16](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] >= scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func LessThanScalarI16(series telem.Series, scalar int16, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int16](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] < scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func LessThanOrEqualScalarI16(series telem.Series, scalar int16, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int16](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] <= scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func EqualScalarI16(series telem.Series, scalar int16, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int16](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] == scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func NotEqualScalarI16(series telem.Series, scalar int16, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int16](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] != scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func GreaterThanScalarI8(series telem.Series, scalar int8, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int8](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] > scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func GreaterThanOrEqualScalarI8(series telem.Series, scalar int8, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int8](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] >= scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func LessThanScalarI8(series telem.Series, scalar int8, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int8](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] < scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func LessThanOrEqualScalarI8(series telem.Series, scalar int8, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int8](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] <= scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func EqualScalarI8(series telem.Series, scalar int8, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int8](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] == scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func NotEqualScalarI8(series telem.Series, scalar int8, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, int8](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] != scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func GreaterThanScalarU64(series telem.Series, scalar uint64, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint64](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] > scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func GreaterThanOrEqualScalarU64(series telem.Series, scalar uint64, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint64](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] >= scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func LessThanScalarU64(series telem.Series, scalar uint64, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint64](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] < scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func LessThanOrEqualScalarU64(series telem.Series, scalar uint64, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint64](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] <= scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func EqualScalarU64(series telem.Series, scalar uint64, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint64](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] == scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func NotEqualScalarU64(series telem.Series, scalar uint64, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint64](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] != scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func GreaterThanScalarU32(series telem.Series, scalar uint32, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint32](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] > scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func GreaterThanOrEqualScalarU32(series telem.Series, scalar uint32, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint32](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] >= scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func LessThanScalarU32(series telem.Series, scalar uint32, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint32](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] < scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func LessThanOrEqualScalarU32(series telem.Series, scalar uint32, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint32](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] <= scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func EqualScalarU32(series telem.Series, scalar uint32, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint32](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] == scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func NotEqualScalarU32(series telem.Series, scalar uint32, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint32](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] != scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func GreaterThanScalarU16(series telem.Series, scalar uint16, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint16](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] > scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func GreaterThanOrEqualScalarU16(series telem.Series, scalar uint16, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint16](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] >= scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func LessThanScalarU16(series telem.Series, scalar uint16, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint16](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] < scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func LessThanOrEqualScalarU16(series telem.Series, scalar uint16, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint16](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] <= scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func EqualScalarU16(series telem.Series, scalar uint16, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint16](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] == scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func NotEqualScalarU16(series telem.Series, scalar uint16, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint16](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] != scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func GreaterThanScalarU8(series telem.Series, scalar uint8, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint8](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] > scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func GreaterThanOrEqualScalarU8(series telem.Series, scalar uint8, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint8](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] >= scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func LessThanScalarU8(series telem.Series, scalar uint8, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint8](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] < scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func LessThanOrEqualScalarU8(series telem.Series, scalar uint8, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint8](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] <= scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func EqualScalarU8(series telem.Series, scalar uint8, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint8](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] == scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}

func NotEqualScalarU8(series telem.Series, scalar uint8, output *telem.Series) {
	length := series.Len()
	output.Resize(length)

	inData := xunsafe.CastSlice[uint8, uint8](series.Data)
	outData := output.Data

	for i := int64(0); i < length; i++ {
		if inData[i] != scalar {
			outData[i] = 1
		} else {
			outData[i] = 0
		}
	}
}
