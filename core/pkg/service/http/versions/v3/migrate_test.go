// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	v2 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v3"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Migration", func() {
	It(
		"Should keep every read config field and leave NaN fill disabled",
		func(ctx SpecContext) {
			stored := v2.ReadConfig{
				Key:                uuid.New(),
				AutoStart:          true,
				DataSavingDisabled: true,
				Device:             "dev",
				Rate:               5,
				Endpoints: []v2.ReadEndpoint{{
					Key:    "ep",
					Path:   "/values",
					Fields: []v2.ReadField{{Key: "f", Channel: 7, Pointer: "/p"}},
				}},
			}
			db := DeferClose(gorp.Wrap(memkv.New()))
			MustSucceed(
				gorp.OpenTable(ctx, gorp.TableConfig[uuid.UUID, v2.ReadConfig]{DB: db}),
			)
			Expect(
				gorp.NewCreate[uuid.UUID, v2.ReadConfig]().Entry(&stored).Exec(ctx, db),
			).
				To(Succeed())
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:         db,
				Namespace:  "http_read_config",
				Migrations: []gorp.Migration{v3.Migration},
			})).To(Succeed())
			var got v3.ReadConfig
			Expect(gorp.NewRetrieve[uuid.UUID, v3.ReadConfig]().
				Where(gorp.MatchKeys[uuid.UUID, v3.ReadConfig](stored.Key)).
				Entry(&got).Exec(ctx, db)).To(Succeed())
			Expect(got.PersistConfig).To(Equal(stored.PersistConfig))
			Expect(got.Device).To(Equal(stored.Device))
			Expect(got.Rate).To(Equal(stored.Rate))
			Expect(got.Endpoints).To(Equal(stored.Endpoints))
			Expect(got.NanFillEnabled).To(BeFalse())
		},
	)
})
