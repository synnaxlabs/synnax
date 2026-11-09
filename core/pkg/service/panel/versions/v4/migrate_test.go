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
	v1 "github.com/synnaxlabs/synnax/pkg/service/panel/versions/v1"
	v3 "github.com/synnaxlabs/synnax/pkg/service/panel/versions/v3"
	v4 "github.com/synnaxlabs/synnax/pkg/service/panel/versions/v4"
	color "github.com/synnaxlabs/x/color/versions/v0"
	colorv2 "github.com/synnaxlabs/x/color/versions/v2"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

func root() v1.Node { return v1.Node{Variant: v1.LeafNode{Tabs: []v1.Tab{}}} }

var _ = Describe("Migration", func() {
	migrate := func(ctx SpecContext, old v3.Panel) v4.Panel {
		GinkgoHelper()
		db := DeferClose(gorp.Wrap(memkv.New()))
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v3.Key, v3.Panel]{DB: db}))
		Expect(gorp.NewCreate[v3.Key, v3.Panel]().Entry(&old).Exec(ctx, db)).
			To(Succeed())
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v4.Key, v4.Panel]{
			DB: db, Migrations: []gorp.Migration{v4.Migration},
		}))
		var got v4.Panel
		Expect(gorp.NewRetrieve[v4.Key, v4.Panel]().
			Where(gorp.MatchKeys[v4.Key, v4.Panel](old.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		return got
	}

	DescribeTable("Should store the icon color alpha as a byte",
		func(ctx SpecContext, a float64, expected uint8) {
			old := v3.Panel{
				Key:   uuid.New(),
				Name:  "p",
				Root:  root(),
				Color: &color.Color{R: 1, G: 2, B: 3, A: a},
			}
			Expect(migrate(ctx, old).Color).
				To(HaveValue(Equal(colorv2.Color{R: 1, G: 2, B: 3, A: expected})))
		},
		Entry("a transparent alpha", 0.0, uint8(0)),
		Entry("a faint alpha", 0.003, uint8(1)),
		Entry("a half alpha", 0.5, uint8(128)),
		Entry("an opaque alpha", 1.0, uint8(255)),
	)

	It("Should keep an absent icon color absent", func(ctx SpecContext) {
		Expect(migrate(ctx, v3.Panel{Key: uuid.New(), Root: root()}).Color).To(BeNil())
	})
})

var _ = Describe("MigratePanel", func() {
	It("Should store the icon color alpha as a byte", func(ctx SpecContext) {
		out := MustSucceed(v4.MigratePanel(ctx, v3.Panel{
			Root: root(), Color: &color.Color{R: 1, A: 0.5},
		}))
		Expect(out.Color).To(Equal(&colorv2.Color{R: 1, A: 128}))
	})
})
