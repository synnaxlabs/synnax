// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	disttestutil "github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/license"
	"github.com/synnaxlabs/synnax/pkg/service/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Layer", func() {
	Describe("Signer", func() {
		It("should sign a license its anchors verify", func() {
			signer := testutil.NewSigner()
			lic := testutil.NewLicense()
			Expect(license.Verify(signer.Anchors, signer.Sign(lic))).To(Equal(lic))
		})
		It("should not verify a license from another signer", func() {
			signer := testutil.NewSigner()
			Expect(license.Verify(
				testutil.NewSigner().Anchors, signer.Sign(testutil.NewLicense()),
			)).Error().To(MatchError(license.ErrInvalid))
		})
	})

	Describe("OpenLayer", func() {
		It("should open a covered layer", func(ctx SpecContext) {
			node := disttestutil.NewNode(ctx)
			layer := MustOpen(testutil.OpenLayer(ctx, node))
			Expect(layer.License.Retrieve().State).To(Equal(license.StateOk))
		})
	})
})
