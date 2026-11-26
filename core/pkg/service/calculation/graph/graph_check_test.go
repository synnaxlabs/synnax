// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package graph_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	calctestutil "github.com/synnaxlabs/synnax/pkg/service/calculation/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Ambiguous inputs (check)", func() {
	It("Should check a calculation again when one holder of its ambiguous input "+
		"goes", func(ctx SpecContext) {
		base := channel.Channel{Name: "rc_amb_pt", DataType: telem.Int64T, Virtual: true}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, &base)
		})).To(Succeed())
		calc := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
			Name:       "rc_amb_calc",
			DataType:   telem.Int64T,
			Expression: "return rc_amb_pt + 1",
		}))
		twin := channel.Channel{Name: "rc_amb_twin", DataType: telem.Int64T, Virtual: true}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, &twin)
		})).To(Succeed())
		// A v0.62 store can hold one name twice.
		twin.Name = base.Name
		Expect(gorp.WrapWriter[channel.Key, channel.Channel](db).Set(ctx, twin)).
			To(Succeed())
		g := openGraph(ctx)
		DeferCleanup(g.Close)
		expectStatus(ctx, calc.Key())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Delete(ctx, twin.Key(), false)
		})).To(Succeed())
		eventuallyExpectNoStatus(ctx, calc.Key())
	})
})
