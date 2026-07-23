// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2

import "github.com/samber/lo"

// Find returns the node with the given key and a boolean indicating whether the node
// was found. This is the safe variant of Get.
func (n Nodes) Find(key string) (Node, bool) {
	return lo.Find(n, func(n Node) bool { return n.Key == key })
}

// Get returns the node with the given key. Panics if the node is not found. Use Find
// for safe lookups with error handling.
func (n Nodes) Get(key string) Node { return lo.Must(n.Find(key)) }
