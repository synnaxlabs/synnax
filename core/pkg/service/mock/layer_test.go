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
	"github.com/synnaxlabs/synnax/pkg/service/channel/license"
	"github.com/synnaxlabs/synnax/pkg/service/mock"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Layer", func() {
	Describe("Keys", func() {
		It("should sign a license its anchors verify", func() {
			keys := mock.NewKeys()
			g := mock.NewLicense()
			Expect(license.Verify(keys.Anchors, keys.Sign(g))).To(Equal(g))
		})
		It("should not verify a license signed by other keys", func() {
			keys := mock.NewKeys()
			Expect(license.Verify(
				mock.NewKeys().Anchors, keys.Sign(mock.NewLicense()),
			)).Error().To(MatchError(license.ErrInvalid))
		})
	})

	Describe("OpenLayer", func() {
		It("should open a covered layer", func(ctx SpecContext) {
			node := distmock.NewNode(ctx)
			layer := MustOpen(mock.OpenLayer(ctx, node))
			Expect(layer.License.Retrieve().State).To(Equal(license.StateOK))
		})
	})
})
