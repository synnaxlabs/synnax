// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ir

import (
	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/types"
)

type Node struct {
	Key          string         `json:"key"`
	Type         string         `json:"type"`
	ConfigValues map[string]any `json:"config_values"`
	Channels     Channels       `json:"channels"`
	Config       types.Params   `json:"config"`
	Inputs       types.Params   `json:"inputs"`
	Outputs      types.Params   `json:"outputs"`
}

type Nodes []Node

func (n Nodes) Get(key string) Node {
	return lo.Must(lo.Find(n, func(n Node) bool { return n.Key == key }))
}
