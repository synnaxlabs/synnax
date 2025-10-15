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
	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/types"
)

type Body struct {
	Raw string
	AST antlr.ParserRuleContext
}

type Function struct {
	Key     string       `json:"key"`
	Body    Body         `json:"body"`
	Config  types.Params `json:"config"`
	Inputs  types.Params `json:"inputs"`
	Outputs types.Params `json:"outputs"`
}

func (f Function) Type() types.Type {
	return types.Function(types.FunctionProperties{
		Config:  &f.Config,
		Inputs:  &f.Inputs,
		Outputs: &f.Outputs,
	})
}

type Functions []Function

func (f Functions) Get(key string) Function {
	return lo.Must(f.Find(key))
}

func (f Functions) Find(key string) (Function, bool) {
	return lo.Find(f, func(fn Function) bool { return fn.Key == key })
}
