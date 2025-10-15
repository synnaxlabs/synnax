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

const (
	DefaultOutputParam = "output"
	DefaultInputParam  = "input"
	LHSInputParam      = "a"
	RHSInputParam      = "b"
)

type Channels struct {
	Read  set.Set[uint32] `json:"read"`
	Write set.Set[uint32] `json:"write"`
}

func NewChannels() Channels {
	return Channels{
		Read:  make(set.Set[uint32]),
		Write: make(set.Set[uint32]),
	}
}

func OverrideChannels(other Channels) Channels {
	return Channels{
		Read:  lo.Ternary(other.Read != nil, other.Read, make(set.Set[uint32])),
		Write: lo.Ternary(other.Write != nil, other.Write, make(set.Set[uint32])),
	}
}

type IR struct {
	Functions Functions     `json:"functions"`
	Nodes     Nodes         `json:"nodes"`
	Edges     Edges         `json:"edges"`
	Strata    Strata        `json:"strata"`
	Symbols   *symbol.Scope `json:"-"`
}
