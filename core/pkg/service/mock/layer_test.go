// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mock_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	distmock "github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/channel/verification"
	"github.com/synnaxlabs/synnax/pkg/service/mock"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Layer", func() {
	Describe("Keys", func() {
		It("should sign a grant its anchors verify", func() {
			keys := mock.NewKeys()
			g := mock.NewLicense()
			Expect(verification.Verify(keys.Anchors, keys.Sign(g))).To(Equal(g))
		})
		It("should not verify a grant signed by other keys", func() {
			keys := mock.NewKeys()
			Expect(verification.Verify(
				mock.NewKeys().Anchors, keys.Sign(mock.NewLicense()),
			)).Error().To(MatchError(verification.ErrInvalid))
		})
	})

	Describe("OpenLayer", func() {
		It("should open a covered layer", func(ctx SpecContext) {
			node := distmock.NewNode(ctx)
			layer := MustOpen(mock.OpenLayer(ctx, node))
			Expect(layer.Verification.Retrieve().State).To(Equal(verification.StateOK))
		})
	})
})
