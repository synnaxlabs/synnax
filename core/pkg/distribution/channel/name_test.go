// Copyright 2026 Synnax Labs, Inc.
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
		)
	})
	Describe("NewRandomName", func() {
		It("Should generate a random channel name that should be unique", func() {
			count := 100
			existingNames := make(set.Set[string], count)
			for range count {
				nextName := channel.NewRandomName()
				Expect(channel.ValidateName(nextName)).To(Succeed())
				Expect(existingNames.Contains(nextName)).To(BeFalse())
				existingNames.Add(nextName)
			}
		})
	})
})
