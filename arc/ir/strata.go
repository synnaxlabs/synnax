// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ir

// Strata represents the execution stratification of a dataflow graph.
// Each node is assigned a stratum (execution level) based on its dependencies.
// Stratification enables single-pass, glitch-free reactive execution.
type Strata [][]string

func (s Strata) Get(key string) int {
	for i, nodes := range s {
		for _, node := range nodes {
			if node == key {
				return i
			}
		}
	}
	return -1
}

func (s Strata) Has(key string) bool {
	return s.Get(key) >= 0
}

func (s Strata) NodeCount() int {
	count := 0
	for _, nodes := range s {
		count += len(nodes)
	}
	return count
}
