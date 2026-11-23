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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	v2 "github.com/synnaxlabs/synnax/pkg/service/legacy/device/versions/v2"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("NIHardwareMigration", func() {
	DescribeTable("Should migrate the properties",
		func(ctx SpecContext, deviceMake string, stored, want msgpack.EncodedJSON) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			StartRun(db.Run)
			d := v2.Device{Key: "dev", Make: deviceMake, Properties: stored}
			Expect(gorp.NewCreate[v2.Key, v2.Device]().Entry(&d).Exec(ctx, db)).
				To(Succeed())
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:         db,
				Namespace:  "Device",
				Migrations: []gorp.Migration{v2.NIHardwareMigration},
			})).To(Succeed())
			var got v2.Device
			Expect(gorp.NewRetrieve[v2.Key, v2.Device]().
				Where(gorp.MatchKeys[v2.Key, v2.Device](d.Key)).
				Entry(&got).Exec(ctx, db)).To(Succeed())
			Expect(got.Properties).To(Equal(want))
		},
		Entry(
			"of an NI device with hardware",
			"NI",
			msgpack.EncodedJSON{
				"identifier": "dev1",
				"hardware":   map[string]any{"ai_min_rate": 0.0},
			},
			msgpack.EncodedJSON{"identifier": "dev1"},
		),
		Entry(
			"of an NI device without hardware unchanged",
			"NI",
			msgpack.EncodedJSON{"identifier": "dev1"},
			msgpack.EncodedJSON{"identifier": "dev1"},
		),
		Entry(
			"of another device unchanged",
			"LabJack",
			msgpack.EncodedJSON{"hardware": map[string]any{"rate": 0.0}},
			msgpack.EncodedJSON{"hardware": map[string]any{"rate": 0.0}},
		),
	)
})
