// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versions_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/table/versions"
	v2 "github.com/synnaxlabs/synnax/pkg/service/table/versions/v2"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Migrations", func() {
	It(
		"Should key the rows and columns of a table stored by a v2 Core",
		func(ctx SpecContext) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			v2Chain := versions.Migrations[:len(versions.Migrations)-2]
			MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[v2.Key, v2.Table]{
				DB:         db,
				Migrations: v2Chain,
			}))
			stored := v2.Table{
				Key:  uuid.New(),
				Name: "t",
				Rows: []v2.Row{
					{Size: 36, Cells: []string{"a", "b"}},
					{Size: 40, Cells: []string{"c", "d"}},
				},
				Columns: []v2.Column{{Size: 72}, {Size: 100}},
				Cells: map[string]v2.CellConfig{
					"a": {Variant: v2.TextCellConfig{Value: "A"}},
				},
			}
			stored.ApplyDefaults()
			Expect(gorp.NewCreate[v2.Key, v2.Table]().Entry(&stored).Exec(ctx, db)).
				To(Succeed())
			MustSucceed(
				gorp.OpenTable(ctx, gorp.TableConfig[versions.Key, versions.Table]{
					DB:         db,
					Migrations: versions.Migrations,
				}),
			)
			var got versions.Table
			Expect(gorp.NewRetrieve[versions.Key, versions.Table]().
				Where(gorp.MatchKeys[versions.Key, versions.Table](stored.Key)).
				Entry(&got).Exec(ctx, db)).To(Succeed())
			Expect(got.Rows).To(Equal([]versions.Row{
				{Key: "row-0", Size: 36, Cells: []string{"a", "b"}},
				{Key: "row-1", Size: 40, Cells: []string{"c", "d"}},
			}))
			Expect(got.Columns).To(Equal([]versions.Column{
				{Key: "col-0", Size: 72},
				{Key: "col-1", Size: 100},
			}))
			Expect(got.Cells).To(HaveKey("a"))
			Expect(got.Validate()).To(Succeed())
		},
	)
	// Applied migrations are tracked by key, so renaming or reordering an entry
	// re-runs or skips it on stores that already migrated. The chain is pinned
	// verbatim; extend it for a new version, never edit shipped entries.
	It("Should pin the migration chain keys in order", func() {
		Expect(lo.Map(
			versions.Migrations,
			func(m gorp.Migration, _ int) string { return m.Key() },
		)).To(Equal([]string{
			"normalize_keys",
			"msgpack_to_orc",
			"v55_lift_typed_table",
			"v2_typed_cell_configs",
			"v3_row_and_column_keys",
			"v61_color_alpha_byte",
		}))
	})
})
