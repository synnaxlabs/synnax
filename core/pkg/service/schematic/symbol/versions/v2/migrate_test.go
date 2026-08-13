// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	legacyv1 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/legacy/v1"
	v0 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v0"
	v2 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v2"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/spatial"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Migration", func() {
	// seedV0 writes a symbol in the untyped v0 storage shape through a plain
	// MessagePack codec, exactly as the pre-SY-4504 server persisted it.
	seedV0 := func(ctx SpecContext, db *gorp.DB, s v0.Symbol) v0.Symbol {
		GinkgoHelper()
		legacyDB := gorp.Wrap(db.DB, gorp.WithCodec(msgpack.Codec))
		t := MustOpen(gorp.OpenTable(
			ctx, gorp.TableConfig[uuid.UUID, v0.Symbol]{DB: legacyDB},
		))
		Expect(t.NewCreate().Entry(&s).Exec(ctx, legacyDB)).To(Succeed())
		return s
	}

	// retrieveMigrated opens the v2 symbol table with the migration wired in, driving
	// the v0 -> typed lift end-to-end through gorp.
	retrieveMigrated := func(ctx SpecContext, db *gorp.DB, key v2.Key) v2.Symbol {
		GinkgoHelper()
		t := MustOpen(gorp.OpenTable(
			ctx, gorp.TableConfig[v2.Key, v2.Symbol]{
				DB:         db,
				Migrations: []migrate.Migration{v2.Migration},
			},
		))
		var got v2.Symbol
		Expect(t.NewRetrieve().
			Where(gorp.MatchKeys[v2.Key, v2.Symbol](key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		return got
	}

	It(
		"Should lift an untyped v0 symbol into the typed Symbol on retrieve",
		func(ctx SpecContext) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			seed := seedV0(ctx, db, v0.Symbol{
				Key:  uuid.New(),
				Name: "pump",
				Data: map[string]any{
					"svg":          "<svg>x</svg>",
					"variant":      "valve",
					"scale":        2.0,
					"scale_stroke": true,
					"states": []any{
						map[string]any{"key": "on", "name": "On", "regions": []any{
							map[string]any{
								"key":          "body",
								"name":         "Body",
								"stroke_color": "#ffffff",
								"selectors":    []any{".body"},
							},
						}},
					},
				},
			})

			got := retrieveMigrated(ctx, db, seed.Key)

			Expect(got.Key).To(Equal(seed.Key))
			Expect(got.Name).To(Equal("pump"))
			Expect(got.Data.SVG).To(Equal("<svg>x</svg>"))
			Expect(got.Data.Variant).To(Equal("valve"))
			Expect(got.Data.Scale).To(Equal(2.0))
			Expect(got.Data.ScaleStroke).To(BeTrue())
			Expect(got.Data.States).To(HaveLen(1))
			Expect(got.Data.States[0].Key).To(Equal("on"))
			Expect(got.Data.States[0].Regions).To(HaveLen(1))
			region := got.Data.States[0].Regions[0]
			Expect(region.Key).To(Equal("body"))
			Expect(
				region.StrokeColor,
			).To(HaveValue(Equal(color.MustFromHex("#ffffff"))))
			Expect(region.Selectors).To(ConsistOf(".body"))
		},
	)

	It("Should default the scale when unset in v0 data", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		seed := seedV0(ctx, db, v0.Symbol{
			Key:  uuid.New(),
			Name: "bare",
			Data: map[string]any{"svg": "<svg/>", "variant": "sensor"},
		})

		Expect(retrieveMigrated(ctx, db, seed.Key).Data.Scale).To(Equal(1.0))
	})
})

var _ = Describe("SpecFromConsole", func() {
	It("Should lift the Console specification into the current Spec", func() {
		out := v2.SpecFromConsole(legacyv1.Spec{
			SVG:             "<svg/>",
			Variant:         "valve",
			Scale:           2,
			ScaleStroke:     true,
			PreviewViewport: &spatial.Viewport{Zoom: 3},
			States: []legacyv1.State{{
				Key:  "base",
				Name: "Base",
				Regions: []legacyv1.Region{{
					Key:         "r1",
					Name:        "Body",
					Selectors:   []string{"#body"},
					StrokeColor: new(color.MustFromHex("#ff0000")),
				}},
			}},
			Handles: []legacyv1.Handle{{
				Key:         "h1",
				Position:    spatial.XY{X: 1, Y: 2},
				Orientation: spatial.OuterLocationLeft,
			}},
		})

		Expect(out.SVG).To(Equal("<svg/>"))
		Expect(out.Variant).To(Equal("valve"))
		Expect(out.Scale).To(Equal(2.0))
		Expect(out.ScaleStroke).To(BeTrue())
		Expect(out.PreviewViewport).To(HaveValue(Equal(spatial.Viewport{Zoom: 3})))
		Expect(out.States).To(Equal([]v2.State{{
			Key:  "base",
			Name: "Base",
			Regions: []v2.Region{{
				Key:         "r1",
				Name:        "Body",
				Selectors:   []string{"#body"},
				StrokeColor: new(color.MustFromHex("#ff0000")),
			}},
		}}))
		Expect(out.Handles).To(Equal([]v2.Handle{{
			Key:         "h1",
			Position:    spatial.XY{X: 1, Y: 2},
			Orientation: spatial.OuterLocationLeft,
		}}))
	})

	It("Should produce an empty Spec from an empty Console body", func() {
		out := v2.SpecFromConsole(legacyv1.Spec{})

		Expect(out.SVG).To(BeEmpty())
		Expect(out.States).To(BeEmpty())
		Expect(out.Handles).To(BeEmpty())
		Expect(out.PreviewViewport).To(BeNil())
	})
})
