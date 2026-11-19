// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/enumeration"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	. "github.com/synnaxlabs/synnax/pkg/service/imex/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions"
	"github.com/synnaxlabs/x/defaults"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

// createImportGroup creates a symbol group in tx and returns its key.
func createImportGroup(ctx context.Context, tx gorp.Tx) *group.Key {
	GinkgoHelper()
	g := group.Group{Name: "Imported", Scope: resource.TypeSchematicSymbol}
	Expect(groupSvc.NewWriter(tx).Create(ctx, &g)).To(Succeed())
	return &g.Key
}

var _ = Describe("ImEx", func() {
	DescribeTable("Match",
		func(body map[string]any, expected bool) {
			Expect(svc.Match(body)).To(Equal(expected))
		},
		Entry("data with svg", map[string]any{"data": map[string]any{"svg": ""}}, true),
		Entry("data without svg", map[string]any{"data": map[string]any{}}, false),
		Entry("top-level svg", map[string]any{"svg": ""}, false),
		Entry("empty body", map[string]any{}, false),
	)

	Describe("Export", func() {
		It("Should export a symbol as a versioned envelope", func(ctx SpecContext) {
			sym := defaults.Fill(symbol.Symbol{
				Name: "exported",
				Data: symbol.Spec{Drawing: svg("<svg/>"), Variant: "valve"},
			})
			// Export reads committed data, so the symbol must be deleted to keep the
			// shared DB's counts intact.
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &sym)
			})).To(Succeed())
			DeferCleanup(func(ctx SpecContext) {
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Delete(ctx, sym.Key)
				})).To(Succeed())
			})
			env := MustSucceed(svc.Export(ctx, db, symbol.ResourceID(sym.Key)))
			Expect(env.Version).To(Equal(versions.Latest))
			Expect(env.Type).To(Equal("schematic_symbol"))
			Expect(env.Name).To(Equal("exported"))

			wire := WireRoundTrip(env)
			decoded := MustSucceed(wire.Decode[symbol.Symbol]())
			Expect(decoded.Name).To(Equal("exported"))
			Expect(decoded.Data.Drawing).To(Equal(svg("<svg/>")))
		})

		It("Should return not found for a missing key", func(ctx SpecContext) {
			id := resource.ID{
				Type: resource.TypeSchematicSymbol,
				Key:  uuid.New().String(),
			}
			Expect(svc.Export(ctx, db, id)).Error().To(MatchError(query.ErrNotFound))
		})

		It("Should error on an invalid UUID key", func(ctx SpecContext) {
			id := resource.ID{
				Type: resource.TypeSchematicSymbol,
				Key:  "not-a-uuid",
			}
			Expect(
				svc.Export(ctx, db, id),
			).Error().
				To(MatchError(ContainSubstring("uuid")))
		})
	})

	Describe("Import", func() {
		// Imports run on the per-spec tx so created rows roll back and the shared DB's
		// symbol counts stay intact for the other specs.
		var tx gorp.Tx
		BeforeEach(func() { tx = DeferClose(db.OpenTx()) })
		importAndRetrieve := func(
			ctx SpecContext,
			path string,
			g *group.Key,
		) symbol.Symbol {
			GinkgoHelper()
			id := MustSucceed(imexSvc.Import(
				ctx,
				tx,
				LoadEnvelope(path),
				imex.ImportOptions{Group: g},
			))
			Expect(id.Type).To(Equal(resource.TypeSchematicSymbol))
			key := MustSucceed(uuid.Parse(id.Key))
			var res symbol.Symbol
			Expect(svc.NewRetrieve().
				Where(symbol.MatchKeys(key)).
				Entry(&res).
				Exec(ctx, tx)).To(Succeed())
			return res
		}

		It(
			"Should import a server export carrying snake_case keys",
			func(ctx SpecContext) {
				res := importAndRetrieve(
					ctx,
					"versions/testdata/import_v2.json",
					createImportGroup(ctx, tx),
				)
				Expect(res.Name).To(Equal("Server Symbol"))
				Expect(res.Data.Drawing).To(Equal(svg("<svg><rect/></svg>")))
				Expect(res.Data.Variant).To(Equal("valve"))
				Expect(res.Data.StrokeScaled).To(BeTrue())
				Expect(res.Data.States).To(HaveLen(1))
				state, ok := res.Data.States[0].Variant.(symbol.EnumerationState)
				Expect(ok).To(BeTrue())
				Expect(state.Enumeration).To(BeNil())
				Expect(res.Data.Looks).To(HaveLen(1))
				Expect(
					res.Data.Looks[0].Stroke,
				).To(HaveValue(Equal(colorPaint("#ff0000"))))
				Expect(
					res.Data.Looks[0].Fill,
				).To(HaveValue(Equal(colorPaint("#00ff00"))))
				Expect(res.Data.Handles).To(HaveLen(1))
				Expect(res.Data.PreviewViewport).ToNot(BeNil())
				Expect(res.Data.PreviewViewport.Zoom).To(Equal(3.0))
			},
		)

		It(
			"Should import a Console export carrying camelCase keys",
			func(ctx SpecContext) {
				res := importAndRetrieve(
					ctx,
					"versions/testdata/import_console.json",
					createImportGroup(ctx, tx),
				)
				Expect(res.Name).To(Equal("Console Symbol"))
				Expect(res.Data.Variant).To(Equal("sensor"))
				Expect(res.Data.Scale).To(Equal(2.0))
				Expect(res.Data.StrokeScaled).To(BeTrue())
				Expect(res.Data.Looks).To(HaveLen(1))
				Expect(
					res.Data.Looks[0].Stroke,
				).To(HaveValue(Equal(colorPaint("#123456"))))
				Expect(
					res.Data.Looks[0].Fill,
				).To(HaveValue(Equal(colorPaint("#654321"))))
				Expect(res.Data.Handles).To(HaveLen(1))
				Expect(res.Data.PreviewViewport).ToNot(BeNil())
				Expect(res.Data.PreviewViewport.Zoom).To(Equal(6.0))
			},
		)

		DescribeTable(
			"Should keep the colors of an old symbol file as color looks with no enumeration",
			func(ctx SpecContext, path string, colors [][2]string) {
				countEnumerations := func() int {
					GinkgoHelper()
					var all []enumeration.Enumeration
					Expect(
						enumSvc.NewRetrieve().Entries(&all).Exec(ctx, tx),
					).To(Succeed())
					return len(all)
				}
				before := countEnumerations()
				res := importAndRetrieve(ctx, path, createImportGroup(ctx, tx))
				Expect(countEnumerations()).To(Equal(before))
				state, ok := res.Data.States[0].Variant.(symbol.EnumerationState)
				Expect(ok).To(BeTrue())
				Expect(state.Enumeration).To(BeNil())
				Expect(res.Data.Looks).To(HaveLen(len(colors)))
				for i, c := range colors {
					look := res.Data.Looks[i]
					Expect(look.When).To(HaveValue(Equal(symbol.Condition{
						Variant: symbol.EqualsCondition{
							State:  "active",
							Values: []float64{float64(i)},
						},
					})))
					Expect(look.Stroke).To(HaveValue(Equal(colorPaint(c[0]))))
					Expect(look.Fill).To(HaveValue(Equal(colorPaint(c[1]))))
				}
			},
			Entry(
				"a v2 file with one state",
				"versions/testdata/import_v2.json",
				[][2]string{{"#ff0000", "#00ff00"}},
			),
			Entry(
				"a v3 file with the states Base and Active",
				"versions/testdata/import_v3.json",
				[][2]string{{"#111111", "#222222"}, {"#333333", "#444444"}},
			),
		)

		It("Should put the imported symbol in the given group", func(ctx SpecContext) {
			g := createImportGroup(ctx, tx)
			res := importAndRetrieve(ctx, "versions/testdata/import_v2.json", g)
			Expect(res.Group).To(Equal(g))
		})

		It("Should put the imported symbol in no group", func(ctx SpecContext) {
			res := importAndRetrieve(ctx, "versions/testdata/import_v2.json", nil)
			Expect(res.Group).To(BeNil())
		})

		It("Should reject a group that does not exist", func(ctx SpecContext) {
			Expect(imexSvc.Import(
				ctx, tx,
				LoadEnvelope("versions/testdata/import_v2.json"),
				imex.ImportOptions{Group: new(uuid.New())},
			)).Error().To(MatchError(ContainSubstring("does not exist")))
		})

		It(
			"Should reject an envelope newer than the supported version",
			func(ctx SpecContext) {
				Expect(imexSvc.Import(ctx, tx,
					LoadEnvelope("versions/testdata/import_bad_version.json"),
					imex.ImportOptions{Group: createImportGroup(ctx, tx)},
				)).Error().To(SatisfyAll(
					MatchError(ContainSubstring("schematic_symbol version 99")),
					MatchError(ContainSubstring("newer than this Core supports")),
				))
			},
		)

		It(
			"Should generate a fresh key, discarding the key on the wire",
			func(ctx SpecContext) {
				id := MustSucceed(imexSvc.Import(
					ctx,
					tx,
					LoadEnvelope(
						"versions/testdata/import_v2.json",
					),
					imex.ImportOptions{Group: createImportGroup(ctx, tx)},
				))
				Expect(id.Key).ToNot(Equal("11111111-2222-3333-4444-555555555555"))
			},
		)
	})

	Describe("Enumerations", func() {
		var used enumeration.Enumeration
		BeforeEach(func(ctx SpecContext) {
			created := []enumeration.Enumeration{{
				Name:    "Valve",
				Entries: []enumeration.Entry{{Key: "off", Value: 0, Name: "Off"}},
			}}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return enumSvc.NewWriter(tx).CreateMany(ctx, created)
			})).To(Succeed())
			used = created[0]
		})
		// export returns the envelope of a symbol whose state shows used, and deletes
		// the symbol so that used can be deleted.
		export := func(ctx SpecContext) imex.Envelope {
			GinkgoHelper()
			sym := defaults.Fill(symbol.Symbol{Name: "valve", Data: symbol.Spec{
				Drawing: svg("<svg/>"),
				Variant: "static",
				States: []symbol.State{{Variant: symbol.EnumerationState{
					Key:         "active",
					Name:        "Active",
					Enumeration: &used.Key,
					Match:       symbol.Matcher{Variant: symbol.ExactMatcher{}},
				}}},
			}})
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &sym)
			})).To(Succeed())
			env := MustSucceed(svc.Export(ctx, db, symbol.ResourceID(sym.Key)))
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, sym.Key)
			})).To(Succeed())
			return env
		}
		deleteUsed := func(ctx SpecContext) {
			GinkgoHelper()
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return enumSvc.NewWriter(tx).Delete(ctx, used.Key)
			})).To(Succeed())
		}
		importState := func(
			ctx SpecContext,
			env imex.Envelope,
		) symbol.EnumerationState {
			GinkgoHelper()
			tx := DeferClose(db.OpenTx())
			id := MustSucceed(imexSvc.Import(
				ctx,
				tx,
				WireRoundTrip(env),
				imex.ImportOptions{Group: createImportGroup(ctx, tx)},
			))
			var res symbol.Symbol
			Expect(svc.NewRetrieve().
				Where(symbol.MatchKeys(MustSucceed(uuid.Parse(id.Key)))).
				Entry(&res).
				Exec(ctx, tx)).To(Succeed())
			state, ok := res.Data.States[0].Variant.(symbol.EnumerationState)
			Expect(ok).To(BeTrue())
			return state
		}

		It("Should keep a state enumeration that the Core has", func(ctx SpecContext) {
			DeferCleanup(deleteUsed)
			env := export(ctx)
			Expect(importState(ctx, env).Enumeration).To(HaveValue(Equal(used.Key)))
		})

		It(
			"Should clear a state enumeration that the Core lacks",
			func(ctx SpecContext) {
				env := export(ctx)
				deleteUsed(ctx)
				Expect(importState(ctx, env).Enumeration).To(BeNil())
			},
		)
	})

	Describe("Round trip", func() {
		It(
			"Should preserve symbol content through export then import",
			func(ctx SpecContext) {
				original := defaults.Fill(symbol.Symbol{
					Name: "round-trip",
					Data: symbol.Spec{
						Drawing:      svg("<svg/>"),
						Variant:      "valve",
						StrokeScaled: true,
					},
				})
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &original)
				})).
					To(Succeed())
				DeferCleanup(func(ctx SpecContext) {
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Delete(ctx, original.Key)
					})).To(Succeed())
				})
				env := MustSucceed(svc.Export(ctx, db, symbol.ResourceID(original.Key)))
				tx := DeferClose(db.OpenTx())
				id := MustSucceed(imexSvc.Import(
					ctx,
					tx,
					WireRoundTrip(env),
					imex.ImportOptions{Group: createImportGroup(ctx, tx)},
				))
				key := MustSucceed(uuid.Parse(id.Key))
				Expect(key).ToNot(Equal(original.Key))
				var res symbol.Symbol
				Expect(svc.NewRetrieve().
					Where(symbol.MatchKeys(key)).
					Entry(&res).
					Exec(ctx, tx)).To(Succeed())
				Expect(res.Name).To(Equal("round-trip"))
				// A required collection always serializes, so each list comes back
				// allocated and empty rather than nil.
				Expect(res.Data).To(Equal(symbol.Spec{
					Drawing:      svg("<svg/>"),
					Parts:        []symbol.Part{},
					Handles:      []symbol.Handle{},
					States:       []symbol.State{},
					Commands:     []symbol.Command{},
					Looks:        []symbol.Look{},
					Interactions: []symbol.Interaction{},
					Variant:      "valve",
					StrokeScaled: true,
					Scale:        original.Data.Scale,
				}))
			},
		)
	})
})
