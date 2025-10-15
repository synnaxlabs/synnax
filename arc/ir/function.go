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

type Function struct {
	Key     string       `json:"key"`
	Body    types.Body   `json:"body"`
	Config  types.Params `json:"config"`
	Inputs  types.Params `json:"inputs"`
	Outputs types.Params `json:"outputs"`
}

func (f Function) Type() types.Type {
	return types.Function(f.Inputs, f.Outputs, f.Config)
}

type Functions []Function

func (f Functions) Get(key string) Function {
	return lo.Must(lo.Find(f, func(fn Function) bool { return fn.Key == key }))
}
