// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package table_test

import (
	"encoding/json"
	"os"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	. "github.com/synnaxlabs/synnax/pkg/service/imex/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	"github.com/synnaxlabs/synnax/pkg/service/table/versions"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// loadEnvelope reads a wire-format envelope fixture from versions/testdata and
// unmarshals it into an imex.Envelope, binding the codec that Decode needs.
func loadEnvelope(path string) imex.Envelope {
	raw := MustSucceed(os.ReadFile(path))
	var env imex.Envelope
	Expect(json.Unmarshal(raw, &env)).To(Succeed())
	return env
}

var _ = Describe("ImEx", func() {
	Describe("Export", func() {
		It("Should export a table as a versioned envelope", func(ctx SpecContext) {
			t := table.Table{Name: "exported"}
			Expect(svc.NewWriter(nil).Create(ctx, proj.Key, &t)).To(Succeed())
			env := MustSucceed(svc.Export(ctx, table.OntologyID(t.Key)))
			Expect(env.Version).To(Equal(versions.Latest))
			Expect(env.Type).To(Equal("table"))
			Expect(env.Name).To(Equal("exported"))

			decoded := MustSucceed(imex.Decode[table.Table](ctx, WireRoundTrip(env)))
			Expect(decoded.Name).To(Equal("exported"))
		})

		It("Should return not found for a missing key", func(ctx SpecContext) {
			id := ontology.ID{Type: ontology.ResourceTypeTable, Key: uuid.NewString()}
			Expect(svc.Export(ctx, id)).Error().To(MatchError(query.ErrNotFound))
		})

		It("Should error on an invalid UUID key", func(ctx SpecContext) {
			id := ontology.ID{Type: ontology.ResourceTypeTable, Key: "not-a-uuid"}
			Expect(svc.Export(ctx, id)).Error().To(MatchError(ContainSubstring("UUID")))
		})
	})

	Describe("Import", func() {
		importAndRetrieve := func(
			ctx SpecContext, path string, opts imex.ImportOptions,
		) table.Table {
			id := MustSucceed(imexSvc.Import(ctx, db, loadEnvelope(path), opts))
			Expect(id.Type).To(Equal(ontology.ResourceTypeTable))
			key := MustSucceed(uuid.Parse(id.Key))
			var res table.Table
			Expect(svc.NewRetrieve().
				Where(table.MatchKeys(key)).
				Entry(&res).
				Exec(ctx, db)).To(Succeed())
			return res
		}

		It("Should import a current snake_case envelope", func(ctx SpecContext) {
			res := importAndRetrieve(
				ctx, "versions/testdata/import_v2.json", imex.ImportOptions{},
			)
			Expect(res.Name).To(Equal("Server Typed"))
			Expect(res.Rows).To(Equal([]table.Row{{Size: 30, Cells: []string{"c1"}}}))
			Expect(res.Columns).To(Equal([]table.Column{{Size: 100}}))
			Expect(res.Cells).To(HaveKey("c1"))
			Expect(res.Cells["c1"].Variant).To(Equal("text"))
		})

		It(
			"Should import a Console typed export carrying camelCase keys",
			func(ctx SpecContext) {
				res := importAndRetrieve(
					ctx,
					"versions/testdata/import_typed_console.json",
					imex.ImportOptions{},
				)
				Expect(res.Name).To(Equal("Console Typed"))
				Expect(res.Rows).To(HaveLen(1))
				var props map[string]any
				Expect(res.Cells["c1"].Props.Unmarshal(&props)).To(Succeed())
				Expect(props).To(HaveKeyWithValue("fooBar", 1.0))
			},
		)

		It(
			"Should import a v1 Console state from its pendingUpload",
			func(ctx SpecContext) {
				res := importAndRetrieve(
					ctx, "versions/testdata/import_v1_state.json",
					imex.ImportOptions{FileName: "My Table.json"},
				)
				Expect(res.Name).To(Equal("My Table"))
				Expect(
					res.Rows,
				).To(Equal([]table.Row{{Size: 30, Cells: []string{"c1"}}}))
				var props map[string]any
				Expect(res.Cells["c1"].Props.Unmarshal(&props)).To(Succeed())
				Expect(props).To(HaveKeyWithValue("fooBar", 2.0))
			},
		)

		It(
			"Should reject a v1 Console state with no structural data",
			func(ctx SpecContext) {
				Expect(imexSvc.Import(ctx, db,
					loadEnvelope("versions/testdata/import_v1_state_empty.json"),
					imex.ImportOptions{FileName: "empty.json"},
				)).Error().To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring("no structural data")),
				))
			},
		)

		It(
			"Should import a v0 Console state through the legacy chain",
			func(ctx SpecContext) {
				res := importAndRetrieve(
					ctx, "versions/testdata/import_v0_state.json",
					imex.ImportOptions{FileName: "Legacy Table.json"},
				)
				Expect(res.Name).To(Equal("Legacy Table"))
				Expect(res.Rows).To(
					Equal([]table.Row{{Size: 30, Cells: []string{"c1", "c2"}}}),
				)
				Expect(res.Columns).To(Equal([]table.Column{{Size: 100}, {Size: 120}}))
				Expect(res.Cells).To(HaveLen(2))
				Expect(res.Cells["c2"].Variant).To(Equal("value"))
				var props map[string]any
				Expect(res.Cells["c1"].Props.Unmarshal(&props)).To(Succeed())
				Expect(props).To(HaveKeyWithValue("fooBar", 3.0))
			},
		)

		It(
			"Should reject an envelope newer than the supported version",
			func(ctx SpecContext) {
				Expect(imexSvc.Import(ctx, db,
					loadEnvelope("versions/testdata/import_bad_version.json"),
					imex.ImportOptions{},
				)).Error().To(SatisfyAll(
					MatchError(ContainSubstring("table version 99")),
					MatchError(ContainSubstring("newer than this Core supports")),
				))
			},
		)

		It(
			"Should generate a fresh key, discarding the key on the wire",
			func(ctx SpecContext) {
				id := MustSucceed(imexSvc.Import(
					ctx,
					db,
					loadEnvelope(
						"versions/testdata/import_v2.json",
					),
					imex.ImportOptions{},
				))
				Expect(id.Key).ToNot(Equal("11111111-2222-3333-4444-555555555555"))
			},
		)
	})

	Describe("Round trip", func() {
		It(
			"Should preserve table content through export then import",
			func(ctx SpecContext) {
				original := table.Table{
					Name:    "round-trip",
					Rows:    []table.Row{{Size: 42, Cells: []string{"c1"}}},
					Columns: []table.Column{{Size: 77}},
				}
				Expect(
					svc.NewWriter(nil).Create(ctx, proj.Key, &original),
				).To(Succeed())
				env := MustSucceed(svc.Export(ctx, table.OntologyID(original.Key)))
				id := MustSucceed(imexSvc.Import(
					ctx, db, WireRoundTrip(env), imex.ImportOptions{},
				))
				key := MustSucceed(uuid.Parse(id.Key))
				Expect(key).ToNot(Equal(original.Key))
				var res table.Table
				Expect(svc.NewRetrieve().
					Where(table.MatchKeys(key)).
					Entry(&res).
					Exec(ctx, db)).To(Succeed())
				Expect(res.Name).To(Equal("round-trip"))
				Expect(res.Rows).To(Equal(original.Rows))
				Expect(res.Columns).To(Equal(original.Columns))
			},
		)
	})
})
