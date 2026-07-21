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
	"embed"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	v1 "github.com/synnaxlabs/synnax/pkg/service/table/types/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/table/types/v2"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
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

// seedV1 stores a v1 wire-format table in a fresh gorp DB and returns the DB.
func seedV1(ctx SpecContext, seed *v1.Table) *gorp.DB {
	db := DeferClose(gorp.Wrap(memkv.New()))
	MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[uuid.UUID, v1.Table]{DB: db}))
	Expect(gorp.NewCreate[uuid.UUID, v1.Table]().Entry(seed).Exec(ctx, db)).
		To(Succeed())
	return db
}

// migrateSeed runs the v2 migration chain over a seeded v1 table and returns the
// migrated typed Table.
func migrateSeed(ctx SpecContext, seed v1.Table) table.Table {
	db := seedV1(ctx, &seed)
	Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
		DB:         db,
		Namespace:  "Table",
		Migrations: v2.Migrations,
	})).To(Succeed())
	var got table.Table
	Expect(gorp.NewRetrieve[table.Key, table.Table]().
		Where(gorp.MatchKeys[table.Key, table.Table](seed.Key)).
		Entry(&got).Exec(ctx, db)).To(Succeed())
	return got
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

var _ = Describe("v1 -> current Table migration", func() {
	// Snapshot tests against the canonical .migrated.json output for every
	// captured fixture. Run with UPDATE_MIGRATED=1 to regenerate the
	// .migrated.json files after intentional migration changes.
	Describe("canonical migrated output", func() {
		fixedKey := uuid.MustParse("00000000-0000-0000-0000-000000000001")
		DescribeTable("Should produce the canonical typed Table",
			func(ctx SpecContext, fixture string) {
				blob := loadFixture(fixture)
				out := migrateSeed(ctx, v1.Table{
					Key: fixedKey, Name: fixture, Data: blob,
				})
				assertMigrated(fixture, out)
			},
			Entry("v0 empty", "v0_empty.json"),
			Entry("v0 mixed variants", "v0_mixed_variants.json"),
		)
	})

	Describe("v0 reshape semantics", func() {
		migrateBody := func(ctx SpecContext, body string) table.Table {
			return migrateSeed(ctx, v1.Table{Key: uuid.New(), Data: jsonMap(body)})
		}

		It("Should flatten layout.rows[*].cells from CellLayout[] to []string", func(ctx SpecContext) {
			out := migrateBody(ctx, `{
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
			out := migrateBody(ctx, `{
				"version": "0.0.0",
				"layout": {"rows": [], "columns": []},
				"cells": {
					"a": {"key": "a", "variant": "text", "selected": true, "props": {"value": "hi"}}
				}
			}`)
			cell, ok := out.Cells["a"]
			Expect(ok).To(BeTrue())
			Expect(cell.Variant).To(Equal("text"))
			Expect(cell.Props["value"]).To(Equal("hi"))
			Expect(cell.Props).NotTo(HaveKey("selected"))
		})

		It("Should preserve arbitrary key casing inside cell props through the migration", func(ctx SpecContext) {
			out := migrateBody(ctx, `{
				"version": "0.0.0",
				"layout": {"rows": [], "columns": []},
				"cells": {
					"a": {
						"key": "a", "variant": "value", "selected": false,
						"props": {
							"camelCaseKey": "v1",
							"PascalCaseKey": "v2",
							"snake_case_key": "v3"
						}
					}
				}
			}`)
			Expect(out.Cells["a"].Props).To(SatisfyAll(
				HaveKeyWithValue("camelCaseKey", "v1"),
				HaveKeyWithValue("PascalCaseKey", "v2"),
				HaveKeyWithValue("snake_case_key", "v3"),
			))
		})

		It("Should pass through the gorp-entry fields (Key, Name)", func(ctx SpecContext) {
			key := uuid.New()
			out := migrateSeed(ctx, v1.Table{
				Key: key, Name: "trip-table", Data: jsonMap(`{"version": "0.0.0"}`),
			})
			Expect(out.Key).To(Equal(key))
			Expect(out.Name).To(Equal("trip-table"))
		})

		DescribeTable("Should produce empty (not nil) collections for empty inputs",
			func(ctx SpecContext, data msgpack.EncodedJSON) {
				out := migrateSeed(ctx, v1.Table{
					Key: uuid.New(), Name: "empty", Data: data,
				})
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

		It("Should preserve cell key, variant, and full props through the migration", func(ctx SpecContext) {
			out := migrateBody(ctx, `{
				"version": "0.0.0",
				"layout": {"rows": [], "columns": []},
				"cells": {
					"alpha": {
						"key": "alpha",
						"variant": "value",
						"selected": false,
						"props": {
							"telem": {"channel": 42},
							"redline": {"lower": 0, "upper": 100},
							"color": "#ff0000"
						}
					}
				}
			}`)
			cell := out.Cells["alpha"]
			Expect(cell.Key).To(Equal("alpha"))
			Expect(cell.Variant).To(Equal("value"))
			Expect(cell.Props).To(SatisfyAll(
				HaveKey("telem"),
				HaveKey("redline"),
				HaveKeyWithValue("color", "#ff0000"),
			))
		})

		It("Should preserve multi-row layout ordering and per-row size", func(ctx SpecContext) {
			out := migrateBody(ctx, `{
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

	Describe("malformed input", func() {
		It("Should return an error for an invalid Data shape", func(ctx SpecContext) {
			db := seedV1(ctx, &v1.Table{
				Key: uuid.New(), Data: msgpack.EncodedJSON{"layout": "not-an-object"},
			})
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:         db,
				Namespace:  "Table",
				Migrations: v2.Migrations,
			})).To(MatchError(ContainSubstring("table data")))
		})
	})

	Describe("storage integration", func() {
		It("Should lift a v0 wire-format blob into the typed Table on retrieve", func(ctx SpecContext) {
			key := uuid.New()
			got := migrateSeed(ctx, v1.Table{
				Key: key, Name: "Pressure Readouts", Data: jsonMap(`{
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
							"props": {"telem": {"channel": 7}, "color": "#00aaff"}
						}
					}
				}`),
			})
			Expect(got.Key).To(Equal(key))
			Expect(got.Name).To(Equal("Pressure Readouts"))
			Expect(got.Rows).To(HaveLen(1))
			Expect(got.Rows[0].Cells).To(Equal([]string{"pt1"}))
			Expect(got.Columns).To(HaveLen(1))
			Expect(got.Cells["pt1"].Variant).To(Equal("value"))
			Expect(got.Cells["pt1"].Props["color"]).To(Equal("#00aaff"))
		})

		It("Should lift a minimal blob lacking layout / cells fields", func(ctx SpecContext) {
			got := migrateSeed(ctx, v1.Table{
				Key: uuid.New(), Name: "Empty", Data: jsonMap(`{"version": "0.0.0"}`),
			})
			Expect(got.Rows).To(BeEmpty())
			Expect(got.Columns).To(BeEmpty())
			Expect(got.Cells).To(BeEmpty())
		})
	})
})
