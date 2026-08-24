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
	v0 "github.com/synnaxlabs/arc/ir/versions/v0"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/vmihailenco/msgpack/v5"
)

var _ = Describe("Handle", func() {
	Describe("String", func() {
		It("Should format the handle as node.param", func() {
			h := v0.Handle{Node: "sensor", Param: "output"}
			Expect(h.String()).To(Equal("sensor.output"))
		})
	})

	Describe("DecodeMsgpack", func() {
		It("Should decode new lowercase msgpack fields", func() {
			original := v0.Handle{Node: "node1", Param: "input"}
			data := MustSucceed(msgpack.Marshal(original))
			var decoded v0.Handle
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded).To(Equal(original))
		})
		It("Should decode legacy uppercase Go field names", func() {
			legacy := struct {
				Node  string
				Param string
			}{Node: "node1", Param: "input"}
			data := MustSucceed(msgpack.Marshal(legacy))
			var decoded v0.Handle
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded.Node).To(Equal("node1"))
			Expect(decoded.Param).To(Equal("input"))
		})
	})
})
