// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package migrations_test

import (
	"context"
	"embed"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	v55 "github.com/synnaxlabs/synnax/pkg/service/table/migrations/v55"
	v56 "github.com/synnaxlabs/synnax/pkg/service/table/migrations/v56"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	. "github.com/synnaxlabs/x/testutil"
)

//go:embed testdata/*.json
var fixtures embed.FS

func loadFixture(name string) msgpack.EncodedJSON {
	raw := MustSucceed(fixtures.ReadFile("testdata/" + name))
	var m map[string]any
	Expect(json.Unmarshal(raw, &m)).To(Succeed())
	return msgpack.EncodedJSON(m)
}

func jsonMap(raw string) msgpack.EncodedJSON {
	var m map[string]any
	Expect(json.Unmarshal([]byte(raw), &m)).To(Succeed())
	return m
}

// assertMigrated compares got against the canonical .migrated.json file for
// fixture, or rewrites it if UPDATE_MIGRATED=1 is set. Outputs are
// canonicalized via json.MarshalIndent (which sorts map keys) so diffs are
// deterministic.
func assertMigrated(fixture string, got table.Table) {
	pretty := MustSucceed(json.MarshalIndent(got, "", "  "))
	pretty = append(pretty, '\n')
	stem := strings.TrimSuffix(fixture, ".json")
	p := filepath.Join("testdata", stem+".migrated.json")
	if os.Getenv("UPDATE_MIGRATED") == "1" {
		Expect(os.WriteFile(p, pretty, 0o644)).To(Succeed())
		return
	}
	expected := MustSucceed(os.ReadFile(p))
	Expect(pretty).To(MatchJSON(expected),
		"%s drifted from its canonical migrated form — review the diff and rerun with UPDATE_MIGRATED=1 if intentional", fixture)
}

// cfgFields returns a typed cell config's wire fields for shape assertions.
func cfgFields(cfg table.CellConfig) map[string]any {
	b := MustSucceed(json.Marshal(cfg))
	var m map[string]any
	Expect(json.Unmarshal(b, &m)).To(Succeed())
	return m
}

// migrateFromV55 composes the full migration chain from a v55 snapshot to the
// current typed Table.
func migrateFromV55(ctx context.Context, old v55.Table) (table.Table, error) {
	mid, err := v56.MigrateTable(ctx, old)
	if err != nil {
		return table.Table{}, err
	}
	return table.MigrateTable(ctx, mid)
}

