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
	v1 "github.com/synnaxlabs/synnax/pkg/service/arc/types/v1"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/vmihailenco/msgpack/v5"
)

var _ = Describe("StatusDetails", func() {
	Describe("DecodeMsgpack", func() {
		It("Should decode new lowercase msgpack fields", func() {
			original := v1.StatusDetails{Running: true}
			data := MustSucceed(msgpack.Marshal(original))
			var decoded v1.StatusDetails
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded.Running).To(BeTrue())
		})
		It("Should decode legacy uppercase Go field name", func() {
			legacy := struct{ Running bool }{Running: true}
			data := MustSucceed(msgpack.Marshal(legacy))
			var decoded v1.StatusDetails
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded.Running).To(BeTrue())
		})
		It("Should handle false value correctly for both formats", func() {
			original := v1.StatusDetails{Running: false}
			data := MustSucceed(msgpack.Marshal(original))
			var decoded v1.StatusDetails
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded.Running).To(BeFalse())
		})
	})
})
