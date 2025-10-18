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
	"github.com/synnaxlabs/arc/symbol"
)

// DefaultOutputParam is the parameter name for single-output functions and stages.
const (
	DefaultOutputParam = "output"
	// DefaultInputParam is the parameter name for single-input functions and stages.
	DefaultInputParam = "input"
	// LHSInputParam is the left-hand side parameter name for binary operators.
	LHSInputParam = "a"
	// RHSInputParam is the right-hand side parameter name for binary operators.
	RHSInputParam = "b"
)

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
