// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mapbool

type MyKey string

type TypeDecl = map[string]bool // want `map\[string\]bool can be replaced with set\.Set\[string\]`

var Global map[int]bool // want `map\[int\]bool can be replaced with set\.Set\[int\]`

type MyStruct struct {
	Field map[MyKey]bool // want `map\[MyKey\]bool can be replaced with set\.Set\[MyKey\]`
}

func TakesSet(s map[string]bool) {} // want `map\[string\]bool can be replaced with set\.Set\[string\]`

func ReturnsSet() map[string]bool { // want `map\[string\]bool can be replaced with set\.Set\[string\]`
	return nil
}

func LocalVar() {
	m := make(map[int]bool) // want `map\[int\]bool can be replaced with set\.Set\[int\]`
	_ = m
	var n map[string]bool // want `map\[string\]bool can be replaced with set\.Set\[string\]`
	_ = n
}
