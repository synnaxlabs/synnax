// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types

// SizedSignedInteger represents signed integers with fixed bit widths.
type SizedSignedInteger interface {
	~int8 | ~int16 | ~int32 | ~int64
}

// SizedUnsignedInteger represents unsigned integers with fixed bit widths.
type SizedUnsignedInteger interface {
	~uint8 | ~uint16 | ~uint32 | ~uint64
}

// Floating represents any floating-point value.
type Floating interface {
	~float32 | ~float64
}

// SignedInteger represents any integer value that can be negative.
type SignedInteger interface {
	~int | SizedSignedInteger
}

// UnsignedInteger represents any integer value that can only be positive.
type UnsignedInteger interface {
	~uint | SizedUnsignedInteger
}

// Integer represents any integer value.
type Integer interface {
	SignedInteger | UnsignedInteger
}

// SizedNumeric represents any numeric value with a fixed bit width.
type SizedNumeric interface {
	SizedSignedInteger | SizedUnsignedInteger | Floating
}

// Sized represents any numeric value with a fixed bit width.
type Sized interface {
	SizedNumeric | ~[4]byte | ~[8]byte | ~[16]byte
}

// Signed represents any numeric value that can be negative.
type Signed interface {
	SignedInteger | Floating
}

// Numeric represents any numeric value.
type Numeric interface {
	Integer | Floating
}
