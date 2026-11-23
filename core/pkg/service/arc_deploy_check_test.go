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
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Arc deploy check", func() {
	It("Should record the deployed lists when the Core's Driver configures an Arc "+
		"task", func(ctx SpecContext) {
		n := testutil.NewNode(ctx)
		l := MustOpen(svctestutil.OpenLayer(ctx, n))
		// The Core's own Driver runs Arc tasks on its rack, which is not the rack of
		// the
		// embedded Driver.
		core := MustSucceed(l.Rack.RetrieveOrCreateEmbedded(
			ctx,
			fmt.Sprintf("Node %d", n.Identity.HostKey),
		))
		in := channel.Channel{Name: "dc_in", Virtual: true, DataType: telem.Float64T}
		out := channel.Channel{Name: "dc_out", Virtual: true, DataType: telem.Float64T}
		Expect(n.DB.WithTx(ctx, func(tx gorp.Tx) error {
			w := l.Channel.NewWriter(tx)
			if err := w.Create(ctx, &in); err != nil {
				return err
			}
			return w.Create(ctx, &out)
		})).To(Succeed())
		a := arc.Arc{Variant: arc.AutomationArc{Base: arc.Base{
			Name: "dc_automation",
			Source: arc.Source{Variant: arc.TextSource{
				Text: text.Text{Raw: "dc_in * 2.0 -> dc_out"},
			}},
		}}}
		Expect(n.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return l.Arc.NewWriter(tx).Create(ctx, &a)
		})).To(Succeed())
		key := a.Base().Key
		Expect(n.DB.WithTx(ctx, func(tx gorp.Tx) error {
			_, err := l.Arc.NewWriter(tx).UpdateTask(
				ctx,
				key,
				arc.TaskUpdate{Rack: new(core.Key)},
			)
			return err
		})).To(Succeed())
		Eventually(func(g Gomega) {
			rec, ok := MustSucceed(l.Arc.RetrieveDrift(ctx, n.DB, key))[key]
			g.Expect(ok).To(BeTrue())
			g.Expect([2][]channel.Key{rec.Reads, rec.Writes}).
				To(Equal([2][]channel.Key{{in.Key()}, {out.Key()}}))
		}).WithTimeout(10 * time.Second).Should(Succeed())
	})
})
