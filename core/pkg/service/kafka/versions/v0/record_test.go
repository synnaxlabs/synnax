// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/synnax/pkg/service/kafka/versions/v0"
)

var _ = Describe("Record", func() {
	DescribeTable("CustomTypeName",
		func(name, expected string) {
			Expect(name).To(Equal(expected))
		},
		Entry("read config", v0.ReadConfig{}.CustomTypeName(), "kafka_read_config"),
		Entry("write config", v0.WriteConfig{}.CustomTypeName(), "kafka_write_config"),
		Entry("scan config", v0.ScanConfig{}.CustomTypeName(), "kafka_scan_config"),
	)
})
