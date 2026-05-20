// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v54_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	colorv54 "github.com/synnaxlabs/x/color/migrations/v54"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	labelv54 "github.com/synnaxlabs/x/label/migrations/v54"
	"github.com/synnaxlabs/x/migrate"
	xstatus "github.com/synnaxlabs/x/status"
	v54 "github.com/synnaxlabs/x/status/migrations/v54"
	"github.com/synnaxlabs/x/telem"
	telemv54 "github.com/synnaxlabs/x/telem/migrations/v54"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("v54 -> current Status migration", func() {
	It("drops Labels from the wire and preserves core fields when v54 entries carry populated Labels", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))

		key := "status-" + uuid.NewString()
		seed := v54.Status[any]{
			Key:         key,
			Name:        "running",
			Variant:     v54.VariantSuccess,
			Message:     "task acquiring",
			Description: "5 channels",
			Time:        telemv54.TimeStamp(telem.Now()),
			Details:     map[string]any{"running": true},
			Labels: []labelv54.Label{
				{Key: uuid.New(), Name: "critical", Color: colorv54.Color{R: 255, A: 1}},
				{Key: uuid.New(), Name: "primary", Color: colorv54.Color{B: 200, A: 1}},
			},
		}
		MustSucceed(gorp.OpenTable[string, v54.Status[any]](
			ctx, gorp.TableConfig[string, v54.Status[any]]{DB: db},
		))
		Expect(gorp.NewCreate[string, v54.Status[any]]().
			Entry(&seed).Exec(ctx, db)).To(Succeed())

		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:        db,
			Namespace: "Status",
			Migrations: []migrate.Migration{
				gorp.NewEntryMigration[string, string, v54.Status[any], xstatus.Status[any]](
					"v54_drop_labels",
					xstatus.MigrateStatus[any],
				),
			},
		})).To(Succeed())

		var got xstatus.Status[any]
		Expect(gorp.NewRetrieve[string, xstatus.Status[any]]().
			Where(gorp.MatchKeys[string, xstatus.Status[any]](seed.Key)).Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got.Key).To(Equal(seed.Key))
		Expect(got.Name).To(Equal(seed.Name))
		Expect(got.Variant).To(Equal(xstatus.Variant(seed.Variant)))
		Expect(got.Message).To(Equal(seed.Message))
		Expect(got.Description).To(Equal(seed.Description))
		Expect(got.Labels).To(BeEmpty())
	})
})
