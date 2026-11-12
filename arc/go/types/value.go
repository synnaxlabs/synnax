// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types

import "math"

// JSON has no form for NaN or infinity, so an IR value holds them as this text. The
// param's float type tells a reader to decode it.
const (
	nanValue    = "NaN"
	infValue    = "Infinity"
	negInfValue = "-Infinity"
)

// EncodeFloat returns the IR value for f.
func EncodeFloat(f float64) any {
	switch {
	case math.IsNaN(f):
		return nanValue
	case math.IsInf(f, 1):
		return infValue
	case math.IsInf(f, -1):
		return negInfValue
	default:
		return f
	}
}

// DecodeValue returns the Go value of the IR value v for a param of type t. It returns
// v unchanged unless v is an encoded float.
func DecodeValue(v any, t Type) any {
	if t.Kind == KindVarRef && t.Elem != nil {
		t = *t.Elem
	}
	text, ok := v.(string)
	if !ok || t.Kind == KindString {
		return v
	}
	var f float64
	switch text {
	case nanValue:
		f = math.NaN()
	case infValue:
		f = math.Inf(1)
	case negInfValue:
		f = math.Inf(-1)
	default:
		return v
	}
	if t.Kind == KindF32 {
		return float32(f)
	}
	return f
}
