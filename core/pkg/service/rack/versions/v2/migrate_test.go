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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	v0 "github.com/synnaxlabs/synnax/pkg/service/rack/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/rack/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/rack/versions/v2"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("MigrateRack", func() {
	migrateSeed := func(ctx SpecContext, seed v0.Rack) v2.Rack {
		db := DeferClose(gorp.Wrap(memkv.New()))
		MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[v0.Key, v0.Rack]{DB: db}))
		Expect(gorp.NewCreate[v0.Key, v0.Rack]().
			Entry(&seed).Exec(ctx, db)).To(Succeed())
		v0Chain := v0.NewMigrations(v0.MigrationConfig{})
		v0Applied := gorp.NewMigration(
			v0Chain[0].Key(),
			func(context.Context, gorp.Tx, alamos.Instrumentation) error { return nil },
		)
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:        db,
			Namespace: "Rack",
			Migrations: append(
				append([]migrate.Migration{v0Applied}, v0Chain[1:]...),
				v1.Migration,
				v2.Migration,
			),
		})).To(Succeed())
		var got v2.Rack
		Expect(gorp.NewRetrieve[v2.Key, v2.Rack]().
			Where(gorp.MatchKeys[v2.Key, v2.Rack](seed.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		return got
	}

	It("drops the task counter and preserves every other field", func(ctx SpecContext) {
		migrated := migrateSeed(ctx, v0.Rack{
			Key:          v0.Key(0x0001_0009),
			Name:         "Node 1",
			TaskCounter:  42,
			Embedded:     true,
			Integrations: []string{"ni", "opc"},
		})
		Expect(migrated.Key).To(Equal(v2.Key(0x0001_0009)))
		Expect(migrated.Name).To(Equal("Node 1"))
		Expect(migrated.Embedded).To(BeTrue())
		Expect(migrated.Integrations).To(Equal([]string{"ni", "opc"}))
	})

	It("leaves a rack that never minted a task key intact", func(ctx SpecContext) {
		migrated := migrateSeed(ctx, v0.Rack{
			Key:      v0.Key(0x0002_0001),
			Name:     "Fresh Rack",
			Embedded: false,
		})
		Expect(migrated.Key).To(Equal(v2.Key(0x0002_0001)))
		Expect(migrated.Name).To(Equal("Fresh Rack"))
		Expect(migrated.Embedded).To(BeFalse())
		Expect(migrated.Integrations).To(BeEmpty())
	})
})
