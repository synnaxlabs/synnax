// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package graph

type Node struct {
	Key    string
	Type   string
	Config map[string]any
}

type Handle struct {
	Param string
	Node  string
}

type Edge struct {
	Source Handle
	Target Handle
}
