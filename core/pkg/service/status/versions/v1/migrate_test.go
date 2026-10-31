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
	"encoding/json"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	label "github.com/synnaxlabs/synnax/pkg/service/label/versions/v0"
	v0 "github.com/synnaxlabs/synnax/pkg/service/status/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/status/versions/v1"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("v0 -> v1 Status migration", func() {
	It(
		"drops Labels from the wire and preserves core fields when v0 entries carry populated Labels",
		func(ctx SpecContext) {
			db := DeferClose(gorp.Wrap(memkv.New()))

			key := "status-" + uuid.New().String()
			old := v0.Status[any]{
				Key:         key,
				Name:        "running",
				Variant:     v0.VariantSuccess,
				Message:     "task acquiring",
				Description: "5 channels",
				Time:        telem.Now(),
				Details:     map[string]any{"running": true},
				Labels: []label.Label{
					{
						Key:   uuid.New(),
						Name:  "critical",
						Color: color.Color{R: 255, A: 1},
					},
					{
						Key:   uuid.New(),
						Name:  "primary",
						Color: color.Color{B: 200, A: 1},
					},
				},
			}
			MustOpen(gorp.OpenTable(
				ctx, gorp.TableConfig[v0.Key, v0.Status[any]]{DB: db},
			))
			Expect(gorp.NewCreate[v0.Key, v0.Status[any]]().
				Entry(&old).Exec(ctx, db)).To(Succeed())

			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:         db,
				Namespace:  "Status",
				Migrations: []gorp.Migration{v1.Migration},
			})).To(Succeed())

			var got v1.Status[any]
			Expect(gorp.NewRetrieve[v1.Key, v1.Status[any]]().
				Where(gorp.MatchKeys[v1.Key, v1.Status[any]](old.Key)).
				Entry(&got).Exec(ctx, db)).To(Succeed())
			Expect(got.Key).To(Equal(old.Key))
			Expect(got.Name).To(Equal(old.Name))
			Expect(got.Variant).To(Equal(v1.Variant(old.Variant)))
			Expect(got.Message).To(Equal(old.Message))
			Expect(got.Description).To(Equal(old.Description))
			Expect(got.Labels).To(BeEmpty())
		},
	)

	It("Should keep details integers past 2^53", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		details := json.RawMessage(`{"count":9007199254740993}`)
		old := v0.Status[json.RawMessage]{
			Key:     "status-" + uuid.New().String(),
			Variant: v0.VariantInfo,
			Message: "Message",
			Details: details,
		}
		Expect(gorp.NewCreate[v0.Key, v0.Status[json.RawMessage]]().
			Entry(&old).Exec(ctx, db)).To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "Status",
			Migrations: []gorp.Migration{v1.Migration},
		})).To(Succeed())
		var got v1.Status[json.RawMessage]
		Expect(gorp.NewRetrieve[v1.Key, v1.Status[json.RawMessage]]().
			Where(gorp.MatchKeys[v1.Key, v1.Status[json.RawMessage]](old.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got.Details).To(Equal(details))
	})
})
