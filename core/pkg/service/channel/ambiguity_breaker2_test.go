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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// A quoted path escapes each backslash and each single quote in it.
var _ = Describe("Ambiguity breaker 2", func() {
	It("Should escape a quote inside each quoted path", func(ctx SpecContext) {
		fresh := testutil.NewNode(ctx)
		db := fresh.DB
		cfg, groupSvc := serviceConfig(ctx, fresh, fresh.DB)
		a := group.Group{Name: "a", Scope: resource.TypeChannel}
		b := group.Group{Name: "b", Scope: resource.TypeChannel}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			for _, g := range []*group.Group{&a, &b} {
				if err := groupSvc.NewWriter(tx).Create(ctx, g); err != nil {
					return err
				}
			}
			return nil
		})).To(Succeed())
		stored := func(key channel.LocalKey, g *group.Key) channel.Channel {
			return channel.Channel{
				Name:        "it's",
				Leaseholder: node.KeyFree,
				LocalKey:    key,
				DataType:    telem.Float32T,
				Virtual:     true,
				Group:       g,
			}
		}
		inA, inB := stored(7, &a.Key), stored(8, &b.Key)
		createStored(ctx, db, inA, inB)
		svc := MustOpen(channel.OpenService(ctx, cfg))
		all := func(context.Context) (func(resource.ID) bool, error) {
			return func(resource.ID) bool { return true }, nil
		}
		Expect(svc.DescribeAmbiguity(ctx, db, channel.AmbiguousNameError{
			Name:    "it's",
			Holders: channel.Keys{inA.Key(), inB.Key()},
		}, all)).To(MatchError(ContainSubstring(`'a.it\'s', 'b.it\'s'`)))
	})
})
