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
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channelv1 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v1"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Kept name upgrade", func() {
	// A v0.62 Core with name validation off stored this name as sent, not in NFC.
	const decomposed = "café"

	createIfExists := func(
		ctx SpecContext,
		n testutil.Node,
		svc *channel.Service,
		name string,
	) (channel.Channel, error) {
		ch := channel.Channel{Name: name, DataType: telem.Float32T, Virtual: true}
		err := n.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &ch, channel.RetrieveIfNameExists())
		})
		return ch, err
	}

	It("Should delete a kept v0.62 name by the exact name that it holds",
		func(ctx SpecContext) {
			n := testutil.NewNode(ctx)
			stored := storeV062(
				ctx,
				n,
				[]string{decomposed},
				func([]channelv1.Channel) {},
			)
			l := MustOpen(svctestutil.OpenLayer(ctx, n))
			// The upgrade writes the name in NFC.
			Expect(namesOf(ctx, n, stored)[stored[0].Key()]).To(Equal("caf\u00E9"))
			Expect(n.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return l.Channel.NewWriter(tx).
					DeleteManyByNames(ctx, []string{decomposed}, false)
			})).To(Succeed())
			Expect(l.Channel.NewRetrieve().
				Where(channel.MatchKeys(stored[0].Key())).
				Exists(ctx, n.DB)).To(BeFalse())
		})

	It("Should retrieve a kept v0.62 name when a create sends its NFC form",
		func(ctx SpecContext) {
			n := testutil.NewNode(ctx)
			stored := storeV062(
				ctx,
				n,
				[]string{decomposed},
				func([]channelv1.Channel) {},
			)
			l := MustOpen(svctestutil.OpenLayer(ctx, n))
			ch := MustSucceed(createIfExists(ctx, n, l.Channel, "café"))
			Expect(ch.Key()).To(Equal(stored[0].Key()))
		})

	It(
		"Should refuse a name that channels in two folders kept from a v0.62 store",
		func(ctx SpecContext) {
			n := testutil.NewNode(ctx)
			rigs := folder("Rigs")
			stored := storeV062(
				ctx,
				n,
				[]string{"pt", "pt"},
				func(cs []channelv1.Channel) {
					cs[0].Group = &rigs.Key
				},
				rigs,
			)
			// A v0.62 create that retrieves by name took the last match in key order,
			// which is also the channel that Arc binds the name to.
			bound := stored[1].Key()
			Expect(bound).To(BeNumerically(">", stored[0].Key()))
			l := MustOpen(svctestutil.OpenLayer(ctx, n))
			// The upgrade renames no name that its channels hold in two folders.
			names := namesOf(ctx, n, stored)
			Expect(names[stored[0].Key()]).To(Equal("pt"))
			Expect(names[bound]).To(Equal("pt"))
			// The flip rewrites a bare name to the path of the channel that v0.62
			// bound,
			// so a lookup of the bare name is refused until then.
			ambiguous := MatchError(ContainSubstring("2 channels are named 'pt'"))
			Expect(l.Channel.NewArcSymbolResolver(n.DB).Resolve(ctx, "pt")).Error().
				To(ambiguous)
			_, err := createIfExists(ctx, n, l.Channel, "pt")
			Expect(err).To(ambiguous)
		},
	)
})
