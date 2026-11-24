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
	v5 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v5"
	task "github.com/synnaxlabs/synnax/pkg/service/task/versions/v4"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Repair", func() {
	DescribeTable("Should repair the request of a stored read endpoint",
		func(in, expected v5.ReadEndpoint) {
			cfg := v5.ReadConfig{Endpoints: []v5.ReadEndpoint{in}}
			v5.RepairReadConfig(&cfg)
			Expect(cfg.Endpoints[0]).To(Equal(expected))
		},
		Entry(
			"rows with no name",
			v5.ReadEndpoint{
				Path:        "/now",
				Headers:     []v5.Header{{Value: "x"}, {Name: "Accept", Value: "j"}},
				QueryParams: []v5.QueryParam{{Value: "1"}},
			},
			v5.ReadEndpoint{
				Path:        "/now",
				Headers:     []v5.Header{{Name: "Accept", Value: "j"}},
				QueryParams: []v5.QueryParam{},
			},
		),
		Entry(
			"an unsorted, encoded query in the path",
			v5.ReadEndpoint{Path: "/now?units=si&city=New%20York"},
			v5.ReadEndpoint{
				Path: "/now",
				QueryParams: []v5.QueryParam{
					{Parameter: "units", Value: "si"},
					{Parameter: "city", Value: "New York"},
				},
			},
		),
		Entry(
			"a query in the path with a name and no value",
			v5.ReadEndpoint{Path: "/now?pretty"},
			v5.ReadEndpoint{Path: "/now?pretty"},
		),
		Entry(
			"a query in the path that repeats a name",
			v5.ReadEndpoint{Path: "/now?a=1&a=2"},
			v5.ReadEndpoint{Path: "/now?a=1&a=2"},
		),
	)

	It("Should repair each stored config on upgrade", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		key := task.KeyedConfig{Key: uuid.New()}
		read := v5.ReadConfig{
			KeyedConfig: key,
			Endpoints: []v5.ReadEndpoint{{
				Path:    "/now?units=si",
				Headers: []v5.Header{{Value: "x"}},
			}},
		}
		write := v5.WriteConfig{
			KeyedConfig: key,
			Endpoints: []v5.WriteEndpoint{{
				Path:        "/cmd?id=7",
				QueryParams: []v5.QueryParam{{Value: "1"}},
			}},
		}
		Expect(gorp.WrapWriter[uuid.UUID, v5.ReadConfig](db).Set(ctx, read)).
			To(Succeed())
		Expect(gorp.WrapWriter[uuid.UUID, v5.WriteConfig](db).Set(ctx, write)).
			To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "http_read_config",
			Migrations: []gorp.Migration{v5.ReadRepairMigration},
		})).To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "http_write_config",
			Migrations: []gorp.Migration{v5.WriteRepairMigration},
		})).To(Succeed())
		var gotRead v5.ReadConfig
		Expect(gorp.NewRetrieve[uuid.UUID, v5.ReadConfig]().
			Where(gorp.MatchKeys[uuid.UUID, v5.ReadConfig](key.Key)).
			Entry(&gotRead).
			Exec(ctx, db)).To(Succeed())
		Expect(gotRead.Endpoints[0].Path).To(Equal("/now"))
		Expect(gotRead.Endpoints[0].Headers).To(BeEmpty())
		Expect(gotRead.Endpoints[0].QueryParams).
			To(Equal([]v5.QueryParam{{Parameter: "units", Value: "si"}}))
		var gotWrite v5.WriteConfig
		Expect(gorp.NewRetrieve[uuid.UUID, v5.WriteConfig]().
			Where(gorp.MatchKeys[uuid.UUID, v5.WriteConfig](key.Key)).
			Entry(&gotWrite).
			Exec(ctx, db)).To(Succeed())
		Expect(gotWrite.Endpoints[0].Path).To(Equal("/cmd"))
		Expect(gotWrite.Endpoints[0].QueryParams).
			To(Equal([]v5.QueryParam{{Parameter: "id", Value: "7"}}))
	})
})
