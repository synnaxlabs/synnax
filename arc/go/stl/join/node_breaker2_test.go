// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package join_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/stl/join"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Breaker 2 node", func() {
	// S2 and the rule that an internal dispatch key fails loud: join.New refuses
	// arrival
	// marks whose count is not the input count, so a symbol that marks an input it does
	// not have must not open.
	It("Should refuse a symbol whose arrival marks an input it does not have", func(
		ctx SpecContext,
	) {
		sym := join.Symbol(
			nodeType,
			[]types.Type{types.F64(), types.F64()},
			[]bool{false, false, true},
			0,
		)
		state := openJoin(ctx, false).state
		h := join.NewHost(nodeType, telem.Now)
		Expect(h.Create(node.Config{
			Node:  ir.Node{Key: "join", Type: nodeType, Inputs: sym.Type.Inputs},
			State: state.Node("join"),
		})).Error().To(MatchError(ContainSubstring("arrival")))
	})
})
