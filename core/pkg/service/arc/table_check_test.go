// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	"slices"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/arc/versions"
	v3 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v3"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("OpenTable check", func() {
	It("Should migrate a stored v3 Arc when it opens", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		chain := versions.NewMigrations(uuid.New())
		swap := slices.IndexFunc(chain, func(m gorp.Migration) bool {
			return m.Key() == "v61_swap_brackets"
		})
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "Arc",
			Migrations: chain[:swap],
		})).To(Succeed())
		MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[v3.Key, v3.Arc]{DB: db}))
		stored := v3.Arc{
			Key:  uuid.New(),
			Mode: v3.ModeText,
			Text: text.Text{Doc: text.Create("func my_fn{k f64}() {\n    out = k\n}")},
		}
		Expect(gorp.NewCreate[v3.Key, v3.Arc]().Entry(&stored).Exec(ctx, db)).
			To(Succeed())
		MustOpen(arc.OpenTable(ctx, arc.TableConfig{DB: db}))
		var migrated arc.Arc
		Expect(gorp.NewRetrieve[arc.Key, arc.Arc]().
			Where(gorp.MatchKeys[arc.Key, arc.Arc](stored.Key)).
			Entry(&migrated).
			Exec(ctx, db)).To(Succeed())
		src, ok := migrated.Base().Source.Variant.(arc.TextSource)
		Expect(ok).To(BeTrue())
		Expect(src.Text.Materialize().Raw).
			To(Equal("func my_fn(k f64){} {\n    out = k\n}"))
	})
})
