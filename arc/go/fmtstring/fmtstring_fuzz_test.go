// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package fmtstring_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/synnaxlabs/arc/fmtstring"
	"github.com/synnaxlabs/arc/types"
)

const (
	catInt   = "int"
	catFloat = "float"
	catStr   = "str"
)

type fuzzType struct {
	name     string
	t        types.Type
	dummy    any
	category string
}

var fuzzTypes = []fuzzType{
	{"i8", types.I8(), int64(0), catInt},
	{"i16", types.I16(), int64(0), catInt},
	{"i32", types.I32(), int64(0), catInt},
	{"i64", types.I64(), int64(0), catInt},
	{"u8", types.U8(), uint64(0), catInt},
	{"u16", types.U16(), uint64(0), catInt},
	{"u32", types.U32(), uint64(0), catInt},
	{"u64", types.U64(), uint64(0), catInt},
	{"f32", types.F32(), float64(0), catFloat},
	{"f64", types.F64(), float64(0), catFloat},
	{"string", types.String(), "", catStr},
}

// FuzzValidateSpec asserts the following invariants for any (spec, type) pair:
//
//  1. ValidateSpec never panics.
//  2. If a non-empty spec is accepted, fmt.Sprintf("%"+spec, dummy) must not
//     contain "%!" (i.e. Go's fmt agrees it is a real spec for the type).
//  3. Accepted specs never contain a globally-blacklisted verb (v, T, U).
//  4. Accepted specs for string types never contain x or X.
//  5. Accepted specs for integer types never contain q.
//
// Run as `go test -fuzz=FuzzValidateSpec` for randomized exploration; plain
// `go test` exercises the seed corpus, which already covers the documented
// flag/width/precision/verb combinations and known malformed shapes.
func FuzzValidateSpec(f *testing.F) {
	seeds := []string{
		"", "d", "x", "X", "b", "o", "c", "f", "e", "E", "g", "G", "s", "q",
		"v", "T", "U", "z",
		"+d", "-d", "#x", "#b", "#o", " d", "0d",
		"5d", "05d", "+05d", "-5d", "-5s", "5s", "20s", "-20q",
		".2f", "5.2f", ".0f", "+f", "+08.2f", "#06x", "6.2f", ".10g",
		"5", ".2", ".", "5+d", "f.2", "d5", "sx", ".2-5f",
		"%", "{", "}", "abc", "+++d", "..2f",
	}
	for _, spec := range seeds {
		for i := range fuzzTypes {
			f.Add(spec, uint8(i))
		}
	}
	f.Fuzz(func(t *testing.T, spec string, kind uint8) {
		ft := fuzzTypes[int(kind)%len(fuzzTypes)]
		err := fmtstring.ValidateSpec(spec, ft.t)
		if err != nil {
			return
		}
		if spec == "" {
			return
		}
		out := fmt.Sprintf("%"+spec, ft.dummy)
		if strings.Contains(out, "%!") {
			t.Fatalf("ValidateSpec accepted %q for %s but Go fmt rejects: %q",
				spec, ft.name, out)
		}
		if strings.ContainsAny(spec, "vTU") {
			t.Fatalf("ValidateSpec accepted blacklisted verb in %q for %s",
				spec, ft.name)
		}
		if ft.category == catStr && strings.ContainsAny(spec, "xX") {
			t.Fatalf("ValidateSpec accepted string-blocked verb in %q", spec)
		}
		if ft.category == catInt && strings.ContainsAny(spec, "q") {
			t.Fatalf("ValidateSpec accepted int-blocked verb q in %q for %s",
				spec, ft.name)
		}
	})
}
