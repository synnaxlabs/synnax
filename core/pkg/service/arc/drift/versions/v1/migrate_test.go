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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	v0 "github.com/synnaxlabs/synnax/pkg/service/arc/drift/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/arc/drift/versions/v1"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Migration", func() {
	It("Should lift each record with its hashes and no deployed channels", func(
		ctx SpecContext,
	) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v0.Key, v0.Record]{DB: db}))
		old := v0.Record{
			Task:     uuid.New(),
			Stored:   v0.Hashes{Text: "b", Modules: map[string]string{"m": "2"}},
			Deployed: v0.Hashes{Text: "a", Modules: map[string]string{"m": "1"}},
		}
		Expect(gorp.NewCreate[v0.Key, v0.Record]().Entry(&old).Exec(ctx, db)).
			To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "Record",
			Migrations: []gorp.Migration{v1.Migration},
		})).To(Succeed())
		var out v1.Record
		Expect(gorp.NewRetrieve[v1.Key, v1.Record]().
			Where(gorp.MatchKeys[v1.Key, v1.Record](old.Task)).
			Entry(&out).
			Exec(ctx, db)).To(Succeed())
		Expect(out).To(Equal(v1.Record{
			Task:     old.Task,
			Stored:   old.Stored,
			Deployed: old.Deployed,
		}))
	})
})
