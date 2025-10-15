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
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/x/set"
)

// DefaultOutputParam is the parameter name for single-output functions and stages.
const DefaultOutputParam = "output"

// DefaultInputParam is the parameter name for single-input functions and stages.
const DefaultInputParam = "input"

// LHSInputParam is the left-hand side parameter name for binary operators.
const LHSInputParam = "a"

// RHSInputParam is the right-hand side parameter name for binary operators.
const RHSInputParam = "b"

// Channels tracks which Synnax channels a node reads from and writes to.
type Channels struct {
	Read  set.Set[uint32] `json:"read"`
	Write set.Set[uint32] `json:"write"`
}

// NewChannels creates an empty Channels with initialized maps.
func NewChannels() Channels {
	return Channels{
		Read:  make(set.Set[uint32]),
		Write: make(set.Set[uint32]),
	}
}

// OverrideChannels creates a Channels from other, ensuring non-nil maps.
func OverrideChannels(other Channels) Channels {
	return Channels{
		Read:  lo.Ternary(other.Read != nil, other.Read, make(set.Set[uint32])),
		Write: lo.Ternary(other.Write != nil, other.Write, make(set.Set[uint32])),
	}
}

// IR is the intermediate representation of an Arc program. It contains function
// definitions, instantiated nodes, dataflow edges, execution stratification, and
// the symbol table from analysis.
type IR struct {
	Functions Functions     `json:"functions"`
	Nodes     Nodes         `json:"nodes"`
	Edges     Edges         `json:"edges"`
	Strata    Strata        `json:"strata"`
	Symbols   *symbol.Scope `json:"-"`
}
