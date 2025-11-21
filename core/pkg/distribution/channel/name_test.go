// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Name Validation", func() {
	Describe("ValidateName", func() {
		DescribeTable("valid names", func(name string) {
			Expect(channel.ValidateName(name)).To(Succeed())
		},
			Entry("names with only letters", "temperature"),
			Entry("names with mixed case", "Pressure"),
			Entry("names with digits", "sensor1"),
			Entry("names with underscores", "sensor_temp"),
			Entry("names with letters and digits", "temp123"),
			Entry("names with letters and underscores", "Sensor_temp"),
			Entry("names with letters and digits and underscores", "temp123_sensor_temp"),
		)
		DescribeTable("invalid names", func(name string, errorMessage string) {
			err := channel.ValidateName(name)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(validate.Error))
			Expect(err).To(MatchError(ContainSubstring(errorMessage)))
		},
			Entry("empty name", "", "name cannot be empty"),
			Entry("name starting with digits", "1sensor", "cannot start with a digit"),
			Entry("name with spaces", "my channel", "contains invalid characters"),
			Entry("name with special characters", "sensor!", "contains invalid characters"),
			Entry("name with reserved keywords", "func", "is an Arc keyword and cannot be used"),
		)
	})
	DescribeTable("TransformName", func(name string, expected string) {
		Expect(channel.TransformName(name)).To(Equal(expected))
	},
		Entry("empty name", "", "channel"),
		Entry("valid name", "temperature", "temperature"),
		Entry("valid name with digits", "sensor1", "sensor1"),
		Entry("valid name with underscores", "sensor_temp", "sensor_temp"),
		Entry("valid name with mixed case", "Pressure", "Pressure"),
		Entry("valid name with mixed case and digits", "temp123", "temp123"),
		Entry("valid name with mixed case and underscores", "Sensor_temp", "Sensor_temp"),
		Entry("valid name with mixed case and digits and underscores", "temp123_sensor_temp", "temp123_sensor_temp"),
		Entry("name starting with digits", "1sensor", "_1sensor"),
		Entry("name with spaces", "my channel", "my_channel"),
		Entry("name with special characters", "sensor!", "sensor_"),
		Entry("name with reserved keywords", "func", "func_channel"),
		Entry("name with only invalid characters", "!!!", "channel"),
		Entry("leading whitespace", "  temperature", "temperature"),
		Entry("trailing whitespace", "temperature  ", "temperature"),
		Entry("leading and trailing whitespace", "  temperature  ", "temperature"),
	)
	Describe("GenerateUniqueName", func() {
		It("Should return base name if it doesn't exist", func() {
			Expect(channel.GenerateUniqueName("temperature", set.New("sensor1", "sensor2"))).
				To(Equal("temperature"))
		})

		It("Should append number if base name exists", func() {
			existingNames := set.Set[string]{"temperature": struct{}{}}
			Expect(channel.GenerateUniqueName("temperature", existingNames)).
				To(Equal("temperature_1"))
		})

		It("Should increment number until unique name is found", func() {
			existingNames := set.Set[string]{
				"temperature":   struct{}{},
				"temperature_1": struct{}{},
				"temperature_2": struct{}{},
			}
			Expect(channel.GenerateUniqueName("temperature", existingNames)).
				To(Equal("temperature_3"))
		})
		It("Should work with empty existing names set", func() {
			existingNames := set.Set[string]{}
			Expect(channel.GenerateUniqueName("temperature", existingNames)).
				To(Equal("temperature"))
		})
	})
})
