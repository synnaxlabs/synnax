// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package node_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/node"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Resources", func() {
	It("Should name each node by its key", func(ctx SpecContext) {
		ids := []resource.ID{node.ResourceID(node.KeyFree), node.ResourceID(1)}
		Expect(
			testSvc.Resources().Retrieve(ctx, nil, ids),
		).To(Equal([]resource.Resource{
			{ID: ids[0], Name: "Node 4095"},
			{ID: ids[1], Name: "Node 1"},
		}))
	})

	It("Should return a validation error for a malformed key", func(ctx SpecContext) {
		Expect(testSvc.Resources().Retrieve(
			ctx,
			nil,
			[]resource.ID{{Type: resource.TypeNode, Key: "pump"}},
		)).Error().To(And(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(`invalid node key "pump"`)),
		))
	})
})
