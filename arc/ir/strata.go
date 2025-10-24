// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ir

// Strata represents the execution stratification of a dataflow graph. Each stratum
// is a slice of node keys that can execute in parallel. Nodes in stratum N depend
// only on nodes in strata 0 to N-1. Stratification enables single-pass, glitch-free
// reactive execution.
type Strata [][]string

// Get returns the stratum level of the node with the given key, or -1 if not found.
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

// Has reports whether the node with the given key exists in any stratum.
func (s Strata) Has(key string) bool {
	return s.Get(key) >= 0
}

// NodeCount returns the total number of nodes across all strata.
func (s Strata) NodeCount() int {
	count := 0
	for _, nodes := range s {
		count += len(nodes)
	}
	return count
}
