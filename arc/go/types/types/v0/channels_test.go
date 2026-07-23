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
	v0 "github.com/synnaxlabs/arc/types/types/v0"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/vmihailenco/msgpack/v5"
)

var _ = Describe("Channels", func() {
	Describe("Copy", func() {
		It("Should deep copy the read and write maps", func() {
			original := v0.Channels{
				Read:  map[uint32]string{1: "sensor"},
				Write: map[uint32]string{2: "actuator"},
			}
			copied := original.Copy()
			copied.Read[1] = "changed"
			copied.Write[2] = "changed"
			Expect(original.Read[1]).To(Equal("sensor"))
			Expect(original.Write[2]).To(Equal("actuator"))
		})
		It("Should replace nil maps with empty maps", func() {
			copied := v0.Channels{}.Copy()
			Expect(copied.Read).ToNot(BeNil())
			Expect(copied.Read).To(BeEmpty())
			Expect(copied.Write).ToNot(BeNil())
			Expect(copied.Write).To(BeEmpty())
		})
	})

	Describe("DecodeMsgpack", func() {
		It("Should decode new lowercase msgpack fields", func() {
			original := v0.Channels{
				Read:  map[uint32]string{1: "sensor"},
				Write: map[uint32]string{2: "actuator"},
			}
			data := MustSucceed(msgpack.Marshal(original))
			var decoded v0.Channels
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded.Read).To(Equal(map[uint32]string{1: "sensor"}))
			Expect(decoded.Write).To(Equal(map[uint32]string{2: "actuator"}))
		})
		It("Should decode legacy uppercase Go field names", func() {
			legacy := struct {
				Read  map[uint32]string
				Write map[uint32]string
			}{
				Read:  map[uint32]string{3: "temp"},
				Write: map[uint32]string{4: "valve"},
			}
			data := MustSucceed(msgpack.Marshal(legacy))
			var decoded v0.Channels
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded.Read).To(Equal(map[uint32]string{3: "temp"}))
			Expect(decoded.Write).To(Equal(map[uint32]string{4: "valve"}))
		})
	})
	Describe("String", func() {
		DescribeTable(
			"Rendering",
			func(channels v0.Channels, expected string) {
				Expect(channels.String()).To(Equal(expected))
			},
			Entry("empty channels", v0.Channels{}, "(none)"),
			Entry("reads only",
				v0.Channels{Read: map[uint32]string{1: "sensor", 2: "temp"}},
				"read [1: sensor, 2: temp]"),
			Entry("writes only",
				v0.Channels{Write: map[uint32]string{3: "valve"}},
				"write [3: valve]"),
			Entry("reads and writes",
				v0.Channels{
					Read:  map[uint32]string{1: "sensor"},
					Write: map[uint32]string{2: "valve"},
				},
				"read [1: sensor], write [2: valve]"),
		)
	})
})
