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
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Retarget breaker", Ordered, func() {
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
		proj = project.Project{Name: "Retarget breaker"}
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

	It("Should leave an unset state channel unset when the map names the zero key",
		func(ctx SpecContext) {
			a, b, c := createChannel(ctx), createChannel(ctx), createChannel(ctx)
			nodes := make([]schematic.Node, 2)
			for i, state := range []channel.Key{a, 0} {
				var valve schematic.ValveNodeConfig
				valve.SetDefaults()
				valve.StateChannel = state
				nodes[i] = schematic.Node{
					Key:    "valve" + string(rune('a'+i)),
					Config: schematic.NodeConfig{Variant: valve},
				}
			}
			sch := schematic.Schematic{Name: "zero", Project: &proj.Key, Nodes: nodes}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Schematic.NewWriter(tx).Create(ctx, &sch)
			})).To(Succeed())
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Channel.NewWriter(tx).Retarget(
					ctx,
					map[channel.Key]channel.Key{a: b, 0: c},
					[]gorp.Dependent{{
						Holder:    gorp.NewTarget("schematic", sch.Key),
						Reference: "nodes.config.state_channel",
					}},
				)
			})).To(Succeed())
			var stored schematic.Schematic
			Expect(layer.Schematic.NewRetrieve().
				Where(schematic.MatchKeys(sch.Key)).
				Entry(&stored).
				Exec(ctx, db)).To(Succeed())
			states := make([]channel.Key, len(stored.Nodes))
			for i, n := range stored.Nodes {
				states[i] = n.Config.Variant.(schematic.ValveNodeConfig).StateChannel
			}
			Expect(states).To(Equal([]channel.Key{b, 0}))
		},
	)
})
