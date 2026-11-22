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
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// D-284: the Core deletes its retired channels once, by key, from the internal v0.62
// records that hold a retired name.
var _ = Describe("Retired channel check", func() {
	It("Should keep a user channel with a retired name that is made after the upgrade",
		func(ctx SpecContext) {
			n := testutil.NewNode(ctx)
			first := MustSucceed(svctestutil.OpenLayer(ctx, n))
			ch := channel.Channel{
				Name:        "sy_range_alias_set",
				DataType:    telem.StringT,
				Virtual:     true,
				Leaseholder: node.KeyFree,
			}
			Expect(n.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return first.Channel.NewWriter(tx).Create(ctx, &ch)
			})).To(Succeed())
			Expect(first.Close()).To(Succeed())
			MustOpen(svctestutil.OpenLayer(ctx, n))
			Expect(retrieveChannel(ctx, n.DB, ch.Key()).Name).
				To(Equal("sy_range_alias_set"))
		})

	It("Should delete the retired channel of a v0.62 Core and keep a user channel "+
		"with its name", func(ctx SpecContext) {
		n := testutil.NewNode(ctx)
		keys := createNamedV062Store(
			ctx,
			n,
			nil,
			storedChannel{name: "sy_range_alias_set", internal: true},
			storedChannel{name: "sy_range_alias_set"},
		)
		MustOpen(svctestutil.OpenLayer(ctx, n))
		var left []channel.Channel
		Expect(gorp.NewRetrieve[channel.Key, channel.Channel]().
			Where(gorp.Match(func(_ gorp.Context, c *channel.Channel) (bool, error) {
				return c.Key() == keys[0] || c.Key() == keys[1], nil
			})).
			Entries(&left).
			Exec(ctx, n.DB)).To(Succeed())
		Expect(left).To(HaveLen(1))
		Expect(left[0].Key()).To(Equal(keys[1]))
	})
})
