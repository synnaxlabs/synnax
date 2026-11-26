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
	"strconv"
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
	symbolv3 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v3"
	v4 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v4"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/versions"
	v11 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v11"
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

//go:embed testdata/released/v0.61.4/*.hex testdata/released/v0.62.0/*.hex
var released embed.FS

// openStore returns a DB on kvDB that holds the schematic migrations through
// schematicLast and the symbol migrations through symbolLast as applied. An empty key
// applies none.
func openStore(ctx SpecContext, kvDB kv.DB, schematicLast, symbolLast string) *gorp.DB {
	GinkgoHelper()
	db := DeferClose(gorp.Wrap(kvDB))
	StartRun(db.Run)
	apply := func(namespace string, chain []gorp.Migration, last string) {
		GinkgoHelper()
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
	return decodeRecord[T]("v0.62.0", name)
}

// decodeRecord decodes the record that the Core of release stored in the named file.
func decodeRecord[T any](release, name string) T {
	GinkgoHelper()
	raw := MustSucceed(released.ReadFile("testdata/released/" + release + "/" + name))
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

	It("Should keep the value 1 looks of a static symbol that a custom_actuator node "+
		"draws, and drop them from one that no node draws", func(ctx SpecContext) {
		db := openShipped(ctx)
		red, green := color.Color{R: 255, A: 255}, color.Color{G: 255, A: 255}
		black, white := color.Color{A: 255}, color.Color{R: 255, G: 255, B: 255, A: 255}
		region := func(fill, stroke color.Color) []v4.Region {
			return []v4.Region{{
				Key:         "b",
				Name:        "b",
				Selectors:   []string{"#b"},
				FillColor:   &fill,
				StrokeColor: &stroke,
			}}
		}
		spec := v4.Spec{
			SVG:     `<svg><rect id="b"/></svg>`,
			Variant: "static",
			Scale:   1,
			States: []v4.State{
				{Key: "off", Name: "Off", Regions: region(red, black)},
				{Key: "on", Name: "On", Regions: region(green, white)},
			},
		}
		drawn := v4.Symbol{Key: uuid.New(), Name: "Drawn", Data: spec}
		plain := v4.Symbol{Key: uuid.New(), Name: "Plain", Data: spec}
		Expect(gorp.NewCreate[v4.Key, v4.Symbol]().
			Entries(&[]v4.Symbol{drawn, plain}).
			Exec(ctx, db)).To(Succeed())
		sc := v13.Schematic{
			Key:  uuid.New(),
			Name: "Schematic",
			Nodes: []v13.Node{{Key: "a", Config: v13.NodeConfig{
				Variant: v13.CustomActuatorNodeConfig{SpecKey: drawn.Key.String()},
			}}},
		}
		Expect(gorp.NewCreate[v13.Key, v13.Schematic]().Entry(&sc).Exec(ctx, db)).
			To(Succeed())
		openService(ctx, db)
		looks := func(key symbol.Key) []symbol.Look {
			GinkgoHelper()
			var sym symbol.Symbol
			Expect(gorp.NewRetrieve[symbol.Key, symbol.Symbol]().
				Where(gorp.MatchKeys[symbol.Key, symbol.Symbol](key)).
				Entry(&sym).
				Exec(ctx, db)).To(Succeed())
			return sym.Data.Looks
		}
		when := func(values ...float64) *symbol.Condition {
			return &symbol.Condition{
				Variant: symbol.EqualsCondition{State: "active", Values: values},
			}
		}
		fill := func(c color.Color) *symbol.Paint {
			return &symbol.Paint{Variant: symbol.ColorPaint{
				Value: servicecolor.Ref{Variant: servicecolor.FixedRef{Value: c}},
			}}
		}
		entry := &symbol.Paint{Variant: symbol.EntryPaint{State: "active"}}
		Expect(looks(drawn.Key)).To(Equal([]symbol.Look{
			{Key: "entry-b-stroke", Part: new("b"), When: when(0, 1), Stroke: entry},
			{Key: "look-1", Part: new("b"), When: when(0), Fill: fill(red)},
			{Key: "look-2", Part: new("b"), When: when(1), Fill: fill(green)},
		}))
		Expect(looks(plain.Key)).To(Equal([]symbol.Look{
			{Key: "entry-b-stroke", Part: new("b"), Stroke: entry},
			{Key: "look-1", Part: new("b"), Fill: fill(red)},
		}))
	})

	Describe("A symbol with one state that a custom_actuator node draws", func() {
		var db *gorp.DB
		BeforeEach(func(ctx SpecContext) { db = openShipped(ctx) })
		// store stores a static v4 symbol named name with the one state Base.
		store := func(ctx SpecContext, name string) v4.Symbol {
			GinkgoHelper()
			blue := color.Color{R: 40, G: 80, B: 220, A: 255}
			sym := v4.Symbol{Key: uuid.New(), Name: name, Data: v4.Spec{
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
			Expect(gorp.NewCreate[v4.Key, v4.Symbol]().Entry(&sym).Exec(ctx, db)).
				To(Succeed())
			return sym
		}
		// upgrade stores a schematic with a custom_actuator node that draws drawn, then
		// opens the service.
		upgrade := func(ctx SpecContext, drawn v4.Symbol) {
			GinkgoHelper()
			sc := v13.Schematic{
				Key:  uuid.New(),
				Name: "Schematic",
				Nodes: []v13.Node{{Key: "a", Config: v13.NodeConfig{
					Variant: v13.CustomActuatorNodeConfig{SpecKey: drawn.Key.String()},
				}}},
			}
			Expect(gorp.NewCreate[v13.Key, v13.Schematic]().Entry(&sc).Exec(ctx, db)).
				To(Succeed())
			openService(ctx, db)
		}
		retrieve := func(ctx SpecContext, key symbol.Key) symbol.Symbol {
			GinkgoHelper()
			var sym symbol.Symbol
			Expect(gorp.NewRetrieve[symbol.Key, symbol.Symbol]().
				Where(gorp.MatchKeys[symbol.Key, symbol.Symbol](key)).
				Entry(&sym).
				Exec(ctx, db)).To(Succeed())
			return sym
		}
		enumerationOf := func(
			ctx SpecContext,
			sym symbol.Symbol,
		) enumeration.Enumeration {
			GinkgoHelper()
			state := as[symbol.EnumerationState](sym.Data.States[0].Variant)
			var e enumeration.Enumeration
			Expect(gorp.NewRetrieve[enumeration.Key, enumeration.Enumeration]().
				Where(gorp.MatchKeys[enumeration.Key, enumeration.Enumeration](
					*state.Enumeration,
				)).
				Entry(&e).
				Exec(ctx, db)).To(Succeed())
			return e
		}
		base := enumeration.Entry{Key: "base", Value: 0, Name: "Base"}
		on := enumeration.Entry{Key: "on", Value: 1, Name: "On"}

		It("Should make it an actuator that draws value 1 as value 0, with the entry "+
			"On in its own enumeration", func(ctx SpecContext) {
			tank := store(ctx, "Tank")
			upgrade(ctx, tank)
			sym := retrieve(ctx, tank.Key)
			Expect(sym.Data.Variant).To(Equal("actuator"))
			Expect(sym.Data.Looks).To(HaveLen(1))
			Expect(sym.Data.Looks[0].When).To(Equal(&symbol.Condition{
				Variant: symbol.EqualsCondition{
					State:  "active",
					Values: []float64{0, 1},
				},
			}))
			Expect(
				enumerationOf(ctx, sym).Entries,
			).To(Equal([]enumeration.Entry{base, on}))
			var named []enumeration.Enumeration
			Expect(gorp.NewRetrieve[enumeration.Key, enumeration.Enumeration]().
				Where(gorp.Match(func(
					_ gorp.Context,
					e *enumeration.Enumeration,
				) (bool, error) {
					return e.Name == "Tank", nil
				})).
				Entries(&named).
				Exec(ctx, db)).To(Succeed())
			Expect(named).To(HaveLen(1))
		})

		It("Should give it a new enumeration with the entry On when another symbol "+
			"links its enumeration", func(ctx SpecContext) {
			tank, twin := store(ctx, "Tank"), store(ctx, "Twin")
			upgrade(ctx, tank)
			drawn, other := retrieve(ctx, tank.Key), retrieve(ctx, twin.Key)
			Expect(enumerationOf(ctx, drawn).Entries).
				To(Equal([]enumeration.Entry{base, on}))
			Expect(
				enumerationOf(ctx, other).Entries,
			).To(Equal([]enumeration.Entry{base}))
		})
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

var _ = Describe("A store from core/v0.61.4", func() {
	It(
		"Should move the overrides of each custom node to records",
		func(ctx SpecContext) {
			db := openStore(
				ctx,
				memkv.New(),
				v11.OwnersMigration.Key(),
				symbolv3.MembersMigration.Key(),
			)
			symbols := []symbolv3.Symbol{
				decodeRecord[symbolv3.Symbol]("v0.61.4", "symbol_vent_valve.hex"),
				decodeRecord[symbolv3.Symbol]("v0.61.4", "symbol_tank.hex"),
				decodeRecord[symbolv3.Symbol]("v0.61.4", "symbol_plain.hex"),
			}
			vent, tank, plain := symbols[0], symbols[1], symbols[2]
			Expect(gorp.NewCreate[symbolv3.Key, symbolv3.Symbol]().
				Entries(&symbols).
				Exec(ctx, db)).To(Succeed())
			stored := decodeRecord[v11.Schematic]("v0.61.4", "custom_nodes.hex")
			Expect(
				gorp.NewCreate[v11.Key, v11.Schematic]().Entry(&stored).Exec(ctx, db),
			).
				To(Succeed())
			openService(ctx, db)
			nodes := map[string]schematic.SymbolNodeConfig{}
			for _, n := range retrieveSchematic(ctx, db, stored.Key).Nodes {
				cfg, ok := n.Config.Variant.(schematic.SymbolNodeConfig)
				Expect(ok).To(BeTrue())
				nodes[cfg.Label.Label] = cfg
			}
			fixedRef := func(c color.Color) *servicecolor.Ref {
				return &servicecolor.Ref{Variant: servicecolor.FixedRef{Value: c}}
			}
			colors := func(key *enumeration.Key) []*servicecolor.Ref {
				GinkgoHelper()
				Expect(key).ToNot(BeNil())
				var e enumeration.Enumeration
				Expect(gorp.NewRetrieve[enumeration.Key, enumeration.Enumeration]().
					Where(gorp.MatchKeys[enumeration.Key, enumeration.Enumeration](*key)).
					Entry(&e).
					Exec(ctx, db)).To(Succeed())
				return lo.Map(
					e.Entries,
					func(e enumeration.Entry, _ int) *servicecolor.Ref {
						return e.Color
					},
				)
			}
			retrieve := func(key symbol.Key) symbol.Symbol {
				GinkgoHelper()
				var sym symbol.Symbol
				Expect(gorp.NewRetrieve[symbol.Key, symbol.Symbol]().
					Where(gorp.MatchKeys[symbol.Key, symbol.Symbol](key)).
					Entry(&sym).
					Exec(ctx, db)).To(Succeed())
				return sym
			}
			var (
				red   = color.Color{R: 220, G: 40, B: 40, A: 255}
				blue  = color.Color{R: 40, G: 80, B: 220, A: 128}
				green = color.Color{R: 40, G: 200, B: 90, A: 255}
			)
			redVent := nodes["Red vent"]
			Expect(redVent.Spec.Variant).To(Equal(symbol.CustomRef{Key: vent.Key}))
			Expect(colors(redVent.States[0].Enumeration)).
				To(Equal([]*servicecolor.Ref{fixedRef(blue), fixedRef(red)}))
			state, ok := retrieve(vent.Key).Data.States[0].Variant.(symbol.EnumerationState)
			Expect(ok).To(BeTrue())
			Expect(colors(state.Enumeration)).
				To(Equal([]*servicecolor.Ref{fixedRef(blue), fixedRef(green)}))
			white, ok := nodes["White vent"].Spec.Variant.(symbol.CustomRef)
			Expect(ok).To(BeTrue())
			Expect(white.Key).ToNot(Equal(vent.Key))
			copied := retrieve(white.Key)
			Expect(copied.Name).To(Equal("Vent valve (Custom nodes)"))
			Expect(copied.Data.Commands).To(HaveLen(2))
			Expect(nodes["Plain actuator"].States).To(HaveLen(1))
			Expect(retrieve(plain.Key).Data.Commands).To(HaveLen(2))
			Expect(nodes["Missing actuator"].Spec.Variant).
				To(BeAssignableToTypeOf(symbol.MissingRef{}))
			tankCopy, ok := nodes["Red tank"].Spec.Variant.(symbol.CustomRef)
			Expect(ok).To(BeTrue())
			Expect(tankCopy.Key).ToNot(Equal(tank.Key))
			paint := &symbol.Paint{Variant: symbol.ColorPaint{Value: *fixedRef(red)}}
			Expect(retrieve(tankCopy.Key).Data.Looks[0].Fill).To(Equal(paint))
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
		var symbols []schematic.SymbolNodeConfig
		nodes := map[string]schematic.SymbolNodeConfig{}
		for _, n := range sc.Nodes {
			cfg, ok := n.Config.Variant.(schematic.SymbolNodeConfig)
			if !ok {
				continue
			}
			if ref, ok := cfg.Spec.Variant.(symbol.BuiltinRef); ok {
				Expect(names.Contains(ref.Name)).To(BeTrue(), ref.Name)
			}
			symbols = append(symbols, cfg)
			nodes[cfg.Label.Label] = cfg
		}
		Expect(symbols).To(HaveLen(len(stored.Nodes)))
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
		light := symbol.RefVariant(symbol.BuiltinRef{Name: "light"})
		unlabeled := lo.Filter(
			symbols,
			func(cfg schematic.SymbolNodeConfig, _ int) bool {
				return cfg.Label.Label == "" && cfg.Spec.Variant == light
			},
		)
		Expect(unlabeled).To(HaveLen(1))
		onColor := func(key *enumeration.Key) (string, *servicecolor.Ref) {
			GinkgoHelper()
			Expect(key).ToNot(BeNil())
			var e enumeration.Enumeration
			Expect(gorp.NewRetrieve[enumeration.Key, enumeration.Enumeration]().
				Where(gorp.MatchKeys[enumeration.Key, enumeration.Enumeration](*key)).
				Entry(&e).
				Exec(ctx, db)).To(Succeed())
			Expect(e.Entries).To(HaveLen(2))
			return e.Name, e.Entries[1].Color
		}
		fixedRef := func(c color.Color) *servicecolor.Ref {
			return &servicecolor.Ref{Variant: servicecolor.FixedRef{Value: c}}
		}
		name, lit := onColor(nodes["On color"].States[0].Enumeration)
		Expect(name).To(Equal("Light: On color"))
		Expect(lit).To(Equal(fixedRef(red)))
		Expect(nodes["Same on color"].States[0].Enumeration).
			To(Equal(nodes["On color"].States[0].Enumeration))
		name, lit = onColor(unlabeled[0].States[0].Enumeration)
		Expect(name).To(Equal("Light: Equipment"))
		Expect(lit).To(Equal(fixedRef(color.Color{R: 40, G: 80, B: 220, A: 255})))
		Expect(nodes["light"].States[0].Enumeration).To(BeNil())
		redVent := nodes["Red vent"]
		Expect(redVent.Spec.Variant).To(Equal(symbol.CustomRef{Key: vent.Key}))
		name, lit = onColor(redVent.States[0].Enumeration)
		Expect(name).To(Equal("Vent valve: Red vent"))
		Expect(lit).To(Equal(fixedRef(red)))
		Expect(nodes["Red vent twin"].States[0].Enumeration).
			To(Equal(redVent.States[0].Enumeration))
		Expect(nodes["Vent equal overrides"].Spec.Variant).
			To(Equal(symbol.CustomRef{Key: vent.Key}))
		Expect(nodes["Vent equal overrides"].States[0].Enumeration).To(BeNil())
		Expect(nodes["Missing actuator"].Spec.Variant).
			To(Equal(symbol.MissingRef{Key: "8c10471e-6e57-4560-9f38-8d68d58ee6d5"}))
		ref, ok := nodes["Red tank custom"].Spec.Variant.(symbol.CustomRef)
		Expect(ok).To(BeTrue())
		var tankCopy symbol.Symbol
		Expect(gorp.NewRetrieve[symbol.Key, symbol.Symbol]().
			Where(gorp.MatchKeys[symbol.Key, symbol.Symbol](ref.Key)).
			Entry(&tankCopy).
			Exec(ctx, db)).To(Succeed())
		Expect(tankCopy.Name).To(Equal("Tank (Equipment)"))
		Expect(tankCopy.Group).To(Equal(tank.Group))
		paint := &symbol.Paint{Variant: symbol.ColorPaint{Value: *fixedRef(red)}}
		Expect(tankCopy.Data.Looks).To(HaveLen(1))
		Expect(tankCopy.Data.Looks[0].Fill).To(Equal(paint))
		Expect(tankCopy.Data.Looks[0].Stroke).To(Equal(paint))
	})
})

var _ = Describe("Stored v13 state overrides", func() {
	var (
		red    = color.Color{R: 220, G: 40, B: 40, A: 255}
		yellow = color.Color{R: 220, G: 200, B: 40, A: 255}
		white  = color.Color{R: 255, G: 255, B: 255, A: 255}
		// ventActive is the active color of the stored vent valve.
		ventActive = color.Color{R: 40, G: 200, B: 90, A: 255}
	)
	// paint returns the states of the stored vent valve with the body of state i in
	// strokes[i] and fills[i].
	paint := func(vent v4.Symbol, strokes, fills []color.Color) []v4.State {
		states := slices.Clone(vent.Data.States)
		for i := range states {
			states[i].Regions = slices.Clone(states[i].Regions)
			states[i].Regions[0].StrokeColor = new(strokes[i])
			states[i].Regions[0].FillColor = new(fills[i])
		}
		return states
	}
	// upgrade stores a node that draws the vent valve for each set of overrides that
	// edit returns, opens the service, and returns the vent and the upgraded nodes.
	upgrade := func(
		ctx SpecContext,
		edit func(vent v4.Symbol) [][]v4.State,
	) (*gorp.DB, v4.Symbol, []schematic.SymbolNodeConfig) {
		GinkgoHelper()
		db := openShipped(ctx)
		vent := decodeReleased[v4.Symbol]("symbol_vent_valve.hex")
		Expect(gorp.NewCreate[v4.Key, v4.Symbol]().Entry(&vent).Exec(ctx, db)).
			To(Succeed())
		stored := v13.Schematic{Key: uuid.New(), Name: "Panel"}
		for i, states := range edit(vent) {
			cfg := v13.CustomActuatorNodeConfig{SpecKey: vent.Key.String()}
			cfg.Label.Label, cfg.StateOverrides = "Vent", states
			stored.Nodes = append(stored.Nodes, v13.Node{
				Key:    strconv.Itoa(i),
				Config: v13.NodeConfig{Variant: cfg},
			})
		}
		Expect(gorp.NewCreate[v13.Key, v13.Schematic]().Entry(&stored).Exec(ctx, db)).
			To(Succeed())
		openService(ctx, db)
		var sc schematic.Schematic
		Expect(gorp.NewRetrieve[schematic.Key, schematic.Schematic]().
			Where(gorp.MatchKeys[schematic.Key, schematic.Schematic](stored.Key)).
			Entry(&sc).
			Exec(ctx, db)).To(Succeed())
		return db, vent, lo.Map(
			sc.Nodes,
			func(n schematic.Node, _ int) schematic.SymbolNodeConfig {
				cfg, ok := n.Config.Variant.(schematic.SymbolNodeConfig)
				Expect(ok).To(BeTrue())
				return cfg
			},
		)
	}

	DescribeTable("Should draw the overrides of a node as 0.62 drew them", func(
		ctx SpecContext,
		edit func(vent v4.Symbol) []v4.State,
		copied bool,
		entries []color.Color,
	) {
		db, vent, nodes := upgrade(ctx, func(vent v4.Symbol) [][]v4.State {
			return [][]v4.State{edit(vent)}
		})
		n := nodes[0]
		if copied {
			Expect(n.Spec.Variant).ToNot(Equal(symbol.CustomRef{Key: vent.Key}))
		} else {
			Expect(n.Spec.Variant).To(Equal(symbol.CustomRef{Key: vent.Key}))
		}
		if entries == nil {
			Expect(n.States[0].Enumeration).To(BeNil())
			return
		}
		Expect(n.States[0].Enumeration).ToNot(BeNil())
		var e enumeration.Enumeration
		Expect(gorp.NewRetrieve[enumeration.Key, enumeration.Enumeration]().
			Where(gorp.MatchKeys[enumeration.Key, enumeration.Enumeration](
				*n.States[0].Enumeration,
			)).
			Entry(&e).
			Exec(ctx, db)).To(Succeed())
		Expect(e.Name).To(Equal("Vent valve: Vent"))
		Expect(lo.Map(e.Entries, func(e enumeration.Entry, _ int) servicecolor.Ref {
			return *e.Color
		})).To(Equal(lo.Map(entries, func(c color.Color, _ int) servicecolor.Ref {
			return servicecolor.Ref{Variant: servicecolor.FixedRef{Value: c}}
		})))
	},
		Entry("overrides equal to the symbol", func(vent v4.Symbol) []v4.State {
			return vent.Data.States
		}, false, nil),
		Entry("new entry colors", func(vent v4.Symbol) []v4.State {
			c := []color.Color{red, yellow}
			return paint(vent, c, c)
		}, false, []color.Color{red, yellow}),
		Entry("an override of the first state alone", func(vent v4.Symbol) []v4.State {
			c := []color.Color{red, yellow}
			return paint(vent, c, c)[:1]
		}, false, []color.Color{red, ventActive}),
		Entry("more overrides than states", func(vent v4.Symbol) []v4.State {
			c := []color.Color{red, yellow}
			return append(paint(vent, c, c), v4.State{Key: "extra"})
		}, false, []color.Color{red, yellow}),
		Entry("a fill that no entry carries", func(vent v4.Symbol) []v4.State {
			return paint(vent, []color.Color{red, yellow}, []color.Color{white, white})
		}, true, nil),
	)

	It(
		"Should give nodes with the same symbol and overrides one copy with the "+
			"commands of the symbol",
		func(ctx SpecContext) {
			db, vent, nodes := upgrade(ctx, func(vent v4.Symbol) [][]v4.State {
				states := paint(
					vent,
					[]color.Color{red, yellow},
					[]color.Color{white, white},
				)
				return [][]v4.State{states, states}
			})
			Expect(nodes[1].Spec).To(Equal(nodes[0].Spec))
			ref, ok := nodes[0].Spec.Variant.(symbol.CustomRef)
			Expect(ok).To(BeTrue())
			Expect(ref.Key).ToNot(Equal(vent.Key))
			var copied symbol.Symbol
			Expect(gorp.NewRetrieve[symbol.Key, symbol.Symbol]().
				Where(gorp.MatchKeys[symbol.Key, symbol.Symbol](ref.Key)).
				Entry(&copied).
				Exec(ctx, db)).To(Succeed())
			Expect(copied.Name).To(Equal("Vent valve (Panel)"))
			Expect(copied.Group).To(Equal(vent.Group))
			Expect(copied.Data.Commands).To(HaveLen(2))
		},
	)
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
