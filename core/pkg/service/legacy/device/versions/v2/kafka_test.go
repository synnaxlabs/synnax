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

var _ = Describe("KafkaSecureMigration", func() {
	DescribeTable("Should migrate the properties",
		func(ctx SpecContext, deviceMake string, stored, want msgpack.EncodedJSON) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			StartRun(db.Run)
			d := v2.Device{Key: "cluster", Make: deviceMake, Properties: stored}
			Expect(gorp.NewCreate[v2.Key, v2.Device]().Entry(&d).Exec(ctx, db)).
				To(Succeed())
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:         db,
				Namespace:  "Device",
				Migrations: []gorp.Migration{v2.KafkaSecureMigration},
			})).To(Succeed())
			var got v2.Device
			Expect(gorp.NewRetrieve[v2.Key, v2.Device]().
				Where(gorp.MatchKeys[v2.Key, v2.Device](d.Key)).
				Entry(&got).Exec(ctx, db)).To(Succeed())
			Expect(got.Properties).To(Equal(want))
		},
		Entry(
			"of a cluster that uses TLS",
			"kafka",
			msgpack.EncodedJSON{"brokers": []any{"b:9093"}, "tls": true},
			msgpack.EncodedJSON{"brokers": []any{"b:9093"}, "secure": true},
		),
		Entry(
			"of a cluster that does not use TLS",
			"kafka",
			msgpack.EncodedJSON{"brokers": []any{"b:9092"}, "tls": false},
			msgpack.EncodedJSON{"brokers": []any{"b:9092"}, "secure": false},
		),
		Entry(
			"of a cluster with no tls property unchanged",
			"kafka",
			msgpack.EncodedJSON{"brokers": []any{"b:9092"}},
			msgpack.EncodedJSON{"brokers": []any{"b:9092"}},
		),
		Entry(
			"of another device unchanged",
			"opc",
			msgpack.EncodedJSON{"tls": true},
			msgpack.EncodedJSON{"tls": true},
		),
	)
})
