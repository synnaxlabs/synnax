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
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	storagetestutil "github.com/synnaxlabs/synnax/pkg/storage/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Checker heal r2: a Core start", func() {
	It("Should heal a schematic onto a built-in that the start declares at the path "+
		"of the channel it held", func(ctx SpecContext) {
		store := DeferClose(storagetestutil.OpenLayer(ctx))
		type core struct {
			node  testutil.Node
			layer *service.Layer
		}
		start := func() core {
			GinkgoHelper()
			n := testutil.OpenNode(ctx, distribution.LayerConfig{Storage: store.Layer})
			return core{node: n, layer: MustSucceed(svctestutil.OpenLayer(ctx, n))}
		}
		stop := func(c core) {
			GinkgoHelper()
			Expect(c.layer.Close()).To(Succeed())
			Expect(c.node.Close()).To(Succeed())
		}
		byName := func(c core, name string) channel.Channel {
			GinkgoHelper()
			var ch channel.Channel
			Expect(c.layer.Channel.NewRetrieve().
				Where(channel.MatchNames(name)).
				Entry(&ch).
				Exec(ctx, c.node.DB)).To(Succeed())
			return ch
		}
		tombstoned := func(c core, k channel.Key) bool {
			GinkgoHelper()
			return MustSucceed(gorp.NewRetrieve[channel.Key, channel.Tombstone]().
				Where(gorp.MatchKeys[channel.Key, channel.Tombstone](k)).
				Exists(ctx, c.node.DB))
		}

		// A Core whose user channel sits at the path, group and name, of a built-in
		// that a later start declares.
		old := start()
		running := true
		DeferCleanup(func() {
			if running {
				stop(old)
			}
		})
		const name = "sy_node_1_metrics_total_size_gb"
		builtIn := byName(old, name)
		Expect(old.node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return old.layer.Channel.NewWriter(tx).Delete(ctx, builtIn.Key(), false)
		})).To(Succeed())
		user := channel.Channel{
			Name:     name,
			Group:    builtIn.Group,
			DataType: telem.Float32T,
			Virtual:  true,
		}
		proj := project.Project{Name: "heal-start"}
		var value schematic.ValueNodeConfig
		value.SetDefaults()
		Expect(old.node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			if err := old.layer.Channel.NewWriter(tx).Create(ctx, &user); err != nil {
				return err
			}
			return old.layer.Project.NewWriter(tx).Create(ctx, &proj)
		})).To(Succeed())
		value.Channel = user.Key()
		sch := schematic.Schematic{
			Name:    "heal-start",
			Project: &proj.Key,
			Nodes: []schematic.Node{{
				Key:    "value",
				Config: schematic.NodeConfig{Variant: value},
			}},
		}
		Expect(old.node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return old.layer.Schematic.NewWriter(tx).Create(ctx, &sch)
		})).To(Succeed())
		Expect(old.node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return old.layer.Channel.NewWriter(tx).Delete(ctx, user.Key(), false)
		})).To(Succeed())
		Expect(tombstoned(old, user.Key())).To(BeTrue())
		stop(old)
		running = false

		next := start()
		DeferCleanup(stop, next)
		declared := byName(next, name)
		Expect(declared.Key()).ToNot(Equal(user.Key()))
		var stored schematic.Schematic
		Expect(next.layer.Schematic.NewRetrieve().
			Where(schematic.MatchKeys(sch.Key)).
			Entry(&stored).
			Exec(ctx, next.node.DB)).To(Succeed())
		healed, ok := stored.Nodes[0].Config.Variant.(schematic.ValueNodeConfig)
		Expect(ok).To(BeTrue())
		Expect(healed.Channel).To(Equal(declared.Key()),
			"the start declares the built-in at the path of the deleted channel, so "+
				"the heal after the tables open moves the schematic onto it")
		Expect(tombstoned(next, user.Key())).To(BeFalse())
		Expect(declared.Rewires).To(HaveLen(1),
			"the heal gives the built-in the list of the tombstone at its path")
	})
})
