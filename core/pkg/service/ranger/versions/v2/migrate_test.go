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
	"github.com/synnaxlabs/gorp"
	v1 "github.com/synnaxlabs/synnax/pkg/service/ranger/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/ranger/versions/v2"
	color "github.com/synnaxlabs/x/color/versions/v0"
	colorv2 "github.com/synnaxlabs/x/color/versions/v2"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Migration", func() {
	migrate := func(ctx SpecContext, old v1.Range) v2.Range {
		GinkgoHelper()
		db := DeferClose(gorp.Wrap(memkv.New()))
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v1.Key, v1.Range]{DB: db}))
		Expect(gorp.NewCreate[v1.Key, v1.Range]().Entry(&old).Exec(ctx, db)).
			To(Succeed())
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v2.Key, v2.Range]{
			DB: db, Migrations: []gorp.Migration{v2.Migration},
		}))
		var got v2.Range
		Expect(gorp.NewRetrieve[v2.Key, v2.Range]().
			Where(gorp.MatchKeys[v2.Key, v2.Range](old.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		return got
	}

	DescribeTable("Should store the color alpha as a byte",
		func(ctx SpecContext, a float64, expected uint8) {
			old := v1.Range{
				Key: uuid.New(), Name: "r", Color: &color.Color{R: 1, G: 2, B: 3, A: a},
			}
			got := migrate(ctx, old)
			Expect(got.Name).To(Equal("r"))
			Expect(got.Color).
				To(HaveValue(Equal(colorv2.Color{R: 1, G: 2, B: 3, A: expected})))
		},
		Entry("a transparent alpha", 0.0, uint8(0)),
		Entry("a faint alpha", 0.003, uint8(1)),
		Entry("a half alpha", 0.5, uint8(128)),
		Entry("an opaque alpha", 1.0, uint8(255)),
	)

	It("Should keep an absent color absent", func(ctx SpecContext) {
		Expect(migrate(ctx, v1.Range{Key: uuid.New()}).Color).To(BeNil())
	})
})

var _ = Describe("MigrateRange", func() {
	It("Should store the color alpha as a byte", func(ctx SpecContext) {
		out := MustSucceed(
			v2.MigrateRange(ctx, v1.Range{Color: &color.Color{R: 1, A: 0.5}}),
		)
		Expect(out.Color).To(Equal(&colorv2.Color{R: 1, A: 128}))
	})
})
