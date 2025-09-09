// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ir

import "github.com/synnaxlabs/arc/types"

type Stage struct {
	Key               string                `json:"key"`
	Params            map[string]types.Type `json:"params"`
	Config            map[string]types.Type `json:"config"`
	Return            types.Type            `json:"return"`
	StatefulVariables map[string]types.Type `json:"stateful_variables"`
	Body              string                `json:"body"`
}

type Node struct {
	Key    string         `json:"key"`
	Type   string         `json:"type"`
	Config map[string]any `json:"config"`
}

type Handle struct {
	Node  string `json:"node"`
	Param string `json:"param"`
}

type Edge struct {
	Source Handle `json:"source"`
	Target Handle `json:"target"`
}

type IR struct {
	Stages []Stage `json:"stages"`
	Nodes  []Node  `json:"nodes"`
	Edges  []Edge  `json:"edges"`
}
