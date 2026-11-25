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
	"embed"
	"encoding/hex"
	"slices"
	"strings"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	gorptestutil "github.com/synnaxlabs/gorp/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	servicecolor "github.com/synnaxlabs/synnax/pkg/service/color"
	"github.com/synnaxlabs/synnax/pkg/service/enumeration"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	symbolversions "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions"
	symbolv0 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v0"
	v4 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v4"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/versions"
	v13 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v13"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/encoding/orc"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/spatial"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/types"
)

//go:embed testdata/released/v0.62.0/*.hex
var released embed.FS

// openStore returns a DB on kvDB that holds the schematic migrations through
// schematicLast and the symbol migrations through symbolLast as applied. An empty key
// applies none.
func openStore(ctx SpecContext, kvDB kv.DB, schematicLast, symbolLast string) *gorp.DB {
	GinkgoHelper()
	db := DeferClose(gorp.Wrap(kvDB))
	StartRun(db.Run)
	apply := func(namespace string, chain []gorp.Migration, last string) {
		if last == "" {
			return
		}
		i := slices.IndexFunc(chain, func(m gorp.Migration) bool {
			return m.Key() == last
		})
		Expect(i).ToNot(Equal(-1))
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  namespace,
			Migrations: chain[:i+1],
		})).To(Succeed())
	}
	apply(
		types.Name[schematic.Schematic](),
		versions.NewMigrations(versions.MigrationsConfig{}),
		schematicLast,
	)
	apply(
		types.Name[symbol.Symbol](),
		symbolversions.NewMigrations(symbolversions.MigrationsConfig{}),
		symbolLast,
	)
	return db
}

// openShipped returns a DB that holds the migrations that core/v0.62.0 ran.
func openShipped(ctx SpecContext) *gorp.DB {
	GinkgoHelper()
	return openStore(
		ctx,
		memkv.New(),
		v13.NameMigration.Key(),
		v4.NameMigration.Key(),
	)
}

// retrieveSchematic returns the stored schematic with key.
func retrieveSchematic(
	ctx SpecContext,
	db *gorp.DB,
	key schematic.Key,
) schematic.Schematic {
	GinkgoHelper()
	var sc schematic.Schematic
	Expect(gorp.NewRetrieve[schematic.Key, schematic.Schematic]().
		Where(gorp.MatchKeys[schematic.Key, schematic.Schematic](key)).
		Entry(&sc).
		Exec(ctx, db)).To(Succeed())
	return sc
}

// openService opens the schematic service on db, which runs its migrations.
func openService(ctx SpecContext, db *gorp.DB) {
	GinkgoHelper()
	enums := MustOpen(enumeration.OpenService(ctx, enumeration.ServiceConfig{DB: db}))
	groups := MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: db}))
	MustOpen(schematic.OpenService(ctx, schematic.ServiceConfig{
		DB:          db,
		Group:       groups,
		ImEx:        imex.NewService(),
		Enumeration: enums,
	}))
}

// decodeReleased decodes the record that core/v0.62.0 stored in the named file.
func decodeReleased[T any](name string) T {
	GinkgoHelper()
	raw := MustSucceed(released.ReadFile("testdata/released/v0.62.0/" + name))
	b := MustSucceed(hex.DecodeString(strings.ReplaceAll(string(raw), "\n", "")))
	var v T
	Expect(orc.Codec.Decode(b, &v)).To(Succeed())
	return v
}

