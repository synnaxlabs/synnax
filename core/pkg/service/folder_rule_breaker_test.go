// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package service_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	channelv1 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Folder rule breaker", func() {
	DescribeTable("Should move a channel folder whose name the upgrade wrote",
		func(ctx SpecContext, stored, upgraded string) {
			n := testutil.NewNode(ctx)
			f, rigs := folder(stored), folder("Rigs")
			storeV062(ctx, n, nil, func([]channelv1.Channel) {}, f, rigs)
			l := MustOpen(svctestutil.OpenLayer(ctx, n))
			Expect(folderName(ctx, n, f.Key)).To(Equal(upgraded))
			parent := group.ResourceID(rigs.Key)
			Expect(n.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return l.Group.NewWriter(tx).SetParent(ctx, &parent, f.Key)
			})).To(Succeed())
		},
		Entry("a kept name", "Stand\u200BA", "Stand\u200BA"),
		Entry("a renamed name", "Stand.\u200BA", "Stand_\u200BA"),
	)
})
