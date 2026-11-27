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
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Folder names", func() {
	create := func(ctx SpecContext, scope resource.Type, name string) error {
		n := testutil.NewNode(ctx)
		l := MustOpen(svctestutil.OpenLayer(ctx, n))
		g := group.Group{Name: name, Scope: scope}
		return n.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return l.Group.NewWriter(tx).Create(ctx, &g)
		})
	}

	It("Should refuse a channel folder whose name holds a '.'", func(ctx SpecContext) {
		Expect(create(ctx, resource.TypeChannel, "Stand.A")).To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("no '.'")),
		))
	})

	DescribeTable("Should refuse a channel folder whose name a path cannot hold",
		func(ctx SpecContext, name string) {
			Expect(create(ctx, resource.TypeChannel, name)).
				To(MatchError(validate.ErrValidation))
		},
		Entry("an empty name", ""),
		Entry("a blank name", "   "),
		Entry("an invisible character", "a\u200Bb"),
	)

	It("Should accept a device folder whose name holds a '.'", func(ctx SpecContext) {
		Expect(create(ctx, resource.TypeDevice, "Stand.A")).To(Succeed())
	})
})
