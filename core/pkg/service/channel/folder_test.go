// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("CheckFolderName", func() {
	DescribeTable("Should return a name that a path can hold in NFC",
		func(name, expected string) {
			Expect(channel.CheckFolderName(name)).To(Equal(expected))
		},
		Entry("a name in NFC", "Stand A", "Stand A"),
		Entry("a decomposed name", "e\u0301", "\u00E9"),
		Entry("an empty name, which the group validation reports", "", ""),
	)

	DescribeTable("Should refuse a name that a path cannot hold",
		func(name string) {
			Expect(channel.CheckFolderName(name)).Error().
				To(MatchError(validate.ErrValidation))
		},
		Entry("a '.'", "stand.a"),
		Entry("an invisible character alone", "\u200B"),
		Entry("bytes that are not UTF-8", "a\xffb"),
	)
})

var _ = Describe("Channel folder names", func() {
	It("Should report an empty channel folder name as required", func(
		ctx SpecContext,
	) {
		node := testutil.NewNode(ctx)
		_, groupSvc := serviceConfig(ctx, node, node.DB)
		g := group.Group{Scope: resource.TypeChannel}
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return groupSvc.NewWriter(tx).Create(ctx, &g)
		})).To(MatchError(ContainSubstring("name: required")))
	})
})
