// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package node

import (
	"github.com/samber/lo"
)

type Group map[Key]Node

func (n Group) WhereState(state State) Group {
	return n.Where(func(_ Key, n Node) bool { return n.State == state })
}

func (n Group) WhereNot(keys ...Key) Group {
	return n.Where(func(id Key, _ Node) bool { return lo.Count(keys, id) == 0 })
}

func (n Group) WhereActive() Group {
	return n.Where(func(_ Key, n Node) bool { return n.State != StateLeft })
}

func (n Group) Where(cond func(Key, Node) bool) Group { return lo.PickBy(n, cond) }

func (n Group) Digests() Digests {
	dig := make(Digests, len(n))
	for id, node := range n {
		dig[id] = node.digest()
	}
	return dig
}

func (n Group) Copy() Group { return lo.PickBy(n, func(_ Key, _ Node) bool { return true }) }
