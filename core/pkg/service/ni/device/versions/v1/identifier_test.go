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
	v1 "github.com/synnaxlabs/synnax/pkg/service/ni/device/versions/v1"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("IdentifierMigration", func() {
	DescribeTable("Should keep only identifiers that follow the rule",
		func(ctx SpecContext, stored, want string) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			StartRun(db.Run)
			MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v1.Key, v1.Device]{DB: db}))
			d := v1.Device{
				Key:        uuid.New(),
				Rack:       1,
				Name:       "PXI-6255",
				Model:      "PXI-6255",
				Identifier: stored,
			}
			Expect(gorp.NewCreate[v1.Key, v1.Device]().Entry(&d).Exec(ctx, db)).
				To(Succeed())
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:         db,
				Namespace:  "ni_device",
				Migrations: []gorp.Migration{v1.IdentifierMigration},
			})).To(Succeed())
			var got v1.Device
			Expect(gorp.NewRetrieve[v1.Key, v1.Device]().
				Where(gorp.MatchKeys[v1.Key, v1.Device](d.Key)).
				Entry(&got).
				Exec(ctx, db)).To(Succeed())
			Expect(got.Identifier).To(Equal(want))
			Expect(got.Name).To(Equal("PXI-6255"))
		},
		Entry("clearing a bad identifier", "dev-1", ""),
		Entry("clearing a one-character identifier", "x", ""),
		Entry("keeping a valid identifier", "dev_1", "dev_1"),
		Entry("keeping an empty identifier", "", ""),
	)
})
