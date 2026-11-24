// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v14_test

import (
	"context"
	"math"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	servicecolor "github.com/synnaxlabs/synnax/pkg/service/color/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/enumeration"
	enumerationv0 "github.com/synnaxlabs/synnax/pkg/service/enumeration/versions/v0"
	symbol "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v5"
	v13 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v13"
	v14 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v14"
	xcolor "github.com/synnaxlabs/x/color/versions/v2"
	"github.com/synnaxlabs/x/encoding/orc"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

type interned struct {
	name    string
	entries []enumerationv0.Entry
}

func selectNode(key, label string, options ...v13.StateMapping) v13.Node {
	cfg := v13.SelectNodeConfig{Options: options}
	cfg.Label.Label = label
	return v13.Node{Key: key, Config: v13.NodeConfig{Variant: cfg}}
}

func indicatorNode(key, label string, options ...v13.StateMapping) v13.Node {
	cfg := v13.StateIndicatorNodeConfig{Options: options}
	cfg.Label.Label = label
	return v13.Node{Key: key, Config: v13.NodeConfig{Variant: cfg}}
}

func state(key string, value float64, name string, c *xcolor.Color) v13.StateMapping {
	return v13.StateMapping{Key: key, Value: value, Name: name, Color: c}
}

var _ = Describe("Select and state indicator states", func() {
	var (
		calls []interned
		// refusal is the error that the intern returns, or nil to record the call.
		refusal error
	)
	BeforeEach(func() { calls, refusal = nil, nil })
	// store writes old to a new DB.
	store := func(ctx SpecContext, old v13.Schematic) *gorp.DB {
		GinkgoHelper()
		db := DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v13.Key, v13.Schematic]{DB: db}))
		Expect(gorp.NewCreate[v13.Key, v13.Schematic]().Entry(&old).Exec(ctx, db)).
			To(Succeed())
		return db
	}
	// lift opens the v14 table on db, which migrates its schematics with an intern
	// that records each call.
	lift := func(ctx SpecContext, db *gorp.DB) error {
		GinkgoHelper()
		intern := func(
			_ context.Context,
			name string,
			entries []enumerationv0.Entry,
		) (enumerationv0.Key, error) {
			if refusal != nil {
				return enumerationv0.Key{}, refusal
			}
			calls = append(calls, interned{name: name, entries: entries})
			return uuid.New(), nil
		}
		table, err := gorp.OpenTable(ctx, gorp.TableConfig[v14.Key, v14.Schematic]{
			DB: db,
			Migrations: []gorp.Migration{v14.NewMigration(v14.MigrationConfig{
				Intern: func(gorp.Tx) symbol.Intern { return intern },
			})},
		})
		if err == nil {
			DeferClose(table)
		}
		return err
	}
	internAll := func(ctx SpecContext, old v13.Schematic) (v14.Schematic, []interned) {
		GinkgoHelper()
		db := store(ctx, old)
		Expect(lift(ctx, db)).To(Succeed())
		var s v14.Schematic
		Expect(gorp.NewRetrieve[v14.Key, v14.Schematic]().
			Where(gorp.MatchKeys[v14.Key, v14.Schematic](old.Key)).
			Entry(&s).Exec(ctx, db)).To(Succeed())
		return s, calls
	}
	options := func(n v14.Node) []v14.StateMapping {
		GinkgoHelper()
		switch cfg := n.Config.Variant.(type) {
		case v14.SelectNodeConfig:
			return cfg.Options
		case v14.StateIndicatorNodeConfig:
			return cfg.Options
		}
		Fail("not a select or a state indicator")
		return nil
	}

	It("Should give each select and indicator with options a named enumeration",
		func(ctx SpecContext) {
			transparent := &xcolor.Color{R: 9}
			s, calls := internAll(ctx, schematicOf(
				"Valve panel",
				selectNode("mode", "Mode", state("off", 0, "Off", shade(1))),
				indicatorNode("tank", " Tank ", state("idle", 0, "Idle", transparent)),
				selectNode("blank", "  ", state("off", 0, "Off", nil)),
				selectNode("empty", "Unused"),
			))
			Expect(calls).To(HaveLen(3))
			Expect(calls[0]).To(Equal(interned{
				name: "Select: Mode",
				entries: []enumerationv0.Entry{
					{Key: "off", Value: 0, Name: "Off", Color: fixed(1)},
				},
			}))
			Expect(calls[1].name).To(Equal("State indicator: Tank"))
			Expect(calls[1].entries[0].Color).To(Equal(&servicecolor.Ref{
				Variant: servicecolor.FixedRef{Value: *transparent},
			}))
			Expect(calls[2].name).To(Equal("Select: Valve panel"))
			Expect(calls[2].entries[0].Color).To(BeNil())
			for _, n := range s.Nodes {
				Expect(options(n)).To(BeEmpty())
			}
			empty, ok := s.Nodes[3].Config.Variant.(v14.SelectNodeConfig)
			Expect(ok).To(BeTrue())
			Expect(empty.Enumeration).To(BeNil())
		})

	It("Should give an entry with an empty or repeated key a new key",
		func(ctx SpecContext) {
			_, calls := internAll(ctx, schematicOf("Panel", selectNode(
				"a",
				"Mode",
				state("", 0, "Off", nil),
				state("on", 1, "On", nil),
				state("on", 2, "Auto", nil),
			)))
			keys := make([]string, 0, 3)
			for _, e := range calls[0].entries {
				keys = append(keys, e.Key)
			}
			Expect(keys[0]).ToNot(BeEmpty())
			Expect(keys[1]).To(Equal("on"))
			Expect(keys).To(HaveLen(3))
			Expect(keys[2]).ToNot(BeElementOf(keys[0], keys[1]))
		})

	It(
		"Should drop a state whose value is not finite or repeats",
		func(ctx SpecContext) {
			s, calls := internAll(ctx, schematicOf(
				"Panel",
				selectNode(
					"a",
					"Mode",
					state("nan", math.NaN(), "Broken", nil),
					state("on", 1, "On", nil),
					state("again", 1, "Also on", nil),
				),
				selectNode("b", "Spare", state("inf", math.Inf(1), "Broken", nil)),
			))
			Expect(calls).To(HaveLen(1))
			Expect(calls[0].entries).To(HaveLen(1))
			Expect(calls[0].entries[0].Key).To(Equal("on"))
			spare, ok := s.Nodes[1].Config.Variant.(v14.SelectNodeConfig)
			Expect(ok).To(BeTrue())
			Expect(spare.Enumeration).To(BeNil())
		},
	)

	It("Should fail the migration with the node when the intern fails",
		func(ctx SpecContext) {
			refusal = errors.New("refused")
			db := store(ctx, schematicOf(
				"Panel",
				selectNode("mode", "Mode", state("off", 0, "Off", nil)),
			))
			Expect(lift(ctx, db)).To(SatisfyAll(
				MatchError(refusal),
				MatchError(ContainSubstring("node mode")),
			))
		})
})

