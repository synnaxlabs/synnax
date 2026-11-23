// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v5_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	servicecolor "github.com/synnaxlabs/synnax/pkg/service/color/versions/v0"
	v4 "github.com/synnaxlabs/synnax/pkg/service/ranger/versions/v4"
	v5 "github.com/synnaxlabs/synnax/pkg/service/ranger/versions/v5"
	color "github.com/synnaxlabs/x/color/versions/v2"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Migration", func() {
	migrate := func(ctx SpecContext, old v4.Range) v5.Range {
		GinkgoHelper()
		db := DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v4.Key, v4.Range]{DB: db}))
		Expect(gorp.NewCreate[v4.Key, v4.Range]().Entry(&old).Exec(ctx, db)).
			To(Succeed())
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v5.Key, v5.Range]{
			DB: db, Migrations: []gorp.Migration{v5.Migration},
		}))
		var got v5.Range
		Expect(gorp.NewRetrieve[v5.Key, v5.Range]().
			Where(gorp.MatchKeys[v5.Key, v5.Range](old.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		return got
	}

	It("Should store the color of a range that v0.62 stored as a fixed color",
		func(ctx SpecContext) {
			parent, label := uuid.New(), uuid.New()
			old := v4.Range{
				Key:       uuid.New(),
				Name:      "Hot fire",
				TimeRange: telem.TimeRange{Start: 1, End: 2},
				Color:     &color.Color{R: 1, G: 2, B: 3, A: 128},
				Labels:    []uuid.UUID{label},
				Parent:    &parent,
			}
			got := migrate(ctx, old)
			Expect(got.Name).To(Equal("Hot fire"))
			Expect(got.TimeRange).To(Equal(old.TimeRange))
			Expect(got.Labels).To(Equal(old.Labels))
			Expect(got.Parent).To(Equal(old.Parent))
			Expect(got.Color).To(Equal(&servicecolor.Ref{
				Variant: servicecolor.FixedRef{
					Value: color.Color{R: 1, G: 2, B: 3, A: 128},
				},
			}))
		},
	)

	It("Should keep an absent range color absent", func(ctx SpecContext) {
		Expect(migrate(ctx, v4.Range{Key: uuid.New(), Name: "r"}).Color).To(BeNil())
	})
})
