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
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	storagetestutil "github.com/synnaxlabs/synnax/pkg/storage/testutil"
	"github.com/synnaxlabs/x/defaults"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// A 0.62 import kept the raw channel keys of another Core in schematics and tables.
// Each spec writes them with a Core, then starts this Core on the store as a 0.62
// store, which runs v63_clear_unissued_channel_links.
var _ = Describe("ClearUnissuedChannelLinks", func() {
	const step = "v63_clear_unissued_channel_links"
	appliedKey := []byte("gorp.migration.Composition")
	type core struct {
		node    testutil.Node
		layer   *service.Layer
		db      *gorp.DB
		stopped *bool
	}
	var (
		store storagetestutil.Layer
		proj  project.Project
	)
	// openStore opens the store of the spec, which its Cores open on in turn. ctx must
	// be
	// the spec's, since the store runs until it ends.
	openStore := func(ctx SpecContext) {
		GinkgoHelper()
		store = storagetestutil.OpenLayer(ctx)
		DeferCleanup(func() { Expect(store.Close()).To(Succeed()) })
	}
	stop := func(c core) {
		GinkgoHelper()
		if *c.stopped {
			return
		}
		*c.stopped = true
		Expect(c.layer.Close()).To(Succeed())
		Expect(c.node.Close()).To(Succeed())
	}
	start := func(ctx SpecContext) core {
		GinkgoHelper()
		n := testutil.OpenNode(ctx, distribution.LayerConfig{Storage: store.Layer})
		c := core{
			node:    n,
			layer:   MustSucceed(svctestutil.OpenLayer(ctx, n)),
			db:      n.DB,
			stopped: new(bool),
		}
		DeferCleanup(func() { stop(c) })
		return c
	}
	applied := func() []string {
		GinkgoHelper()
		b, closer := MustSucceed2(store.KV.Get(appliedKey))
		var keys []string
		Expect(json.Unmarshal(b, &keys)).To(Succeed())
		Expect(closer.Close()).To(Succeed())
		return keys
	}
	// openOld starts the Core that writes the 0.62 records, with a project for them.
	openOld := func(ctx SpecContext) core {
		GinkgoHelper()
		openStore(ctx)
		c := start(ctx)
		proj = project.Project{Name: "Upgraded"}
		Expect(c.db.WithTx(ctx, func(tx gorp.Tx) error {
			return c.layer.Project.NewWriter(tx).Create(ctx, &proj)
		})).To(Succeed())
		return c
	}
	// clearLinks stops old, makes its store a 0.62 store, which has not applied the
	// step and holds no first-open record of a key counter, and starts this Core on it.
	// The step then runs as a Core that opened after every channel so far.
	clearLinks := func(ctx SpecContext, old core) core {
		GinkgoHelper()
		stop(old)
		keys := slices.DeleteFunc(applied(), func(k string) bool { return k == step })
		Expect(store.KV.Set(ctx, appliedKey, MustSucceed(json.Marshal(keys)))).
			To(Succeed())
		iter := MustSucceed(store.KV.OpenIterator(kv.IterPrefix(nil)))
		var firstOpen [][]byte
		for ok := iter.First(); ok; ok = iter.Next() {
			if bytes.HasSuffix(iter.Key(), []byte(".firstOpen")) {
				firstOpen = append(firstOpen, slices.Clone(iter.Key()))
			}
		}
		Expect(iter.Close()).To(Succeed())
		Expect(firstOpen).ToNot(BeEmpty())
		for _, k := range firstOpen {
			Expect(store.KV.Delete(ctx, k)).To(Succeed())
		}
		c := start(ctx)
		Expect(applied()).To(ContainElement(step))
		return c
	}

	createChannel := func(ctx SpecContext, c core) channel.Channel {
		GinkgoHelper()
		ch := channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.Float32T,
			Virtual:  true,
		}
		Expect(c.db.WithTx(ctx, func(tx gorp.Tx) error {
			return c.layer.Channel.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		return ch
	}

	// storeUpgraded writes a schematic and a table that hold keys, as a 0.62 import
	// left
	// them: with no reference check.
	storeUpgraded := func(
		ctx SpecContext,
		c core,
		keys ...channel.Key,
	) (schematic.Schematic, table.Table) {
		GinkgoHelper()
		sch := schematic.Schematic{
			Key:     uuid.New(),
			Name:    "imported",
			Project: &proj.Key,
		}
		tbl := table.Table{Key: uuid.New(), Name: "imported", Project: &proj.Key}
		row := table.Row{Key: "r0", Size: 30}
		for i, k := range keys {
			var value schematic.ValueNodeConfig
			value.SetDefaults()
			value.Channel = k
			sch.Nodes = append(sch.Nodes, schematic.Node{
				Key:    fmt.Sprintf("value_%d", i),
				Config: schematic.NodeConfig{Variant: value},
			})
			cfg := defaults.Fill(table.CellConfig{
				Variant: table.ValueCellConfig{Channel: k},
			})
			tbl.Columns = append(tbl.Columns, table.Column{
				Key:  fmt.Sprintf("c%d", i),
				Size: 80,
			})
			row.Cells = append(row.Cells, table.Cell{
				Key:    fmt.Sprintf("cell_%d", i),
				Config: cfg,
			})
		}
		tbl.Rows = []table.Row{row}
		Expect(c.db.WithTx(ctx, func(tx gorp.Tx) error {
			if err := gorp.WrapWriter[schematic.Key, schematic.Schematic](tx).
				Set(ctx, sch); err != nil {
				return err
			}
			return gorp.WrapWriter[table.Key, table.Table](tx).Set(ctx, tbl)
		})).To(Succeed())
		return sch, tbl
	}

	schematicKeys := func(ctx SpecContext, c core, k schematic.Key) []channel.Key {
		GinkgoHelper()
		var sch schematic.Schematic
		Expect(c.layer.Schematic.NewRetrieve().
			Where(schematic.MatchKeys(k)).
			Entry(&sch).
			Exec(ctx, c.db)).To(Succeed())
		keys := make([]channel.Key, len(sch.Nodes))
		for i, n := range sch.Nodes {
			value, ok := n.Config.Variant.(schematic.ValueNodeConfig)
			Expect(ok).To(BeTrue())
			keys[i] = value.Channel
		}
		return keys
	}

	tableKeys := func(ctx SpecContext, c core, k table.Key) []channel.Key {
		GinkgoHelper()
		var tbl table.Table
		Expect(c.layer.Table.NewRetrieve().
			Where(table.MatchKeys(k)).
			Entry(&tbl).
			Exec(ctx, c.db)).To(Succeed())
		keys := make([]channel.Key, len(tbl.Rows[0].Cells))
		for i, cell := range tbl.Rows[0].Cells {
			value, ok := cell.Config.Variant.(table.ValueCellConfig)
			Expect(ok).To(BeTrue())
			keys[i] = value.Channel
		}
		return keys
	}

	It(
		"Should keep its key, which the applied set holds forever",
		func(ctx SpecContext) {
			openStore(ctx)
			start(ctx)
			Expect(applied()).To(ContainElement(step))
		},
	)

	It("Should clear a never-issued key and keep a deleted and a live key", func(
		ctx SpecContext,
	) {
		old := openOld(ctx)
		live := createChannel(ctx, old)
		dead := createChannel(ctx, old)
		Expect(old.db.WithTx(ctx, func(tx gorp.Tx) error {
			return old.layer.Channel.NewWriter(tx).Delete(ctx, dead.Key(), false)
		})).To(Succeed())
		never := distchannel.NewKey(live.Leaseholder, live.LocalKey+1000)
		sch, tbl := storeUpgraded(ctx, old, never, dead.Key(), live.Key())
		c := clearLinks(ctx, old)
		Expect(schematicKeys(ctx, c, sch.Key)).To(Equal(
			[]channel.Key{0, dead.Key(), live.Key()},
		))
		Expect(tableKeys(ctx, c, tbl.Key)).To(Equal(
			[]channel.Key{0, dead.Key(), live.Key()},
		))
	})

	It("Should keep a key leased to another node", func(ctx SpecContext) {
		old := openOld(ctx)
		live := createChannel(ctx, old)
		other := distchannel.NewKey(node.Key(2), live.LocalKey+500)
		sch, tbl := storeUpgraded(ctx, old, other)
		c := clearLinks(ctx, old)
		Expect(schematicKeys(ctx, c, sch.Key)).To(Equal([]channel.Key{other}))
		Expect(tableKeys(ctx, c, tbl.Key)).To(Equal([]channel.Key{other}))
	})

	It("Should let a write hold a key leased to another node", func(ctx SpecContext) {
		c := openOld(ctx)
		live := createChannel(ctx, c)
		var value schematic.ValueNodeConfig
		value.SetDefaults()
		value.Channel = distchannel.NewKey(node.Key(2), live.LocalKey+600)
		sch := schematic.Schematic{
			Name:    "other node",
			Project: &proj.Key,
			Nodes: []schematic.Node{{
				Key:    "value",
				Config: schematic.NodeConfig{Variant: value},
			}},
		}
		Expect(c.db.WithTx(ctx, func(tx gorp.Tx) error {
			return c.layer.Schematic.NewWriter(tx).Create(ctx, &sch)
		})).To(Succeed())
	})

	It("Should let an upgraded schematic be copied", func(ctx SpecContext) {
		old := openOld(ctx)
		live := createChannel(ctx, old)
		sch, _ := storeUpgraded(
			ctx,
			old,
			distchannel.NewKey(live.Leaseholder, live.LocalKey+2000),
		)
		c := clearLinks(ctx, old)
		var cp schematic.Schematic
		Expect(c.db.WithTx(ctx, func(tx gorp.Tx) error {
			return c.layer.Schematic.NewWriter(tx).Copy(ctx, sch.Key, "copy", &cp)
		})).To(Succeed())
	})

	It("Should let the removal of an upgraded symbol be undone", func(ctx SpecContext) {
		old := openOld(ctx)
		live := createChannel(ctx, old)
		sch, _ := storeUpgraded(
			ctx,
			old,
			distchannel.NewKey(live.Leaseholder, live.LocalKey+3000),
		)
		c := clearLinks(ctx, old)
		apply := func(act schematic.Action) error {
			return c.db.WithTx(ctx, func(tx gorp.Tx) error {
				_, err := c.layer.Schematic.NewWriter(tx).Apply(ctx, sch.Key, act)
				return err
			})
		}
		var stored schematic.Schematic
		Expect(c.layer.Schematic.NewRetrieve().
			Where(schematic.MatchKeys(sch.Key)).
			Entry(&stored).
			Exec(ctx, c.db)).To(Succeed())
		Expect(apply(schematic.NewRemoveNodeAction(
			schematic.RemoveNodePayload{Key: "value_0"},
		))).To(Succeed())
		Expect(apply(schematic.NewSetNodeAction(
			schematic.SetNodePayload{Node: stored.Nodes[0]},
		))).To(Succeed())
	})

	It("Should not bind an upgraded symbol to the next channel the Core creates", func(
		ctx SpecContext,
	) {
		old := openOld(ctx)
		last := createChannel(ctx, old)
		never := distchannel.NewKey(last.Leaseholder, last.LocalKey+1)
		sch, _ := storeUpgraded(ctx, old, never)
		c := clearLinks(ctx, old)
		next := createChannel(ctx, c)
		Expect(next.Key()).To(Equal(never))
		uses := MustSucceed(c.layer.Channel.Uses(ctx, c.db, []channel.Key{next.Key()}))
		Expect(uses[0]).ToNot(ContainElement(HaveField(
			"Holder", gorp.NewTarget("schematic", sch.Key),
		)))
	})
})
