// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	v2 "github.com/synnaxlabs/synnax/pkg/service/ranger/versions/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/ranger/versions/v3"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Migration", func() {
	It("Should keep every field of a stored range", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		parent := uuid.New()
		old := v2.Range{
			Key:       uuid.New(),
			Name:      "Range",
			TimeRange: telem.NewRangeSeconds(1, 2),
			Parent:    &parent,
			Labels:    []uuid.UUID{uuid.New()},
		}
		Expect(gorp.WrapWriter[v2.Key, v2.Range](db).Set(ctx, old)).To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "Range",
			Migrations: []gorp.Migration{v3.Migration},
		})).To(Succeed())
		var r v3.Range
		Expect(gorp.NewRetrieve[v3.Key, v3.Range]().
			Where(gorp.MatchKeys[v3.Key, v3.Range](old.Key)).
			Entry(&r).
			Exec(ctx, db)).To(Succeed())
		Expect(r).To(Equal(v3.Range{
			Key:       old.Key,
			Name:      old.Name,
			TimeRange: old.TimeRange,
			Parent:    old.Parent,
			Labels:    old.Labels,
		}))
	})
})

var _ = Describe("MigrateWhere", func() {
	It("Should keep each line through every branch", func(ctx SpecContext) {
		label, rng := uuid.New(), uuid.New()
		tr := telem.NewRangeSeconds(1, 2)
		old := v2.Where{
			And: []v2.Where{{Labels: []uuid.UUID{label}}},
			Or:  []v2.Where{{Parents: []uuid.UUID{rng}}, {Overlaps: &tr}},
			Not: &v2.Where{Labels: []uuid.UUID{label}},
		}
		Expect(v3.MigrateWhere(ctx, old)).To(Equal(v3.Where{
			And: []v3.Where{{Labels: []uuid.UUID{label}}},
			Or:  []v3.Where{{Parents: []uuid.UUID{rng}}, {Overlaps: &tr}},
			Not: &v3.Where{Labels: []uuid.UUID{label}},
		}))
	})
})
