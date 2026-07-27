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
	"context"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	v0 "github.com/synnaxlabs/synnax/pkg/service/ranger/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/ranger/versions/v1"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("v0 -> v1 Range migration", func() {
	migrateColors := func(ctx SpecContext, seeds ...v0.Range) map[v1.Key]v1.Range {
		db := DeferClose(gorp.Wrap(memkv.New()))
		MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[v0.Key, v0.Range]{DB: db}))
		for i := range seeds {
			Expect(gorp.NewCreate[v0.Key, v0.Range]().Entry(&seeds[i]).Exec(ctx, db)).
				To(Succeed())
		}
		v0Chain := v0.NewMigrations(v0.MigrationConfig{})
		v0Applied := gorp.NewMigration(
			v0Chain[0].Key(),
			func(context.Context, gorp.Tx, alamos.Instrumentation) error { return nil },
		)
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:        db,
			Namespace: "Range",
			Migrations: append(
				append([]migrate.Migration{v0Applied}, v0Chain[1:]...), v1.Migration,
			),
		})).To(Succeed())
		out := make(map[v1.Key]v1.Range, len(seeds))
		for _, seed := range seeds {
			var rng v1.Range
			Expect(gorp.NewRetrieve[v1.Key, v1.Range]().
				Where(gorp.MatchKeys[v1.Key, v1.Range](seed.Key)).
				Entry(&rng).Exec(ctx, db)).To(Succeed())
			out[seed.Key] = rng
		}
		return out
	}

	It("Should map a zero stored color to a nil pointer", func(ctx SpecContext) {
		key := uuid.New()
		out := migrateColors(ctx, v0.Range{Key: key, Name: "no color"})
		Expect(out[key].Name).To(Equal("no color"))
		Expect(out[key].Color).To(BeNil())
	})

	It("Should preserve a non-zero color as a pointer", func(ctx SpecContext) {
		key := uuid.New()
		red := color.Color{R: 255, A: 1}
		out := migrateColors(ctx, v0.Range{Key: key, Name: "red", Color: red})
		Expect(out[key].Color).ToNot(BeNil())
		Expect(*out[key].Color).To(Equal(red))
	})
})
