// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package schematic_test

import (
	"fmt"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channeltestutil "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("DanglingMigration", func() {
	It("Should clear each channel key that issued reports as never issued", func(
		ctx SpecContext,
	) {
		cleared := MustSucceed(channeltestutil.CreateVirtual(ctx, db, channelSvc, 1))[0]
		sch := schematic.Schematic{
			Key:     uuid.New(),
			Name:    "imported",
			Project: &proj.Key,
		}
		for i, k := range []channel.Key{ch1, cleared} {
			var value schematic.ValueNodeConfig
			value.SetDefaults()
			value.Channel = k
			sch.Nodes = append(sch.Nodes, schematic.Node{
				Key:    fmt.Sprintf("value_%d", i),
				Config: schematic.NodeConfig{Variant: value},
			})
		}
		Expect(gorp.WrapWriter[schematic.Key, schematic.Schematic](db).Set(ctx, sch)).
			To(Succeed())
		issued := func(t gorp.Target) (bool, error) {
			return t != gorp.NewTarget("channel", cleared), nil
		}
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "schematic_dangling",
			Migrations: []gorp.Migration{svc.DanglingMigration("clear", issued)},
		})).To(Succeed())
		var got schematic.Schematic
		Expect(svc.NewRetrieve().
			Where(schematic.MatchKeys(sch.Key)).
			Entry(&got).
			Exec(ctx, db)).To(Succeed())
		channels := make([]channel.Key, len(got.Nodes))
		for i, n := range got.Nodes {
			value, ok := n.Config.Variant.(schematic.ValueNodeConfig)
			Expect(ok).To(BeTrue())
			channels[i] = value.Channel
		}
		Expect(channels).To(Equal([]channel.Key{ch1, 0}))
	})
})
