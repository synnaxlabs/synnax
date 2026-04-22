// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package clean

type Custom struct {
	Name string
}

var StringMap map[string]int

var StructMap map[string]Custom

var SliceMap map[string][]byte

func TakesMap(m map[int]string) {}

func ReturnsMap() map[string]float64 {
	return nil
}

func LocalVars() {
	m := make(map[string]int)
	_ = m
	var n map[int]Custom
	_ = n
}
