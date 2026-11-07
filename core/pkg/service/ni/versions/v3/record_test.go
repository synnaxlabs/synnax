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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v2 "github.com/synnaxlabs/synnax/pkg/service/ni/versions/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/ni/versions/v3"
)

var _ = DescribeTable("CustomTypeName",
	func(got, stored, name string) {
		Expect(got).To(Equal(stored))
		Expect(got).To(Equal(name))
	},
	Entry("analog read",
		v3.AnalogReadConfig{}.CustomTypeName(),
		v2.AnalogReadConfig{}.CustomTypeName(),
		"ni_analog_read_config",
	),
	Entry("analog write",
		v3.AnalogWriteConfig{}.CustomTypeName(),
		v2.AnalogWriteConfig{}.CustomTypeName(),
		"ni_analog_write_config",
	),
	Entry("counter read",
		v3.CounterReadConfig{}.CustomTypeName(),
		v2.CounterReadConfig{}.CustomTypeName(),
		"ni_counter_read_config",
	),
	Entry("digital read",
		v3.DigitalReadConfig{}.CustomTypeName(),
		v2.DigitalReadConfig{}.CustomTypeName(),
		"ni_digital_read_config",
	),
	Entry("digital write",
		v3.DigitalWriteConfig{}.CustomTypeName(),
		v2.DigitalWriteConfig{}.CustomTypeName(),
		"ni_digital_write_config",
	),
)

var _ = Describe("ScanConfig", func() {
	Describe("CustomTypeName", func() {
		It("Should return the NI scanner config gorp type name", func() {
			Expect(v3.ScanConfig{}.CustomTypeName()).To(Equal("ni_scanner_config"))
		})
	})
})
