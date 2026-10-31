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
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	v1 "github.com/synnaxlabs/synnax/pkg/service/status/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/status/versions/v2"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Migration", func() {
	It(
		"Should keep each status's fields and leave since and cleared empty",
		func(ctx SpecContext) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			MustOpen(gorp.OpenTable(
				ctx,
				gorp.TableConfig[v1.Key, v1.Status[any]]{DB: db},
			))
			old := v1.Status[any]{
				Key:         "rack-1",
				Name:        "Rack 1",
				Variant:     v1.VariantWarning,
				Message:     "Driver not running",
				Description: "The Core has not heard from the Driver on Rack 1.",
				Time:        telem.Now(),
				Details:     map[string]any{"rack": float64(1)},
			}
			Expect(gorp.NewCreate[v1.Key, v1.Status[any]]().
				Entry(&old).Exec(ctx, db)).To(Succeed())
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:         db,
				Namespace:  "Status",
				Migrations: []gorp.Migration{v2.Migration},
			})).To(Succeed())
			var got v2.Status[any]
			Expect(gorp.NewRetrieve[v2.Key, v2.Status[any]]().
				Where(gorp.MatchKeys[v2.Key, v2.Status[any]](old.Key)).
				Entry(&got).Exec(ctx, db)).To(Succeed())
			Expect(got).To(Equal(v2.Status[any]{
				Key:         old.Key,
				Name:        old.Name,
				Variant:     old.Variant,
				Message:     old.Message,
				Description: old.Description,
				Time:        old.Time,
				Details:     old.Details,
			}))
		},
	)

	It("Should keep details integers past 2^53", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		details := json.RawMessage(`{"count":9007199254740993}`)
		old := v1.Status[json.RawMessage]{
			Key:     "rack-1",
			Variant: v1.VariantInfo,
			Message: "Message",
			Details: details,
		}
		Expect(gorp.NewCreate[v1.Key, v1.Status[json.RawMessage]]().
			Entry(&old).Exec(ctx, db)).To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "Status",
			Migrations: []gorp.Migration{v2.Migration},
		})).To(Succeed())
		var got v2.Status[json.RawMessage]
		Expect(gorp.NewRetrieve[v2.Key, v2.Status[json.RawMessage]]().
			Where(gorp.MatchKeys[v2.Key, v2.Status[json.RawMessage]](old.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got.Details).To(Equal(details))
	})
})
