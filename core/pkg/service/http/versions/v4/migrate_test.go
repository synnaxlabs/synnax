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
	v3 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v3"
	v4 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v4"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Migration", func() {
	It(
		"Should keep every read config field and leave fields not optional",
		func(ctx SpecContext) {
			stored := v3.ReadConfig{
				Key:            uuid.New(),
				Device:         "dev",
				Rate:           5,
				NanFillEnabled: true,
				Endpoints: []v3.ReadEndpoint{{
					Key:    "ep",
					Path:   "/values",
					Index:  "f",
					Fields: []v3.ReadField{{Key: "f", Channel: 7, Pointer: "/p"}},
				}},
			}
			db := DeferClose(gorp.Wrap(memkv.New()))
			MustSucceed(
				gorp.OpenTable(ctx, gorp.TableConfig[uuid.UUID, v3.ReadConfig]{DB: db}),
			)
			Expect(
				gorp.NewCreate[uuid.UUID, v3.ReadConfig]().Entry(&stored).Exec(ctx, db),
			).
				To(Succeed())
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:         db,
				Namespace:  "http_read_config",
				Migrations: []gorp.Migration{v4.Migration},
			})).To(Succeed())
			var got v4.ReadConfig
			Expect(gorp.NewRetrieve[uuid.UUID, v4.ReadConfig]().
				Where(gorp.MatchKeys[uuid.UUID, v4.ReadConfig](stored.Key)).
				Entry(&got).Exec(ctx, db)).To(Succeed())
			Expect(got.Device).To(Equal(stored.Device))
			Expect(got.NanFillEnabled).To(BeTrue())
			Expect(got.Endpoints).To(HaveLen(1))
			Expect(got.Endpoints[0].Index).To(Equal("f"))
			Expect(got.Endpoints[0].Fields).To(HaveLen(1))
			Expect(got.Endpoints[0].Fields[0].Channel).To(BeEquivalentTo(7))
			Expect(got.Endpoints[0].Fields[0].Pointer).To(Equal("/p"))
			Expect(got.Endpoints[0].Fields[0].Optional).To(BeFalse())
		},
	)
})
