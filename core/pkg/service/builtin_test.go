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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channelv1 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// expectRehomed expects the channel with key held to have the name of the built-in
// builtIn with the suffix _2 and a warning status that says so, and builtIn to have
// another key.
func expectRehomed(
	ctx context.Context,
	n testutil.Node,
	l *service.Layer,
	held channel.Key,
	builtIn channel.Channel,
) {
	GinkgoHelper()
	Expect(builtIn.Key()).ToNot(Equal(held))
	Expect(l.Channel.IsBuiltIn(builtIn.Key())).To(BeTrue())
	var renamed channel.Channel
	Expect(l.Channel.NewRetrieve().
		Where(channel.MatchKeys(held)).
		Entry(&renamed).
		Exec(ctx, n.DB)).To(Succeed())
	Expect(renamed.Name).To(Equal(builtIn.Name + "_2"))
	var warnings []status.Status[any]
	Expect(l.Status.NewRetrieve[any]().
		Where(status.MatchNames[any](renamed.Name)).
		Entries(&warnings).
		Exec(ctx, n.DB)).To(Succeed())
	Expect(warnings).To(HaveLen(1))
	Expect(warnings[0].Variant).To(Equal(status.VariantWarning))
	Expect(warnings[0].Message).To(Equal("Renamed channel"))
}

var _ = Describe("Built-in channel named like a calculation's output", func() {
	It("Should rename the output and create the built-in", func(ctx SpecContext) {
		n := testutil.NewNode(ctx)
		l := MustOpen(svctestutil.OpenLayer(ctx, n))
		c := calculation.Calculation{Expression: "return 1.0"}
		var outputs []channel.Channel
		Expect(n.DB.WithTx(ctx, func(tx gorp.Tx) (err error) {
			outputs, err = l.Calculation.NewWriter(tx).
				CreateOutput(ctx, &c, "computed", nil)
			return err
		})).To(Succeed())
		output := outputs[0]
		builtIn := channel.Channel{
			Name:     output.Name,
			DataType: output.DataType,
			Virtual:  true,
		}
		Expect(n.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return l.Channel.NewWriter(tx).Declare(ctx, &builtIn)
		})).To(Succeed())
		expectRehomed(ctx, n, l, output.Key(), builtIn)
		Expect(calculationOf(ctx, n, l.Calculation, output.Key()).Key).To(Equal(c.Key))
	})

	It("Should rename a v0.62 output that holds a built-in's name at upgrade", func(
		ctx SpecContext,
	) {
		var held channelv1.Channel
		n := openUpgraded(ctx, func(n testutil.Node) {
			keys := MustSucceed(n.Channel.Allocate(ctx, []distchannel.Channel{
				{Name: "sy_task_cmd", Leaseholder: node.KeyFree, Virtual: true},
			}))
			// The output holds JSON, as each built-in does, so a Declare that runs
			// before the calculation chain retypes it would adopt it.
			held = channelv1.Channel{
				Name:        keys[0].Name,
				Leaseholder: node.KeyFree,
				LocalKey:    keys[0].LocalKey,
				DataType:    telem.JSONT,
				Virtual:     true,
				Expression:  "return 1",
			}
			Expect(gorp.WrapWriter[channelv1.Key, channelv1.Channel](n.DB).
				Set(ctx, held)).To(Succeed())
		})
		l := MustOpen(svctestutil.OpenLayer(ctx, n))
		var builtIn channel.Channel
		Expect(l.Channel.NewRetrieve().
			Where(channel.MatchNames("sy_task_cmd")).
			Entry(&builtIn).
			Exec(ctx, n.DB)).To(Succeed())
		expectRehomed(ctx, n, l, held.Key(), builtIn)
		Expect(calculationOf(ctx, n, l.Calculation, held.Key()).Expression).
			To(Equal("return 1"))
	})
})
