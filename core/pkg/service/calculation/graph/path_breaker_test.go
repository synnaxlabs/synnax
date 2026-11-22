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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	graph "github.com/synnaxlabs/synnax/pkg/service/calculation/graph"
	calctestutil "github.com/synnaxlabs/synnax/pkg/service/calculation/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Bare paths in the calculation graph (breaker)", Ordered, func() {
	var (
		pdb     *gorp.DB
		psvc    *channel.Service
		pstatus *status.Service
		stand   group.Key
	)
	const standDir = "heal_stand"
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		node := testutil.NewNode(ctx)
		pdb = node.DB
		groups := MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: pdb}))
		MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: pdb}))
		pstatus = MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: pdb}))
		psvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
			Channel: node.Channel,
			TS:      node.Storage.TS,
			DB:      pdb,
			HostKey: node.Identity.HostKey,
			Status:  pstatus,
		}))
		g := group.Group{Name: standDir, Scope: resource.TypeChannel}
		Expect(pdb.WithTx(ctx, func(tx gorp.Tx) error {
			return groups.NewWriter(tx).Create(ctx, &g)
		})).To(Succeed())
		stand = g.Key
	})
	createInStand := func(ctx SpecContext, name string) channel.Channel {
		GinkgoHelper()
		ch := channel.Channel{
			Name:     name,
			DataType: telem.Int64T,
			Virtual:  true,
			Group:    &stand,
		}
		Expect(pdb.WithTx(ctx, func(tx gorp.Tx) error {
			return psvc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		return ch
	}
	deleteKey := func(ctx SpecContext, key channel.Key) {
		GinkgoHelper()
		Expect(pdb.WithTx(ctx, func(tx gorp.Tx) error {
			return psvc.NewWriter(tx).Delete(ctx, key, false)
		})).To(Succeed())
	}
	hasStatus := func(ctx SpecContext, key channel.Key) bool {
		var statuses []calculation.Status
		Expect(pstatus.NewRetrieve[struct{}]().
			Where(status.MatchOwners[struct{}](channel.ResourceID(key))).
			Entries(&statuses).
			Exec(ctx, pdb)).To(Succeed())
		return len(statuses) > 0
	}

	DescribeTable("G1 heals a calculation when the channel at its path is made again",
		func(ctx SpecContext, dep, ref string) {
			MustOpen(graph.Open(ctx, graph.Config{
				DB:      pdb,
				Channel: psvc,
				Status:  pstatus,
			}))
			first := createInStand(ctx, dep)
			calc := MustSucceed(calctestutil.Creator{DB: pdb, Channel: psvc}.Create(
				ctx,
				calctestutil.Calculation{
					Name:       dep + "_calc",
					DataType:   telem.Int64T,
					Expression: "return " + ref + " + 1",
				},
			))
			By("Deleting the channel that the calculation reads")
			deleteKey(ctx, first.Key())
			Eventually(func() bool { return hasStatus(ctx, calc.Key()) },
				2*time.Second, 10*time.Millisecond).Should(BeTrue())
			By("Making the channel again under the same path")
			second := createInStand(ctx, dep)
			Expect(second.Key()).ToNot(Equal(first.Key()))
			Eventually(func() bool { return hasStatus(ctx, calc.Key()) },
				2*time.Second, 10*time.Millisecond).Should(BeFalse(),
				"the calculation that reads %s still has an error status", ref)
		},
		Entry("control: a bare name", "heal_name", "heal_name"),
		Entry("a bare path", "heal_path", standDir+".heal_path"),
	)
})
