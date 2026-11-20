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
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channelv1 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Calculation records upgrade checks", func() {
	// core/v0.62.0 checks only retrieve access on a reset channel, so its API stores a
	// reset key that no channel ever held.
	It(
		"Should start on a v0.62 store whose calculation resets on an unissued key",
		func(
			ctx SpecContext,
		) {
			n := testutil.NewNode(ctx)
			keys := MustSucceed(n.Channel.Allocate(ctx, []distchannel.Channel{
				{Name: "windowed", Leaseholder: node.KeyFree, Virtual: true},
			}))
			unissued := distchannel.NewKey(node.KeyFree, 1<<19)
			Expect(n.Channel.Issued(unissued)).To(BeFalse())
			windowed := channelv1.Channel{
				Name:        keys[0].Name,
				Leaseholder: node.KeyFree,
				LocalKey:    keys[0].LocalKey,
				DataType:    telem.Float32T,
				Virtual:     true,
				Expression:  "return 3",
				Operations: []channelv1.Operation{
					{Type: channelv1.OperationTypeMax, ResetChannel: unissued},
				},
			}
			Expect(gorp.WrapWriter[channelv1.Key, channelv1.Channel](n.DB).
				Set(ctx, windowed)).To(Succeed())

			l := MustOpen(svctestutil.OpenLayer(ctx, n))

			Expect(l.Calculation.NewRetrieve().
				Where(calculation.MatchOutputs(windowed.Key())).
				Exists(ctx, n.DB)).To(BeTrue())
		},
	)

	It(
		"Should keep the compile error status of an invalid calculation after upgrade",
		func(
			ctx SpecContext,
		) {
			n := testutil.NewNode(ctx)
			keys := MustSucceed(n.Channel.Allocate(ctx, []distchannel.Channel{
				{Name: "broken", Leaseholder: node.KeyFree, Virtual: true},
			}))
			broken := channelv1.Channel{
				Name:        keys[0].Name,
				Leaseholder: node.KeyFree,
				LocalKey:    keys[0].LocalKey,
				DataType:    telem.Float32T,
				Virtual:     true,
				Expression:  "return no_such_channel * 2",
			}
			Expect(gorp.WrapWriter[channelv1.Key, channelv1.Channel](n.DB).
				Set(ctx, broken)).To(Succeed())

			l := MustOpen(svctestutil.OpenLayer(ctx, n))

			var statuses []status.Status[any]
			Expect(gorp.NewRetrieve[status.Key, status.Status[any]]().
				Entries(&statuses).
				Exec(ctx, n.DB)).To(Succeed())
			owner := channel.ResourceID(broken.Key())
			var owned []status.Status[any]
			for _, st := range statuses {
				if st.Owner != nil && *st.Owner == owner {
					owned = append(owned, st)
				}
			}
			Expect(owned).To(HaveLen(1))
			Expect(owned[0].Variant).To(Equal(status.VariantError))
			Expect(l.Calculation.NewRetrieve().
				Where(calculation.MatchOutputs(broken.Key())).
				Exists(ctx, n.DB)).To(BeTrue())
		},
	)
})
