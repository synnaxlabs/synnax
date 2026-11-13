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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v3 "github.com/synnaxlabs/synnax/pkg/service/ni/versions/v3"
	v4 "github.com/synnaxlabs/synnax/pkg/service/ni/versions/v4"
)

var _ = Describe("CustomTypeName", func() {
	type named interface{ CustomTypeName() string }
	DescribeTable("Should keep the v3 table name of each config",
		func(cfg, stored named, want string) {
			Expect(cfg.CustomTypeName()).To(Equal(stored.CustomTypeName()))
			Expect(cfg.CustomTypeName()).To(Equal(want))
		},
		Entry(
			"analog read",
			v4.AnalogReadConfig{},
			v3.AnalogReadConfig{},
			"ni_analog_read_config",
		),
		Entry(
			"analog write",
			v4.AnalogWriteConfig{},
			v3.AnalogWriteConfig{},
			"ni_analog_write_config",
		),
		Entry(
			"counter read",
			v4.CounterReadConfig{},
			v3.CounterReadConfig{},
			"ni_counter_read_config",
		),
		Entry(
			"digital read",
			v4.DigitalReadConfig{},
			v3.DigitalReadConfig{},
			"ni_digital_read_config",
		),
		Entry(
			"digital write",
			v4.DigitalWriteConfig{},
			v3.DigitalWriteConfig{},
			"ni_digital_write_config",
		),
	)
})
