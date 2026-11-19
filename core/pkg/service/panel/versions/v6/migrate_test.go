// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v6_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	servicecolor "github.com/synnaxlabs/synnax/pkg/service/color/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/panel/versions/v1"
	v5 "github.com/synnaxlabs/synnax/pkg/service/panel/versions/v5"
	v6 "github.com/synnaxlabs/synnax/pkg/service/panel/versions/v6"
	color "github.com/synnaxlabs/x/color/versions/v2"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

func root() v1.Node { return v1.Node{Variant: v1.LeafNode{Tabs: []v1.Tab{}}} }

var _ = Describe("Migration", func() {
	migrate := func(ctx SpecContext, old v5.Panel) v6.Panel {
		GinkgoHelper()
		db := DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v5.Key, v5.Panel]{DB: db}))
		Expect(gorp.NewCreate[v5.Key, v5.Panel]().Entry(&old).Exec(ctx, db)).
			To(Succeed())
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v6.Key, v6.Panel]{
			DB: db, Migrations: []gorp.Migration{v6.Migration},
		}))
		var got v6.Panel
		Expect(gorp.NewRetrieve[v6.Key, v6.Panel]().
			Where(gorp.MatchKeys[v6.Key, v6.Panel](old.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		return got
	}

	It("Should store a panel that v0.62 stored with its icon color as a fixed color",
		func(ctx SpecContext) {
			icon := "valve"
			old := v5.Panel{
				Key:   uuid.New(),
				Name:  "p",
				Root:  root(),
				Icon:  &icon,
				Color: &color.Color{R: 1, G: 2, B: 3, A: 128},
			}
			got := migrate(ctx, old)
			Expect(got.Name).To(Equal("p"))
			Expect(got.Icon).To(HaveValue(Equal("valve")))
			Expect(got.Color).To(Equal(&servicecolor.Ref{
				Variant: servicecolor.FixedRef{
					Value: color.Color{R: 1, G: 2, B: 3, A: 128},
				},
			}))
		},
	)

	It("Should keep an absent icon color absent", func(ctx SpecContext) {
		Expect(migrate(ctx, v5.Panel{Key: uuid.New(), Name: "p", Root: root()}).Color).
			To(BeNil())
	})
})