var _ = Describe("Migrated states", func() {
	var db *gorp.DB
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v13.Key, v13.Schematic]{DB: db}))
	})
	create := func(ctx SpecContext, s v13.Schematic) {
		GinkgoHelper()
		Expect(gorp.NewCreate[v13.Key, v13.Schematic]().Entry(&s).Exec(ctx, db)).
			To(Succeed())
	}
	retrieve := func(ctx SpecContext, key v13.Key) v14.Schematic {
		GinkgoHelper()
		var s v14.Schematic
		Expect(gorp.NewRetrieve[v14.Key, v14.Schematic]().
			Where(gorp.MatchKeys[v14.Key, v14.Schematic](key)).
			Entry(&s).Exec(ctx, db)).To(Succeed())
		return s
	}
	enumerationOf := func(
		ctx SpecContext,
		enums *enumeration.Service,
		n v14.Node,
	) enumeration.Enumeration {
		GinkgoHelper()
		var key *enumerationv0.Key
		switch cfg := n.Config.Variant.(type) {
		case v14.SelectNodeConfig:
			key = cfg.Enumeration
		case v14.StateIndicatorNodeConfig:
			key = cfg.Enumeration
		}
		Expect(key).ToNot(BeNil())
		var e enumeration.Enumeration
		Expect(enums.NewRetrieve().
			Where(enumeration.MatchKeys(*key)).
			Entry(&e).Exec(ctx, db)).To(Succeed())
		return e
	}

	It("Should give the select and indicator that 0.62.0 stored one enumeration",
		func(ctx SpecContext) {
			var stored v13.Schematic
			Expect(orc.Codec.Decode(loadWire("v13_released.hex"), &stored)).
				To(Succeed())
			create(ctx, stored)
			enums := migrate(ctx, db)
			s := retrieve(ctx, stored.Key)
			sel := enumerationOf(ctx, enums, s.Nodes[0])
			Expect(sel.Name).To(Equal("Select: Valve panel"))
			old, ok := stored.Nodes[0].Config.Variant.(v13.SelectNodeConfig)
			Expect(ok).To(BeTrue())
			Expect(sel.Entries).To(HaveLen(2))
			Expect(sel.Entries[0]).To(Equal(enumeration.Entry{
				Key: "closed", Value: 0, Name: "Closed",
			}))
			Expect(sel.Entries[1]).To(Equal(enumeration.Entry{
				Key:   "open",
				Value: 1,
				Name:  "Open",
				Color: &servicecolor.Ref{
					Variant: servicecolor.FixedRef{Value: *old.Options[1].Color},
				},
			}))
			Expect(enumerationOf(ctx, enums, s.Nodes[1]).Key).To(Equal(sel.Key))
		})

	It("Should share one enumeration between equal lists in different schematics",
		func(ctx SpecContext) {
			off, on := state("x", 0, "Off", nil), state("y", 1, "On", nil)
			first := schematicOf("A", selectNode("a", "Mode", off, on))
			// The same list in another order, with other keys.
			off.Key, on.Key = "q", "r"
			second := schematicOf(
				"B",
				selectNode("b", "Mode 2", on, off),
				indicatorNode("c", "Lamp", state("y", 1, "On", shade(2))),
			)
			create(ctx, first)
			create(ctx, second)
			enums := migrate(ctx, db)
			a := enumerationOf(ctx, enums, retrieve(ctx, first.Key).Nodes[0])
			got := retrieve(ctx, second.Key)
			b := enumerationOf(ctx, enums, got.Nodes[0])
			Expect(b.Key).To(Equal(a.Key))
			Expect(b.Name).To(BeElementOf("Select: Mode", "Select: Mode 2"))
			c := enumerationOf(ctx, enums, got.Nodes[1])
			Expect(c.Key).ToNot(Equal(a.Key))
			Expect(c.Name).To(Equal("State indicator: Lamp"))
		})
})

func schematicOf(name string, nodes ...v13.Node) v13.Schematic {
	return v13.Schematic{Key: uuid.New(), Name: name, Nodes: nodes}
}
