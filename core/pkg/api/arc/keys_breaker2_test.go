// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/gorp"
	apiarc "github.com/synnaxlabs/synnax/pkg/api/arc"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Deployed lists breaker 2", func() {
	It("Should keep the deployed lists of a task when a rack that does not run the "+
		"task takes the program of its automation", func(ctx SpecContext) {
		in := channel.Channel{Name: "bk2_in", DataType: telem.Float64T, Virtual: true}
		out := channel.Channel{Name: "bk2_out", DataType: telem.Float64T, Virtual: true}
		otherRack := rack.Rack{Name: "bk2 other rack"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			w := channelSvc.NewWriter(tx)
			for _, ch := range []*channel.Channel{&in, &out} {
				if err := w.Create(ctx, ch); err != nil {
					return err
				}
			}
			return rackSvc.NewWriter(tx).Create(ctx, &otherRack)
		})).To(Succeed())
		a := arc.Arc{Variant: arc.AutomationArc{Base: arc.Base{
			Name: "bk2_automation",
			Source: arc.Source{Variant: arc.TextSource{
				Text: text.Text{Raw: "bk2_in * 2.0 -> bk2_out"},
			}},
		}}}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return arcSvc.NewWriter(tx).Create(ctx, &a)
		})).To(Succeed())
		key := a.Base().Key
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return arcSvc.NewWriter(tx).Delete(ctx, key)
			})).To(Succeed())
		})
		deploy(ctx, a)
		deployed := func() [2][]channel.Key {
			rec := MustSucceed(arcSvc.RetrieveDrift(ctx, db, key))[key]
			return [2][]channel.Key{rec.Reads, rec.Writes}
		}
		req := apiarc.RetrieveRequest{
			Where:          &arc.Where{Key: &filter.Set[arc.Key]{In: []arc.Key{key}}},
			IncludeProgram: true,
		}
		// The Driver of the task's rack takes the program.
		MustSucceed(retrieve(SubjectCtx(ctx, testRack.ResourceID()), req))
		running := [2][]channel.Key{{in.Key()}, {out.Key()}}
		Expect(deployed()).To(Equal(running))
		// A rename schedules no resync, so the task's config and the program that its
		// Driver runs stay on the first bk2_in.
		again := channel.Channel{
			Name:     "bk2_in",
			DataType: telem.Float64T,
			Virtual:  true,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			w := channelSvc.NewWriter(tx)
			if err := w.Rename(ctx, in.Key(), "bk2_was", false); err != nil {
				return err
			}
			return w.Create(ctx, &again)
		})).To(Succeed())
		// A rack that does not run the task takes the program.
		MustSucceed(retrieve(SubjectCtx(ctx, otherRack.ResourceID()), req))
		Expect(deployed()).To(Equal(running))
	})
})
