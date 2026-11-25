// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calculation_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	disttestutil "github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	acttestutil "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/search"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Resources", func() {
	var (
		node     disttestutil.Node
		channels *channel.Service
		svc      *calculation.Service
	)
	BeforeEach(func(ctx SpecContext) {
		node = disttestutil.NewNode(ctx)
		statuses := MustSucceed(
			status.OpenService(ctx, status.ServiceConfig{DB: node.DB}),
		)
		DeferCleanup(func() { Expect(statuses.Close()).To(Succeed()) })
		channels = MustSucceed(channel.OpenService(ctx, channel.ServiceConfig{
			Channel: node.Channel,
			TS:      node.Storage.TS,
			DB:      node.DB,
			HostKey: node.Identity.HostKey,
			Status:  statuses,
			AnnounceLinked: func(
				ctx context.Context,
				tx gorp.Tx,
				w channel.Writer,
				keys ...channel.Key,
			) error {
				return svc.AnnounceOutputs(ctx, tx, w, keys...)
			},
			AnnounceRenamed: func(
				ctx context.Context,
				tx gorp.Tx,
				keys ...channel.Key,
			) error {
				return svc.AnnounceRenamed(ctx, tx, keys...)
			},
		}))
		DeferCleanup(func() { Expect(channels.Close()).To(Succeed()) })
		svc = MustSucceed(calculation.OpenService(ctx, calculation.ServiceConfig{
			DB:      node.DB,
			Channel: channels,
		}))
		DeferCleanup(func() { Expect(svc.Close()).To(Succeed()) })
	})
	create := func(ctx SpecContext, name string) calculation.Calculation {
		GinkgoHelper()
		out := channel.Channel{Name: name, DataType: telem.Float32T, Virtual: true}
		c := calculation.Calculation{Expression: "return 2"}
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			if err := channels.NewWriter(tx).Create(ctx, &out); err != nil {
				return err
			}
			c.Output = out.Key()
			return svc.NewWriter(tx).Create(ctx, &c)
		})).To(Succeed())
		return c
	}

	It("Should name a calculation by its output channel", func(ctx SpecContext) {
		c := create(ctx, "tank_mean")
		Expect(svc.Resources().Retrieve(ctx, node.DB, []resource.ID{
			calculation.ResourceID(c.Key),
		})).To(ConsistOf(resource.Resource{
			ID:   calculation.ResourceID(c.Key),
			Name: "tank_mean",
		}))
	})

	It("Should take the new name of a renamed output", func(ctx SpecContext) {
		c := create(ctx, "tank_mean")
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return channels.NewWriter(tx).Rename(ctx, c.Output, "tank_average", false)
		})).To(Succeed())
		Expect(svc.Resources().Retrieve(ctx, node.DB, []resource.ID{
			calculation.ResourceID(c.Key),
		})).To(ConsistOf(HaveField("Name", "tank_average")))
	})

	It("Should announce the calculation of a renamed output, so its name is read again",
		func(ctx SpecContext) {
			c := create(ctx, "tank_mean")
			other := create(ctx, "other_mean")
			rec := &acttestutil.Recorder[calculation.Key, calculation.Action]{}
			DeferCleanup(svc.Actions().OnChange(rec.Record))
			Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return channels.NewWriter(tx).
					Rename(ctx, c.Output, "tank_average", false)
			})).To(Succeed())
			Eventually(rec.Snapshot).Should(ConsistOf(HaveField("Key", c.Key)))
			Expect(rec.Snapshot()).ToNot(ContainElement(HaveField("Key", other.Key)))
		})

	It("Should leave out a key that names no calculation", func(ctx SpecContext) {
		Expect(svc.Resources().Retrieve(ctx, node.DB, []resource.ID{
			calculation.ResourceID(uuid.New()),
		})).To(BeEmpty())
	})

	It("Should list the calculations that except keeps", func(ctx SpecContext) {
		kept := create(ctx, "kept")
		left := create(ctx, "left")
		except := func(id resource.ID) bool { return id == calculation.ResourceID(left.Key) }
		Expect(svc.Resources().List(ctx, node.DB, 10, except)).To(ConsistOf(
			resource.Resource{ID: calculation.ResourceID(kept.Key), Name: "kept"},
		))
	})

	It("Should find no calculation by a search, since its output channel is found",
		func(ctx SpecContext) {
			create(ctx, "tank_mean")
			Expect(svc.Resources().Search(ctx, node.DB, search.Query[resource.ID]{
				Term:  "tank_mean",
				Limit: 10,
			})).To(BeZero())
		})
})
