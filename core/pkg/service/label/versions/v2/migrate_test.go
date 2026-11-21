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
	servicecolor "github.com/synnaxlabs/synnax/pkg/service/color/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/label/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/label/versions/v2"
	color "github.com/synnaxlabs/x/color/versions/v2"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Migration", func() {
	It("Should store the color of a label that v0.62 stored as a fixed color",
		func(ctx SpecContext) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			StartRun(db.Run)
			MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v1.Key, v1.Label]{DB: db}))
			old := v1.Label{
				Key:   uuid.New(),
				Name:  "Critical",
				Color: color.Color{R: 1, G: 2, B: 3, A: 128},
			}
			Expect(gorp.NewCreate[v1.Key, v1.Label]().Entry(&old).Exec(ctx, db)).
				To(Succeed())
			MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v2.Key, v2.Label]{
				DB: db, Migrations: []gorp.Migration{v2.Migration},
			}))
			var got v2.Label
			Expect(gorp.NewRetrieve[v2.Key, v2.Label]().
				Where(gorp.MatchKeys[v2.Key, v2.Label](old.Key)).
				Entry(&got).Exec(ctx, db)).To(Succeed())
			Expect(got.Name).To(Equal("Critical"))
			Expect(got.Color).To(Equal(servicecolor.Ref{
				Variant: servicecolor.FixedRef{
					Value: color.Color{R: 1, G: 2, B: 3, A: 128},
				},
			}))
		},
	)
})
