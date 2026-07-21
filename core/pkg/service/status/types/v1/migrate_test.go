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
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	label "github.com/synnaxlabs/synnax/pkg/service/label/types/v0"
	v0 "github.com/synnaxlabs/synnax/pkg/service/status/types/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/status/types/v1"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("v1 -> current Status migration", func() {
	It("drops Labels from the wire and preserves core fields when v1 entries carry populated Labels", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))

		key := "status-" + uuid.NewString()
		seed := v0.Status[any]{
			Key:         key,
			Name:        "running",
			Variant:     v0.VariantSuccess,
			Message:     "task acquiring",
			Description: "5 channels",
			Time:        telem.Now(),
			Details:     map[string]any{"running": true},
			Labels: []label.Label{
				{Key: uuid.New(), Name: "critical", Color: color.Color{R: 255, A: 1}},
				{Key: uuid.New(), Name: "primary", Color: color.Color{B: 200, A: 1}},
			},
		}
		MustOpen(gorp.OpenTable(
			ctx, gorp.TableConfig[string, v0.Status[any]]{DB: db},
		))
		Expect(gorp.NewCreate[string, v0.Status[any]]().
			Entry(&seed).Exec(ctx, db)).To(Succeed())

		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "Status",
			Migrations: []migrate.Migration{v1.Migration},
		})).To(Succeed())

		var got v1.Status[any]
		Expect(gorp.NewRetrieve[string, v1.Status[any]]().
			Where(gorp.MatchKeys[string, v1.Status[any]](seed.Key)).Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got.Key).To(Equal(seed.Key))
		Expect(got.Name).To(Equal(seed.Name))
		Expect(got.Variant).To(Equal(v1.Variant(seed.Variant)))
		Expect(got.Message).To(Equal(seed.Message))
		Expect(got.Description).To(Equal(seed.Description))
		Expect(got.Labels).To(BeEmpty())
	})
})
