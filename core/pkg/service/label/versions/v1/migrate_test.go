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
	v0 "github.com/synnaxlabs/synnax/pkg/service/label/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/label/versions/v1"
	color "github.com/synnaxlabs/x/color/versions/v0"
	colorv2 "github.com/synnaxlabs/x/color/versions/v2"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Migration", func() {
	DescribeTable("Should store the color alpha as a byte",
		func(ctx SpecContext, a float64, expected uint8) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v0.Key, v0.Label]{DB: db}))
			old := v0.Label{
				Key: uuid.New(), Name: "l", Color: color.Color{R: 1, G: 2, B: 3, A: a},
			}
			Expect(gorp.NewCreate[v0.Key, v0.Label]().Entry(&old).Exec(ctx, db)).
				To(Succeed())
			MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v1.Key, v1.Label]{
				DB: db, Migrations: []gorp.Migration{v1.Migration},
			}))
			var got v1.Label
			Expect(gorp.NewRetrieve[v1.Key, v1.Label]().
				Where(gorp.MatchKeys[v1.Key, v1.Label](old.Key)).
				Entry(&got).Exec(ctx, db)).To(Succeed())
			Expect(got).To(Equal(v1.Label{
				Key:   old.Key,
				Name:  "l",
				Color: colorv2.Color{R: 1, G: 2, B: 3, A: expected},
			}))
		},
		Entry("a transparent alpha", 0.0, uint8(0)),
		Entry("a faint alpha", 0.003, uint8(1)),
		Entry("a half alpha", 0.5, uint8(128)),
		Entry("an opaque alpha", 1.0, uint8(255)),
	)
})

var _ = Describe("MigrateLabel", func() {
	It("Should store the color alpha as a byte", func(ctx SpecContext) {
		out := MustSucceed(v1.MigrateLabel(ctx, v0.Label{
			Name: "l", Color: color.Color{R: 1, A: 0.5},
		}))
		Expect(out.Color).To(Equal(colorv2.Color{R: 1, A: 128}))
	})
})
