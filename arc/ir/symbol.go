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
	"github.com/antlr4-go/antlr/v4"
)

type Kind int

//go:generate stringer -type=Kind
const (
	KindVariable Kind = iota
	KindStatefulVariable
	KindParam
	KindFunction
	KindStage
	KindChannel
	KindConfigParam
	KindBlock
)

type Symbol struct {
	Name       string
	Kind       Kind
	Type       Type
	ParserRule antlr.ParserRuleContext
	ID         int
}
