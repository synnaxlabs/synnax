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
	v1 "github.com/synnaxlabs/synnax/pkg/service/panel/versions/v1"
	v4 "github.com/synnaxlabs/synnax/pkg/service/panel/versions/v4"
	v5 "github.com/synnaxlabs/synnax/pkg/service/panel/versions/v5"
	color "github.com/synnaxlabs/x/color/versions/v0"
	colorv2 "github.com/synnaxlabs/x/color/versions/v2"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

func root() v1.Node { return v1.Node{Variant: v1.LeafNode{Tabs: []v1.Tab{}}} }

func fixed(c colorv2.Color) *servicecolor.Ref {
	return &servicecolor.Ref{Variant: servicecolor.FixedRef{Value: c}}
}

var _ = Describe("Migration", func() {
	migrate := func(ctx SpecContext, old v4.Panel) v5.Panel {
		GinkgoHelper()
		db := DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v4.Key, v4.Panel]{DB: db}))
		Expect(gorp.NewCreate[v4.Key, v4.Panel]().Entry(&old).Exec(ctx, db)).
			To(Succeed())
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v5.Key, v5.Panel]{
			DB: db, Migrations: []gorp.Migration{v5.Migration},
		}))
		var got v5.Panel
		Expect(gorp.NewRetrieve[v5.Key, v5.Panel]().
			Where(gorp.MatchKeys[v5.Key, v5.Panel](old.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		return got
	}

	DescribeTable("Should store the icon color as a fixed color with a byte alpha",
		func(ctx SpecContext, a float64, expected uint8) {
			old := v4.Panel{
				Key:   uuid.New(),
				Name:  "p",
				Root:  root(),
				Color: &color.Color{R: 1, G: 2, B: 3, A: a},
			}
			Expect(migrate(ctx, old).Color).
				To(Equal(fixed(colorv2.Color{R: 1, G: 2, B: 3, A: expected})))
		},
		Entry("a transparent alpha", 0.0, uint8(0)),
		Entry("a faint alpha", 0.003, uint8(1)),
		Entry("a half alpha", 0.5, uint8(128)),
		Entry("an opaque alpha", 1.0, uint8(255)),
	)

	It("Should keep an absent icon color absent", func(ctx SpecContext) {
		Expect(migrate(ctx, v4.Panel{Key: uuid.New(), Root: root()}).Color).To(BeNil())
	})
})

var _ = Describe("MigratePanel", func() {
	It("Should store the icon color as a fixed color", func(ctx SpecContext) {
		out := MustSucceed(v5.MigratePanel(ctx, v4.Panel{
			Root: root(), Color: &color.Color{R: 1, A: 0.5},
		}))
		Expect(out.Color).To(Equal(fixed(colorv2.Color{R: 1, A: 128})))
	})
})

var _ = Describe("NameMigration", func() {
	migrated := func(ctx SpecContext, in v5.Panel) v5.Panel {
		GinkgoHelper()
		db := DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		Expect(gorp.NewCreate[v5.Key, v5.Panel]().Entry(&in).Exec(ctx, db)).
			To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "Panel",
			Migrations: []gorp.Migration{v5.NameMigration},
		})).To(Succeed())
		var out v5.Panel
		Expect(gorp.NewRetrieve[v5.Key, v5.Panel]().
			Where(gorp.MatchKeys[v5.Key, v5.Panel](in.Key)).
			Entry(&out).Exec(ctx, db)).To(Succeed())
		return out
	}
	key := uuid.MustParse("1a2b3c4d-0000-0000-0000-000000000000")
	DescribeTable(
		"Should name only nameless panels",
		func(ctx SpecContext, name, expected string) {
			in := v5.Panel{Key: key, Name: name, Root: root()}
			Expect(migrated(ctx, in).Name).To(Equal(expected))
		},
		Entry("empty name", "", "Panel 1a2b3c4d"),
		Entry("named panel", "Named", "Named"),
	)
})