var _ = Describe("Stored v4 symbols", func() {
	It("Should give the commands of an actuator to each symbol that a custom_actuator "+
		"node draws", func(ctx SpecContext) {
		db := openShipped(ctx)
		spec := v4.Spec{
			SVG:     `<svg><rect id="b"/></svg>`,
			Variant: "static",
			Scale:   1,
			States:  []v4.State{{Key: "off", Name: "Off"}, {Key: "on", Name: "On"}},
		}
		drawn := v4.Symbol{Key: uuid.New(), Name: "Drawn", Data: spec}
		static := v4.Symbol{Key: uuid.New(), Name: "Static", Data: spec}
		Expect(gorp.NewCreate[v4.Key, v4.Symbol]().
			Entries(&[]v4.Symbol{drawn, static}).
			Exec(ctx, db)).To(Succeed())
		node := func(key string, v v13.NodeConfigVariant) v13.Node {
			return v13.Node{Key: key, Config: v13.NodeConfig{Variant: v}}
		}
		sc := v13.Schematic{
			Key:  uuid.New(),
			Name: "Schematic",
			Nodes: []v13.Node{
				node("a", v13.CustomActuatorNodeConfig{SpecKey: drawn.Key.String()}),
				node("b", v13.CustomStaticNodeConfig{SpecKey: static.Key.String()}),
				node("c", v13.CustomActuatorNodeConfig{SpecKey: "not-a-uuid"}),
			},
		}
		Expect(gorp.NewCreate[v13.Key, v13.Schematic]().
			Entry(&sc).
			Exec(ctx, db)).To(Succeed())
		openService(ctx, db)
		retrieve := func(key symbol.Key) symbol.Symbol {
			GinkgoHelper()
			var sym symbol.Symbol
			Expect(gorp.NewRetrieve[symbol.Key, symbol.Symbol]().
				Where(gorp.MatchKeys[symbol.Key, symbol.Symbol](key)).
				Entry(&sym).
				Exec(ctx, db)).To(Succeed())
			return sym
		}
		Expect(retrieve(drawn.Key).Data.Commands).To(HaveLen(2))
		Expect(retrieve(static.Key).Data.Commands).To(BeEmpty())
	})

	It("Should make a symbol with one state that a custom_actuator node draws an "+
		"actuator that draws value 1 as value 0", func(ctx SpecContext) {
		db := openShipped(ctx)
		blue := color.Color{R: 40, G: 80, B: 220, A: 255}
		tank := v4.Symbol{Key: uuid.New(), Name: "Tank", Data: v4.Spec{
			SVG:     `<svg><rect class="body"/></svg>`,
			Variant: "static",
			Scale:   1,
			States: []v4.State{{Key: "base", Name: "Base", Regions: []v4.Region{{
				Key:       "body",
				Name:      "Body",
				Selectors: []string{".body"},
				FillColor: &blue,
			}}}},
		}}
		Expect(gorp.NewCreate[v4.Key, v4.Symbol]().Entry(&tank).Exec(ctx, db)).
			To(Succeed())
		sc := v13.Schematic{
			Key:  uuid.New(),
			Name: "Schematic",
			Nodes: []v13.Node{{Key: "a", Config: v13.NodeConfig{
				Variant: v13.CustomActuatorNodeConfig{SpecKey: tank.Key.String()},
			}}},
		}
		Expect(gorp.NewCreate[v13.Key, v13.Schematic]().Entry(&sc).Exec(ctx, db)).
			To(Succeed())
		openService(ctx, db)
		var sym symbol.Symbol
		Expect(gorp.NewRetrieve[symbol.Key, symbol.Symbol]().
			Where(gorp.MatchKeys[symbol.Key, symbol.Symbol](tank.Key)).
			Entry(&sym).
			Exec(ctx, db)).To(Succeed())
		Expect(sym.Data.Variant).To(Equal("actuator"))
		Expect(sym.Data.Looks).To(HaveLen(1))
		Expect(sym.Data.Looks[0].When).To(Equal(&symbol.Condition{
			Variant: symbol.EqualsCondition{State: "active", Values: []float64{0, 1}},
		}))
		state := as[symbol.EnumerationState](sym.Data.States[0].Variant)
		var e enumeration.Enumeration
		Expect(gorp.NewRetrieve[enumeration.Key, enumeration.Enumeration]().
			Where(gorp.MatchKeys[enumeration.Key, enumeration.Enumeration](
				*state.Enumeration,
			)).
			Entry(&e).
			Exec(ctx, db)).To(Succeed())
		Expect(e.Entries).To(Equal([]enumeration.Entry{
			{Key: "base", Value: 0, Name: "Base"},
			{Key: "on", Value: 1, Name: "On"},
		}))
	})

	It("Should drop the state binding of a custom_static node whose symbol has no "+
		"states", func(ctx SpecContext) {
		db := openShipped(ctx)
		plain := v4.Symbol{
			Key:  uuid.New(),
			Name: "Plain",
			Data: v4.Spec{SVG: "<svg/>", Variant: "static", Scale: 1},
		}
		Expect(gorp.NewCreate[v4.Key, v4.Symbol]().Entry(&plain).Exec(ctx, db)).
			To(Succeed())
		sc := v13.Schematic{
			Key:  uuid.New(),
			Name: "Schematic",
			Nodes: []v13.Node{{Key: "a", Config: v13.NodeConfig{
				Variant: v13.CustomStaticNodeConfig{SpecKey: plain.Key.String()},
			}}},
		}
		Expect(gorp.NewCreate[v13.Key, v13.Schematic]().Entry(&sc).Exec(ctx, db)).
			To(Succeed())
		openService(ctx, db)
		n := retrieveSchematic(ctx, db, sc.Key).Nodes[0]
		cfg, ok := n.Config.Variant.(schematic.SymbolNodeConfig)
		Expect(ok).To(BeTrue())
		Expect(cfg.Spec.Variant).To(Equal(symbol.CustomRef{Key: plain.Key}))
		Expect(cfg.States).To(BeEmpty())
	})
})

