// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/vmihailenco/msgpack/v5"
)

var _ = Describe("DecodeMsgpack", func() {
	Describe("Param", func() {
		It("Should decode new lowercase msgpack fields", func() {
			original := types.Param{
				Name: "rate",
				Type: types.Type{Kind: types.KindF64},
			}
			data := MustSucceed(msgpack.Marshal(original))
			var decoded types.Param
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded.Name).To(Equal("rate"))
			Expect(decoded.Type.Kind).To(Equal(types.KindF64))
		})
		It("Should decode legacy uppercase Go field names", func() {
			legacy := struct {
				Name  string
				Type  types.Type
				Value any
			}{
				Name:  "rate",
				Type:  types.Type{Kind: types.KindF32},
				Value: "100",
			}
			data := MustSucceed(msgpack.Marshal(legacy))
			var decoded types.Param
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded.Name).To(Equal("rate"))
			Expect(decoded.Type.Kind).To(Equal(types.KindF32))
		})
	})

	Describe("Channels", func() {
		It("Should decode new lowercase msgpack fields", func() {
			original := types.Channels{
				Read:  map[uint32]string{1: "sensor"},
				Write: map[uint32]string{2: "actuator"},
			}
			data := MustSucceed(msgpack.Marshal(original))
			var decoded types.Channels
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
			var decoded types.Channels
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded.Read).To(Equal(map[uint32]string{3: "temp"}))
			Expect(decoded.Write).To(Equal(map[uint32]string{4: "valve"}))
		})
	})
})
