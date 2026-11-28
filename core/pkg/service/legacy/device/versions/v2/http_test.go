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

func healthCheck(headers ...any) msgpack.EncodedJSON {
	return msgpack.EncodedJSON{
		"health_check": map[string]any{"path": "/health", "headers": headers},
	}
}

func header(name, value string) map[string]any {
	return map[string]any{"name": name, "value": value}
}

var _ = Describe("HTTPBlankHeaderMigration", func() {
	DescribeTable("Should migrate the properties",
		func(ctx SpecContext, deviceMake string, stored, want msgpack.EncodedJSON) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			StartRun(db.Run)
			d := v2.Device{Key: "server", Make: deviceMake, Properties: stored}
			Expect(gorp.NewCreate[v2.Key, v2.Device]().Entry(&d).Exec(ctx, db)).
				To(Succeed())
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:         db,
				Namespace:  "Device",
				Migrations: []gorp.Migration{v2.HTTPBlankHeaderMigration},
			})).To(Succeed())
			var got v2.Device
			Expect(gorp.NewRetrieve[v2.Key, v2.Device]().
				Where(gorp.MatchKeys[v2.Key, v2.Device](d.Key)).
				Entry(&got).Exec(ctx, db)).To(Succeed())
			Expect(got.Properties).To(Equal(want))
		},
		Entry(
			"of a server with a header that has no value",
			"http",
			healthCheck(header("X-Key", ""), header("X-Trace", "on")),
			healthCheck(header("X-Trace", "on")),
		),
		Entry(
			"of a server with a header that has no name",
			"http",
			healthCheck(header("", "abc"), header("X-Trace", "on")),
			healthCheck(header("X-Trace", "on")),
		),
		Entry(
			"of a server whose headers all have names and values unchanged",
			"http",
			healthCheck(header("X-Key", "abc")),
			healthCheck(header("X-Key", "abc")),
		),
		Entry(
			"of a server with no health check unchanged",
			"http",
			msgpack.EncodedJSON{"secure": true},
			msgpack.EncodedJSON{"secure": true},
		),
		Entry(
			"of another device unchanged",
			"opc",
			healthCheck(header("X-Key", ""), header("", "abc")),
			healthCheck(header("X-Key", ""), header("", "abc")),
		),
	)
})
