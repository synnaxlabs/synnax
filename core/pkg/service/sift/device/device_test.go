// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package device_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/sift/device"
)

var _ = Describe("Device", func() {
	Describe("ParseProperties", func() {
		It("Should parse valid JSON properties", func() {
			Expect(device.ParseProperties(
				`{"api_key": "sk-test", "uri": "grpc-api.siftstack.com:443"}`,
			)).To(Equal(device.Properties{
				APIKey: "sk-test",
				URI:    "grpc-api.siftstack.com:443",
			}))
		})

		It("Should return error for invalid JSON", func() {
			Expect(device.ParseProperties("not json")).Error().
				To(MatchError(ContainSubstring("failed to parse device properties")))
		})

		It("Should handle empty JSON object", func() {
			Expect(device.ParseProperties("{}")).To(Equal(device.Properties{
				APIKey: "",
				URI:    "",
			}))
		})
	})
})