var _ = Describe("MigrateTable", func() {
	// Snapshot tests against the canonical .migrated.json output for every
	// captured fixture. Run with UPDATE_MIGRATED=1 to regenerate the
	// .migrated.json files after intentional migration changes.
	Describe("canonical migrated output", func() {
		fixedKey := uuid.MustParse("00000000-0000-0000-0000-000000000001")
		DescribeTable("Should produce the canonical typed Table",
			func(ctx SpecContext, fixture string) {
				blob := loadFixture(fixture)
				snap := v55.Table{Key: fixedKey, Name: fixture, Data: blob}
				out := MustSucceed(migrateFromV55(ctx, snap))
				assertMigrated(fixture, out)
			},
			Entry("v0 empty", "v0_empty.json"),
			Entry("v0 mixed variants", "v0_mixed_variants.json"),
		)
	})

	Describe("v0 reshape semantics", func() {
		migrate := func(ctx SpecContext, body string) table.Table {
			return MustSucceed(migrateFromV55(ctx, v55.Table{
				Key: uuid.New(), Data: jsonMap(body),
			}))
		}

		It("Should flatten layout.rows[*].cells from CellLayout[] to []string", func(ctx SpecContext) {
			out := migrate(ctx, `{
				"version": "0.0.0",
				"layout": {
					"rows": [{"size": 36, "cells": [{"key": "a"}, {"key": "b"}]}],
					"columns": [{"size": 72}]
				},
				"cells": {}
			}`)
			Expect(out.Rows).To(HaveLen(1))
			Expect(out.Rows[0].Size).To(Equal(36.0))
			Expect(out.Rows[0].Cells).To(Equal([]string{"a", "b"}))
		})

		It("Should drop the per-cell selected flag", func(ctx SpecContext) {
			out := migrate(ctx, `{
				"version": "0.0.0",
				"layout": {"rows": [], "columns": []},
				"cells": {
					"a": {"key": "a", "variant": "text", "selected": true, "props": {"value": "hi"}}
				}
			}`)
			fields := cfgFields(out.Cells["a"])
			Expect(fields).To(SatisfyAll(
				HaveKeyWithValue("variant", "text"),
				HaveKeyWithValue("value", "hi"),
			))
			Expect(fields).NotTo(HaveKey("selected"))
		})

		It("Should pass through the gorp-entry fields (Key, Name)", func(ctx SpecContext) {
			key := uuid.New()
			out := MustSucceed(migrateFromV55(ctx, v55.Table{
				Key: key, Name: "trip-table", Data: jsonMap(`{"version": "0.0.0"}`),
			}))
			Expect(out.Key).To(Equal(key))
			Expect(out.Name).To(Equal("trip-table"))
		})

		DescribeTable("Should produce empty (not nil) collections for empty inputs",
			func(ctx SpecContext, data msgpack.EncodedJSON) {
				out := MustSucceed(migrateFromV55(ctx, v55.Table{
					Key: uuid.New(), Name: "empty", Data: data,
				}))
				Expect(out.Rows).NotTo(BeNil())
				Expect(out.Rows).To(BeEmpty())
				Expect(out.Columns).NotTo(BeNil())
				Expect(out.Columns).To(BeEmpty())
				Expect(out.Cells).NotTo(BeNil())
				Expect(out.Cells).To(BeEmpty())
			},
			Entry("nil blob", msgpack.EncodedJSON(nil)),
			Entry("empty object", jsonMap(`{}`)),
		)

		It("Should preserve multi-row layout ordering and per-row size", func(ctx SpecContext) {
			out := migrate(ctx, `{
				"version": "0.0.0",
				"layout": {
					"rows": [
						{"size": 36, "cells": [{"key": "a"}, {"key": "b"}]},
						{"size": 50, "cells": [{"key": "c"}, {"key": "d"}]}
					],
					"columns": [{"size": 72}, {"size": 100}]
				},
				"cells": {}
			}`)
			Expect(out.Rows).To(HaveLen(2))
			Expect(out.Rows[0].Size).To(Equal(36.0))
			Expect(out.Rows[0].Cells).To(Equal([]string{"a", "b"}))
			Expect(out.Rows[1].Size).To(Equal(50.0))
			Expect(out.Rows[1].Cells).To(Equal([]string{"c", "d"}))
			Expect(out.Columns).To(HaveLen(2))
			Expect(out.Columns[0].Size).To(Equal(72.0))
			Expect(out.Columns[1].Size).To(Equal(100.0))
		})
	})

	Describe("typed cell configs", func() {
		migrateCell := func(ctx SpecContext, cell string) table.CellConfig {
			out := MustSucceed(migrateFromV55(ctx, v55.Table{
				Key: uuid.New(), Data: jsonMap(`{
					"version": "0.0.0",
					"layout": {"rows": [], "columns": []},
					"cells": {"a": ` + cell + `}
				}`),
			}))
			Expect(out.Cells).To(HaveKey("a"))
			return out.Cells["a"]
		}

		It("Should normalize camelCase props to the snake_case wire form", func(ctx SpecContext) {
			fields := cfgFields(migrateCell(ctx, `{
				"key": "a", "variant": "text", "selected": false,
				"props": {"value": "hi", "backgroundColor": "#112233"}
			}`))
			Expect(fields).To(HaveKey("background_color"))
			Expect(fields).NotTo(HaveKey("backgroundColor"))
		})

		It("Should extract value cell telem args from the stored pipeline spec", func(ctx SpecContext) {
			fields := cfgFields(migrateCell(ctx, `{
				"key": "a", "variant": "value", "selected": false,
				"props": {
					"telem": {
						"type": "source-pipeline",
						"variant": "source",
						"valueType": "string",
						"props": {
							"segments": {
								"valueStream": {"props": {"channel": 42}},
								"rollingAverage": {"props": {"windowSize": 5}},
								"stringifier": {"props": {"precision": 3, "notation": "scientific"}}
							},
							"outlet": "stringifier"
						}
					},
					"units": "psi"
				}
			}`))
			Expect(fields).To(SatisfyAll(
				HaveKeyWithValue("variant", "value"),
				HaveKeyWithValue("channel", 42.0),
				HaveKeyWithValue("rolling_average", 5.0),
				HaveKeyWithValue("precision", 3.0),
				HaveKeyWithValue("notation", "scientific"),
				HaveKeyWithValue("units", "psi"),
			))
			Expect(fields).NotTo(HaveKey("telem"))
		})

		It("Should leave the channel at the zero sentinel for a legacy zero channel", func(ctx SpecContext) {
			fields := cfgFields(migrateCell(ctx, `{
				"key": "a", "variant": "value", "selected": false,
				"props": {
					"telem": {
						"props": {"segments": {"valueStream": {"props": {"channel": 0}}}}
					}
				}
			}`))
			Expect(fields).To(HaveKeyWithValue("channel", 0.0))
		})

		DescribeTable("Should map legacy x-location alignments onto flex alignments",
			func(ctx SpecContext, legacy, want string) {
				fields := cfgFields(migrateCell(ctx, `{
					"key": "a", "variant": "text", "selected": false,
					"props": {"value": "hi", "align": "`+legacy+`"}
				}`))
				Expect(fields).To(HaveKeyWithValue("align", want))
			},
			Entry("left", "left", "start"),
			Entry("right", "right", "end"),
			Entry("center", "center", "center"),
			Entry("start passes through", "start", "start"),
			Entry("end passes through", "end", "end"),
		)

		It("Should map named font weights onto numeric weights", func(ctx SpecContext) {
			fields := cfgFields(migrateCell(ctx, `{
				"key": "a", "variant": "text", "selected": false,
				"props": {"value": "hi", "weight": "bold"}
			}`))
			Expect(fields).To(HaveKeyWithValue("weight", 700.0))
		})

		It("Should degrade cells with an unknown variant to an empty text cell", func(ctx SpecContext) {
			fields := cfgFields(migrateCell(ctx, `{
				"key": "a", "variant": "hologram", "selected": false,
				"props": {"value": "hi"}
			}`))
			Expect(fields).To(HaveKeyWithValue("variant", "text"))
		})
	})

	Describe("malformed input", func() {
		It("Should return an error for an invalid Data shape", func(ctx SpecContext) {
			Expect(migrateFromV55(context.Background(), v55.Table{
				Key: uuid.New(), Data: msgpack.EncodedJSON{"layout": "not-an-object"},
			})).Error().To(MatchError(ContainSubstring("table data")))
		})
	})

	// Drives the full chain through the real gorp migration pipeline so the
	// on-disk v55 → v56 → typed Table path is exercised end-to-end.
	Describe("storage integration", func() {
		openMigratedTable := func(ctx SpecContext, db *gorp.DB) *gorp.Table[uuid.UUID, table.Table] {
			return MustOpen(gorp.OpenTable[uuid.UUID, table.Table](
				ctx, gorp.TableConfig[uuid.UUID, table.Table]{
					DB: db,
					Migrations: []migrate.Migration{
						gorp.NewEntryMigration[uuid.UUID, uuid.UUID, v55.Table, v56.Table](
							"v55_lift_typed_table",
							v56.MigrateTable,
						),
						migrate.WithAddedDeps(
							gorp.NewEntryMigration[uuid.UUID, uuid.UUID, v56.Table, table.Table](
								"v56_typed_cell_configs",
								table.MigrateTable,
							),
							"v55_lift_typed_table",
						),
					},
				},
			))
		}

		seedV55 := func(ctx SpecContext, db *gorp.DB, name, body string) v55.Table {
			t := MustOpen(gorp.OpenTable[uuid.UUID, v55.Table](
				ctx, gorp.TableConfig[uuid.UUID, v55.Table]{DB: db},
			))
			seed := v55.Table{Key: uuid.New(), Name: name, Data: jsonMap(body)}
			Expect(t.NewCreate().Entry(&seed).Exec(ctx, db)).To(Succeed())
			return seed
		}

		retrieve := func(ctx SpecContext, db *gorp.DB, t *gorp.Table[uuid.UUID, table.Table], key uuid.UUID) table.Table {
			var got table.Table
			Expect(t.NewRetrieve().
				Where(gorp.MatchKeys[table.Key, table.Table](key)).
				Entry(&got).Exec(ctx, db)).To(Succeed())
			return got
		}

		It("Should lift a v0 wire-format blob into the typed Table on retrieve", func(ctx SpecContext) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			seed := seedV55(ctx, db, "Pressure Readouts", `{
				"version": "0.0.0",
				"editable": true,
				"layout": {
					"rows": [{"size": 36, "cells": [{"key": "pt1"}]}],
					"columns": [{"size": 80}]
				},
				"cells": {
					"pt1": {
						"key": "pt1",
						"variant": "value",
						"selected": false,
						"props": {
							"telem": {"props": {"segments": {"valueStream": {"props": {"channel": 7}}}}},
							"color": "#00aaff"
						}
					}
				}
			}`)
			got := retrieve(ctx, db, openMigratedTable(ctx, db), seed.Key)
			Expect(got.Key).To(Equal(seed.Key))
			Expect(got.Name).To(Equal("Pressure Readouts"))
			Expect(got.Rows).To(HaveLen(1))
			Expect(got.Rows[0].Cells).To(Equal([]string{"pt1"}))
			Expect(got.Columns).To(HaveLen(1))
			fields := cfgFields(got.Cells["pt1"])
			Expect(fields).To(SatisfyAll(
				HaveKeyWithValue("variant", "value"),
				HaveKeyWithValue("channel", 7.0),
			))
		})

		It("Should lift a minimal blob lacking layout / cells fields", func(ctx SpecContext) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			seed := seedV55(ctx, db, "Empty", `{"version": "0.0.0"}`)
			got := retrieve(ctx, db, openMigratedTable(ctx, db), seed.Key)
			Expect(got.Rows).To(BeEmpty())
			Expect(got.Columns).To(BeEmpty())
			Expect(got.Cells).To(BeEmpty())
		})
	})
})
