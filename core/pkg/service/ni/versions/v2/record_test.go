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
	v2 "github.com/synnaxlabs/synnax/pkg/service/ni/versions/v2"
)

var _ = Describe("BaseAIChannel", func() {
	Describe("AIBase", func() {
		It("Should return the shared fields of an analog input variant", func() {
			base := v2.BaseAIChannel{Key: "ai", Name: "Voltage", Port: 3}
			ch := v2.AIVoltageChannel{BaseAIChannel: base}
			Expect(ch.AIBase()).To(Equal(base))
		})
	})
})
