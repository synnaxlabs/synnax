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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/synnaxlabs/synnax/pkg/service/project/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/project/versions/v2"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("MigrateProject", func() {
	migrateSeed := func(ctx SpecContext, seed v1.Project) v2.Project {
		GinkgoHelper()
		db := DeferClose(gorp.Wrap(memkv.New()))
		MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[v1.Key, v1.Project]{DB: db}))
		Expect(gorp.NewCreate[v1.Key, v1.Project]().
			Entry(&seed).Exec(ctx, db)).To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "Project",
			Migrations: []migrate.Migration{v2.Migration},
		})).To(Succeed())
		var got v2.Project
		Expect(gorp.NewRetrieve[v2.Key, v2.Project]().
			Where(gorp.MatchKeys[v2.Key, v2.Project](seed.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		return got
	}

	It("Should drop the layout and preserve every other field", func(ctx SpecContext) {
		key := uuid.New()
		Expect(migrateSeed(ctx, v1.Project{
			Key:    key,
			Name:   "Ops",
			Layout: msgpack.EncodedJSON{"mosaic": "tree"},
		})).To(Equal(v2.Project{Key: key, Name: "Ops"}))
	})

	It("Should migrate a project with no layout", func(ctx SpecContext) {
		key := uuid.New()
		Expect(migrateSeed(ctx, v1.Project{Key: key, Name: "Empty"})).
			To(Equal(v2.Project{Key: key, Name: "Empty"}))
	})
})