var _ = Describe("A store from before v0.54", func() {
	It("Should keep a node on a symbol stored under the pre-v0.54 key format",
		func(ctx SpecContext) {
			kvDB := memkv.New()
			db := openStore(ctx, kvDB, v13.NameMigration.Key(), "")
			sym := symbolv0.Symbol{
				Key:  uuid.New(),
				Name: "Valve",
				Data: msgpack.EncodedJSON{
					"svg":     "<svg/>",
					"variant": "static",
					"scale":   1,
				},
			}
			gorptestutil.SetPreV54Row(ctx, kvDB, "Symbol", sym.Key, sym)
			sc := v13.Schematic{
				Key:  uuid.New(),
				Name: "Schematic",
				Nodes: []v13.Node{{Key: "a", Config: v13.NodeConfig{
					Variant: v13.CustomStaticNodeConfig{SpecKey: sym.Key.String()},
				}}},
			}
			Expect(gorp.NewCreate[v13.Key, v13.Schematic]().Entry(&sc).Exec(ctx, db)).
				To(Succeed())
			openService(ctx, db)
			n := retrieveSchematic(ctx, db, sc.Key).Nodes[0]
			cfg, ok := n.Config.Variant.(schematic.SymbolNodeConfig)
			Expect(ok).To(BeTrue())
			Expect(cfg.Spec.Variant).To(Equal(symbol.CustomRef{Key: sym.Key}))
		},
	)
})

var _ = Describe("A store from core/v0.62.0", func() {
	It("Should draw each equipment node as a symbol node", func(ctx SpecContext) {
		db := openShipped(ctx)
		vent := decodeReleased[v4.Symbol]("symbol_vent_valve.hex")
		tank := decodeReleased[v4.Symbol]("symbol_tank.hex")
		Expect(gorp.NewCreate[v4.Key, v4.Symbol]().
			Entries(&[]v4.Symbol{vent, tank}).
			Exec(ctx, db)).To(Succeed())
		stored := decodeReleased[v13.Schematic]("equipment.hex")
		Expect(gorp.NewCreate[v13.Key, v13.Schematic]().
			Entry(&stored).
			Exec(ctx, db)).To(Succeed())
		openService(ctx, db)
		var sc schematic.Schematic
		Expect(gorp.NewRetrieve[schematic.Key, schematic.Schematic]().
			Where(gorp.MatchKeys[schematic.Key, schematic.Schematic](stored.Key)).
			Entry(&sc).
			Exec(ctx, db)).To(Succeed())
		Expect(sc.Nodes).To(HaveLen(len(stored.Nodes)))
		names := set.New(lo.Map(
			symbol.Builtins(),
			func(b symbol.Builtin, _ int) string { return b.Name },
		)...)
		nodes := map[string]schematic.SymbolNodeConfig{}
		for _, n := range sc.Nodes {
			cfg, ok := n.Config.Variant.(schematic.SymbolNodeConfig)
			if !ok {
				continue
			}
			if ref, ok := cfg.Spec.Variant.(symbol.BuiltinRef); ok {
				Expect(names.Contains(ref.Name)).To(BeTrue(), ref.Name)
			}
			nodes[cfg.Label.Label] = cfg
		}
		// The 3 lights with an on color and 5 custom nodes with overrides stay.
		Expect(nodes).To(HaveLen(len(stored.Nodes) - 8))
		match := func(label string) *symbol.Matcher {
			GinkgoHelper()
			Expect(nodes[label].States).To(HaveLen(1))
			return nodes[label].States[0].Match
		}
		Expect(match("Default threshold")).To(BeNil())
		Expect(match("Empty threshold")).To(BeNil())
		within := &symbol.Matcher{Variant: symbol.WithinMatcher{
			Bounds: spatial.Bounds{Lower: 10, Upper: 20},
		}}
		Expect(match("Differing threshold")).To(Equal(within))
		Expect(match("Inverted threshold")).To(Equal(within))
		Expect(nodes["custom_actuator"].Spec.Variant).
			To(Equal(symbol.CustomRef{Key: vent.Key}))
		Expect(nodes["custom_static"].Spec.Variant).
			To(Equal(symbol.CustomRef{Key: tank.Key}))
		Expect(nodes["Missing static"].Spec.Variant).
			To(Equal(symbol.MissingRef{Key: "e040ea6d-24f5-4e8d-ba3a-fc1143909e72"}))
		Expect(nodes["Normally open"].Spec.Variant).
			To(Equal(symbol.BuiltinRef{Name: "normally_open_solenoid_valve"}))
		Expect(nodes["Red filter"].Scale).To(Equal(1.5))
		red := color.Color{R: 220, G: 40, B: 40, A: 255}
		Expect(nodes["Red filter"].Color).To(HaveValue(Equal(servicecolor.Ref{
			Variant: servicecolor.FixedRef{Value: red},
		})))
		source := func(label string) schematic.SourceVariant {
			GinkgoHelper()
			Expect(nodes[label].States).To(HaveLen(1))
			return nodes[label].States[0].Source.Variant
		}
		// The capture's channel keys: valve_cmd, then gone_state, which it deleted.
		command, deleted := channel.Key(1048584), channel.Key(1048586)
		Expect(source("Deleted channel")).To(Equal(schematic.ChannelSource{
			Key:     &deleted,
			Command: &command,
		}))
		Expect(source("No channels")).To(Equal(schematic.ChannelSource{}))
		Expect(nodes["Styled pump"].StalenessTimeout).To(Equal(10 * telem.Second))
		Expect(nodes["Styled pump"].Control.Authority).To(HaveValue(Equal(uint8(200))))
		on := nodes["Manual enabled=True clickable=False"]
		Expect(on.Disabled).To(BeTrue())
		Expect(source("Manual enabled=True clickable=False")).
			To(Equal(schematic.LocalSource{Value: 1}))
		Expect(nodes["Manual enabled=False clickable=True"].Disabled).To(BeFalse())
	})
})

