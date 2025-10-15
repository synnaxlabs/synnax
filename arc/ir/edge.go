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
)

type Handle struct {
	Node  string `json:"node"`
	Param string `json:"param"`
}

type Edge struct {
	Source Handle `json:"source"`
	Target Handle `json:"target"`
}

type Edges []Edge

func (e Edges) GetBySource(handle Handle) Edge {
	return e.get(func(e Edge) bool { return e.Source == handle })
}

func (e Edges) GetByTarget(handle Handle) Edge {
	return e.get(func(e Edge) bool { return e.Target == handle })
}

func (e Edges) find(f func(e Edge) bool) (Edge, bool) {
	return lo.Find(e, f)
}

func (e Edges) get(f func(e Edge) bool) Edge {
	return lo.Must(e.find(f))
}

func (e Edges) FindBySource(handle Handle) (Edge, bool) {
	return e.find(func(e Edge) bool { return e.Source == handle })
}

func (e Edges) FindByTarget(handle Handle) (Edge, bool) {
	return e.find(func(e Edge) bool { return e.Target == handle })
}
