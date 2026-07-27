// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v1"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Migration", func() {
	// seedV0 writes a symbol in the untyped v0 storage shape, exactly as the
	// pre-SY-4504 server persisted it.
	seedV0 := func(ctx SpecContext, db *gorp.DB, s v0.Symbol) v0.Symbol {
		t := MustOpen(gorp.OpenTable(
			ctx, gorp.TableConfig[uuid.UUID, v0.Symbol]{DB: db},
		))
		Expect(t.NewCreate().Entry(&s).Exec(ctx, db)).To(Succeed())
		return s
	}

	// openMigrated opens the v1 symbol table with the migration chain wired in, driving
	// the v0 -> typed lift end-to-end through gorp.
	openMigrated := func(ctx SpecContext, db *gorp.DB) *gorp.Table[v1.Key, v1.Symbol] {
		return MustOpen(gorp.OpenTable(
			ctx, gorp.TableConfig[v1.Key, v1.Symbol]{
				DB:         db,
				Migrations: []migrate.Migration{v1.Migration},
			},
		))
	}

	retrieve := func(ctx SpecContext, db *gorp.DB, t *gorp.Table[v1.Key, v1.Symbol], key v1.Key) v1.Symbol {
		var got v1.Symbol
		Expect(t.NewRetrieve().
			Where(gorp.MatchKeys[v1.Key, v1.Symbol](key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		return got
	}

	It("Should lift an untyped v0 symbol into the typed Symbol on retrieve", func(ctx SpecContext) {
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

		got := retrieve(ctx, db, openMigrated(ctx, db), seed.Key)

		Expect(got.Key).To(Equal(seed.Key))
		Expect(got.Name).To(Equal("pump"))
		Expect(got.Version).To(Equal(uint32(1)))
		Expect(got.Data.SVG).To(Equal("<svg>x</svg>"))
		Expect(got.Data.Variant).To(Equal("valve"))
		Expect(got.Data.Scale).To(Equal(2.0))
		Expect(got.Data.ScaleStroke).To(BeTrue())
		Expect(got.Data.States).To(HaveLen(1))
		Expect(got.Data.States[0].Key).To(Equal("on"))
		Expect(got.Data.States[0].Regions).To(HaveLen(1))
		region := got.Data.States[0].Regions[0]
		Expect(region.Key).To(Equal("body"))
		Expect(region.StrokeColor).To(HaveValue(Equal("#ffffff")))
		Expect(region.Selectors).To(ConsistOf(".body"))
	})

	It("Should default the scale and stamp the version when unset in v0 data", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		seed := seedV0(ctx, db, v0.Symbol{
			Key:  uuid.New(),
			Name: "bare",
			Data: map[string]any{"svg": "<svg/>", "variant": "sensor"},
		})

		got := retrieve(ctx, db, openMigrated(ctx, db), seed.Key)

		Expect(got.Version).To(Equal(uint32(1)))
		Expect(got.Data.Scale).To(Equal(1.0))
	})
})