var _ = Describe("Stored v13 selects", func() {
	It("Should give each select an enumeration for its options when the service "+
		"opens", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		schematics := versions.NewMigrations(versions.MigrationsConfig{})
		shipped := slices.IndexFunc(schematics, func(m gorp.Migration) bool {
			return m.Key() == v13.NameMigration.Key()
		})
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  types.Name[schematic.Schematic](),
			Migrations: schematics[:shipped+1],
		})).To(Succeed())
		sc := v13.Schematic{
			Key:  uuid.New(),
			Name: "Panel",
			Nodes: []v13.Node{{Key: "a", Config: v13.NodeConfig{
				Variant: v13.SelectNodeConfig{Options: []v13.StateMapping{
					{Key: "on", Name: "On", Value: 1},
				}},
			}}},
		}
		Expect(gorp.NewCreate[v13.Key, v13.Schematic]().
			Entry(&sc).
			Exec(ctx, db)).To(Succeed())
		enums := MustOpen(
			enumeration.OpenService(ctx, enumeration.ServiceConfig{DB: db}),
		)
		groups := MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: db}))
		svc := MustOpen(schematic.OpenService(ctx, schematic.ServiceConfig{
			DB:          db,
			Group:       groups,
			ImEx:        imex.NewService(),
			Enumeration: enums,
		}))
		var got schematic.Schematic
		Expect(svc.NewRetrieve().
			Where(schematic.MatchKeys(sc.Key)).
			Entry(&got).
			Exec(ctx, db)).To(Succeed())
		sel, ok := got.Nodes[0].Config.Variant.(schematic.SelectNodeConfig)
		Expect(ok).To(BeTrue())
		Expect(sel.Enumeration).ToNot(BeNil())
		var states enumeration.Enumeration
		Expect(enums.NewRetrieve().
			Where(enumeration.MatchKeys(*sel.Enumeration)).
			Entry(&states).
			Exec(ctx, db)).To(Succeed())
		Expect(states.Name).To(Equal("Select: Panel"))
		Expect(states.Entries).To(Equal([]enumeration.Entry{
			{Key: "on", Name: "On", Value: 1},
		}))
	})
})
