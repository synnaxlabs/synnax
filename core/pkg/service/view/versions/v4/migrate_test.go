// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v4_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	rangerv3 "github.com/synnaxlabs/synnax/pkg/service/ranger/versions/v3"
	statusv2 "github.com/synnaxlabs/synnax/pkg/service/status/versions/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/view/versions/v3"
	v4 "github.com/synnaxlabs/synnax/pkg/service/view/versions/v4"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Migration", func() {
	It("Should keep each range and status view as it was", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v3.Key, v3.View]{DB: db}))
		label := uuid.New()
		ranges := v3.Base{Key: uuid.New(), Name: "Ranges"}
		statuses := v3.Base{Key: uuid.New(), Name: "Statuses"}
		views := []v3.View{
			{Variant: v3.RangeView{
				Base:   ranges,
				Where:  &rangerv3.Where{Labels: []uuid.UUID{label}},
				Sort:   v3.RangeSortName,
				Layout: v3.RangeLayoutList,
			}},
			{Variant: v3.StatusView{
				Base:  statuses,
				Where: &statusv2.Where{Labels: []uuid.UUID{label}},
			}},
		}
		Expect(gorp.NewCreate[v3.Key, v3.View]().Entries(&views).Exec(ctx, db)).
			To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "View",
			Migrations: []gorp.Migration{v4.Migration},
		})).To(Succeed())
		var out []v4.View
		Expect(gorp.NewRetrieve[v4.Key, v4.View]().Entries(&out).Exec(ctx, db)).
			To(Succeed())
		Expect(out).To(ConsistOf(
			v4.View{Variant: v4.RangeView{
				Base:   ranges,
				Where:  &rangerv3.Where{Labels: []uuid.UUID{label}},
				Sort:   v4.RangeSortName,
				Layout: v4.RangeLayoutList,
			}},
			v4.View{Variant: v4.StatusView{
				Base:  statuses,
				Where: &statusv2.Where{Labels: []uuid.UUID{label}},
			}},
		))
	})
})
