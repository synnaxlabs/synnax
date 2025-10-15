// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol

import (
	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/types"
)

type Kind int

//go:generate stringer -type=Kind
const (
	KindVariable Kind = iota
	KindStatefulVariable
	KindInput
	KindFunction
	KindChannel
	KindConfig
	KindBlock
	KindOutput
)

type Symbol struct {
	Type types.Type
	Name string
	Kind Kind
	AST  antlr.ParserRuleContext
	ID   int
}
