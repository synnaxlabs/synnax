// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mqtt_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt"
	"github.com/synnaxlabs/x/types"
)

// A stored record lives under the key prefix of its type name, so a new name hides
// every record that a released Core stored.
var _ = Describe("Table names", func() {
	DescribeTable("Should keep the table name of each stored config",
		func(name, expected string) { Expect(name).To(Equal(expected)) },
		Entry("read", types.Name[mqtt.ReadConfig](), "mqtt_read_config"),
		Entry("write", types.Name[mqtt.WriteConfig](), "mqtt_write_config"),
		Entry("scan", types.Name[mqtt.ScanConfig](), "mqtt_scan_config"),
		Entry("edge", types.Name[mqtt.EdgeConfig](), "mqtt_sparkplug_edge_config"),
	)
})
