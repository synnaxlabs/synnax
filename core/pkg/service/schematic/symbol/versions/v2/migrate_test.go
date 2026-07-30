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
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v2"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Migration", func() {
	// seedV1 writes a symbol in the v1 storage shape, version field included.
	seedV1 := func(ctx SpecContext, db *gorp.DB, s v1.Symbol) v1.Symbol {
		t := MustOpen(gorp.OpenTable(
			ctx, gorp.TableConfig[v1.Key, v1.Symbol]{DB: db},
		))
		Expect(t.NewCreate().Entry(&s).Exec(ctx, db)).To(Succeed())
		return s
	}

	It("Should drop the persisted version field on retrieve", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		seed := seedV1(ctx, db, v1.Symbol{
			Key:     uuid.New(),
			Version: 1,
			Name:    "pump",
			Data: v1.Spec{
				SVG: "<svg>x</svg>", Variant: "valve", Scale: 2, ScaleStroke: true,
			},
		})

		t := MustOpen(gorp.OpenTable(
			ctx, gorp.TableConfig[v2.Key, v2.Symbol]{
				DB:         db,
				Migrations: []migrate.Migration{v2.Migration},
			},
		))
		var got v2.Symbol
		Expect(t.NewRetrieve().
			Where(gorp.MatchKeys[v2.Key, v2.Symbol](seed.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())

		Expect(got.Key).To(Equal(seed.Key))
		Expect(got.Name).To(Equal("pump"))
		Expect(got.Data.SVG).To(Equal("<svg>x</svg>"))
		Expect(got.Data.Variant).To(Equal("valve"))
		Expect(got.Data.Scale).To(Equal(2.0))
		Expect(got.Data.ScaleStroke).To(BeTrue())
	})
})
