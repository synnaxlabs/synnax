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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/synnaxlabs/synnax/pkg/service/mqtt/versions/v1"
)

var _ = Describe("ReadConfig", func() {
	Describe("CustomTypeName", func() {
		It("Should return the MQTT read config gorp type name", func() {
			Expect(v1.ReadConfig{}.CustomTypeName()).To(Equal("mqtt_read_config"))
		})
	})
})

var _ = Describe("WriteConfig", func() {
	Describe("CustomTypeName", func() {
		It("Should return the MQTT write config gorp type name", func() {
			Expect(v1.WriteConfig{}.CustomTypeName()).To(Equal("mqtt_write_config"))
		})
	})
})

var _ = Describe("ScanConfig", func() {
	Describe("CustomTypeName", func() {
		It("Should return the MQTT scan config gorp type name", func() {
			Expect(v1.ScanConfig{}.CustomTypeName()).To(Equal("mqtt_scan_config"))
		})
	})
})

var _ = Describe("EdgeConfig", func() {
	Describe("CustomTypeName", func() {
		It("Should return the Sparkplug B edge config gorp type name", func() {
			Expect(v1.EdgeConfig{}.CustomTypeName()).
				To(Equal("mqtt_sparkplug_edge_config"))
		})
	})
})
