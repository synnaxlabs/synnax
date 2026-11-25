// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package impact_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/impact"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/defaults"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Hidden count checks", Ordered, func() {
	var (
		layer *service.Layer
		db    *gorp.DB
		proj  project.Project
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		node := testutil.NewNode(ctx)
		db = node.DB
		layer = MustOpen(svctestutil.OpenLayer(ctx, node))
		proj = project.Project{Name: "Hidden count checks"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Project.NewWriter(tx).Create(ctx, &proj)
		})).To(Succeed())
	})
	createChannel := func(ctx SpecContext) channel.Key {
		GinkgoHelper()
		ch := channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.Float32T,
			Virtual:  true,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Channel.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		return ch.Key()
	}
	// createStatus creates a status that the channel with key k owns, so the delete of
	// the channel cascades to it.
	createStatus := func(ctx SpecContext, k channel.Key) status.Key {
		GinkgoHelper()
		owner := channel.ResourceID(k)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Status.NewWriter(tx).Set(ctx, &status.Status[any]{
				Owner:   &owner,
				Name:    "reading",
				Variant: status.VariantSuccess,
				Message: "ok",
				Time:    telem.Now(),
			})
		})).To(Succeed())
		var st status.Status[any]
		Expect(layer.Status.NewRetrieve[any]().
			Where(status.MatchOwners[any](owner)).
			Entry(&st).
			Exec(ctx, db)).To(Succeed())
		return st.Key
	}
	deleteOf := func(k channel.Key) []resource.ID {
		return []resource.ID{channel.ResourceID(k)}
	}
	noStatus := func(r resource.ID) bool { return r.Type != resource.TypeStatus }

	It("Should count an entry the caller cannot see that the delete cascades to", func(
		ctx SpecContext,
	) {
		k := createChannel(ctx)
		createStatus(ctx, k)
		p := MustSucceed(layer.Impact.Plan(
			ctx,
			db,
			deleteOf(k),
			impact.PlanOptions{Visible: noStatus},
		))
		Expect(p.Entries).To(BeEmpty())
		Expect(p.Hidden).To(Equal(impact.Hidden{Cascade: 1}))
	})

	It(
		"Should refuse a token from before a hidden entry that the delete cascades to",
		func(ctx SpecContext) {
			k := createChannel(ctx)
			stale := MustSucceed(layer.Impact.Plan(
				ctx,
				db,
				deleteOf(k),
				impact.PlanOptions{Visible: noStatus},
			))
			Expect(stale.Hidden).To(BeZero())
			key := createStatus(ctx, k)
			err := db.WithTx(ctx, func(tx gorp.Tx) error {
				_, err := layer.Impact.Apply(ctx, tx, deleteOf(k), impact.ApplyOptions{
					Token:   stale.Token,
					Visible: noStatus,
				})
				return err
			})
			changed, ok := errors.AsType[impact.PlanChangedError](err)
			Expect(ok).To(BeTrue())
			Expect(changed.Plan.Hidden).To(Equal(impact.Hidden{Cascade: 1}))
			var kept status.Status[any]
			Expect(layer.Status.NewRetrieve[any]().
				Where(status.MatchKeys[any](key)).
				Entry(&kept).
				Exec(ctx, db)).To(Succeed())
		},
	)

	It(
		"Should refuse a token from before a hidden entry that the delete detaches",
		func(ctx SpecContext) {
			k := createChannel(ctx)
			noPlot := func(r resource.ID) bool {
				return r.Type != resource.TypeLineplot
			}
			stale := MustSucceed(layer.Impact.Plan(
				ctx,
				db,
				deleteOf(k),
				impact.PlanOptions{Visible: noPlot},
			))
			Expect(stale.Hidden).To(BeZero())
			plot := defaults.Fill(lineplot.LinePlot{Name: "plot", Project: &proj.Key})
			plot.Channels.Y1 = []channel.Key{k}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.LinePlot.NewWriter(tx).Create(ctx, &plot)
			})).To(Succeed())
			err := db.WithTx(ctx, func(tx gorp.Tx) error {
				_, err := layer.Impact.Apply(ctx, tx, deleteOf(k), impact.ApplyOptions{
					Token:   stale.Token,
					Visible: noPlot,
				})
				return err
			})
			changed, ok := errors.AsType[impact.PlanChangedError](err)
			Expect(ok).To(BeTrue())
			Expect(changed.Plan.Hidden).To(Equal(impact.Hidden{Detach: 1}))
			Expect(MustSucceed(layer.Channel.NewRetrieve().
				Where(channel.MatchKeys(k)).
				Exists(ctx, db))).To(BeTrue())
		},
	)
})
