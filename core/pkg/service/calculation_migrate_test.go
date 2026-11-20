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

var _ = Describe("Calculation records upgrade", func() {
	It("Should build a record for each calculated channel of a v0.62 store", func(
		ctx SpecContext,
	) {
		n := testutil.NewNode(ctx)
		// v0.62 stores channel v1 records, under keys that its counter issued.
		keys := MustSucceed(n.Channel.Allocate(ctx, []distchannel.Channel{
			{Name: "plain", Leaseholder: node.KeyFree, Virtual: true},
			{Name: "doubled", Leaseholder: node.KeyFree, Virtual: true},
			{Name: "windowed", Leaseholder: node.KeyFree, Virtual: true},
		}))
		virtual := func(i int) channelv1.Channel {
			return channelv1.Channel{
				Name:        keys[i].Name,
				Leaseholder: node.KeyFree,
				LocalKey:    keys[i].LocalKey,
				DataType:    telem.Float32T,
				Virtual:     true,
			}
		}
		plain := virtual(0)
		doubled := virtual(1)
		doubled.Expression = "return 2"
		windowed := virtual(2)
		windowed.Expression = "return 3"
		windowed.Operations = []channelv1.Operation{
			{Type: channelv1.OperationTypeAvg, Duration: telem.Second},
			{Type: channelv1.OperationTypeMax, ResetChannel: plain.Key()},
		}
		Expect(gorp.WrapWriter[channelv1.Key, channelv1.Channel](n.DB).
			Set(ctx, plain, doubled, windowed)).To(Succeed())
		// v0.62 stores status v2 records, and its status chain has run, so opening the
		// status service once applies the chain before the record is written.
		Expect(MustSucceed(status.OpenService(ctx, status.ServiceConfig{DB: n.DB})).
			Close()).To(Succeed())
		owned := status.Status[any]{
			Key:     "calculation-status",
			Name:    doubled.Name,
			Variant: status.VariantError,
			Message: "Failed to compile calculation",
			Owner:   new(channel.ResourceID(doubled.Key())),
		}
		Expect(gorp.WrapWriter[status.Key, status.Status[any]](n.DB).Set(ctx, owned)).
			To(Succeed())

		l := MustOpen(svctestutil.OpenLayer(ctx, n))

		var records []calculation.Calculation
		Expect(l.Calculation.NewRetrieve().Entries(&records).Exec(ctx, n.DB)).
			To(Succeed())
		byOutput := make(map[channel.Key]calculation.Calculation, len(records))
		for _, c := range records {
			byOutput[c.Output] = c
		}
		Expect(byOutput).To(HaveKey(doubled.Key()))
		Expect(byOutput).To(HaveKey(windowed.Key()))
		Expect(byOutput).ToNot(HaveKey(plain.Key()))
		Expect(byOutput[doubled.Key()].Key).
			To(Equal(calculation.MigratedKey(doubled.Key())))
		Expect(byOutput[doubled.Key()].Expression).To(Equal("return 2"))
		Expect(byOutput[doubled.Key()].Operations).To(BeEmpty())
		Expect(byOutput[windowed.Key()].Operations).To(Equal([]calculation.Operation{
			{Variant: calculation.AvgOperation{
				Window: calculation.Window{Duration: telem.Second},
			}},
			{Variant: calculation.MaxOperation{
				Window: calculation.Window{Reset: new(plain.Key())},
			}},
		}))
		Expect(gorp.NewRetrieve[status.Key, status.Status[any]]().
			Where(gorp.MatchKeys[status.Key, status.Status[any]](owned.Key)).
			Exists(ctx, n.DB)).To(BeFalse())
	})
})
