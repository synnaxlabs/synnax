// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mapstruct

type MyKey string

type TypeDecl = map[string]struct{} // want `map\[string\]struct\{\} can be replaced with set\.Set\[string\]`

var Global map[int]struct{} // want `map\[int\]struct\{\} can be replaced with set\.Set\[int\]`

type MyStruct struct {
	Field map[MyKey]struct{} // want `map\[MyKey\]struct\{\} can be replaced with set\.Set\[MyKey\]`
}

func TakesSet(s map[string]struct{}) {} // want `map\[string\]struct\{\} can be replaced with set\.Set\[string\]`

func ReturnsSet() map[string]struct{} { // want `map\[string\]struct\{\} can be replaced with set\.Set\[string\]`
	return nil
}

func LocalVar() {
	m := make(map[int]struct{}) // want `map\[int\]struct\{\} can be replaced with set\.Set\[int\]`
	_ = m
	var n map[string]struct{} // want `map\[string\]struct\{\} can be replaced with set\.Set\[string\]`
	_ = n
}
