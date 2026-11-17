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
	"encoding/json/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v4 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v4"
	"github.com/synnaxlabs/x/encoding/orc"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("ReadField", func() {
	DescribeTable("Should keep a fill value that is a zero value",
		func(value any) {
			original := v4.ReadField{Optional: true, FillValue: &value}
			w := orc.NewWriter(0)
			Expect(original.EncodeOrc(w)).To(Succeed())
			r := orc.NewReader(nil)
			r.ResetBytes(w.Bytes())
			var decoded v4.ReadField
			Expect(decoded.DecodeOrc(r)).To(Succeed())
			Expect(decoded.FillValue).ToNot(BeNil())
			Expect(*decoded.FillValue).To(BeEquivalentTo(value))

			encoded := MustSucceed(json.Marshal(original))
			var parsed v4.ReadField
			Expect(json.Unmarshal(encoded, &parsed)).To(Succeed())
			Expect(parsed.FillValue).ToNot(BeNil())
			Expect(*parsed.FillValue).To(BeEquivalentTo(value))
		},
		Entry("a number", float64(0)),
		Entry("a string", ""),
		Entry("a bool", false),
	)
})

var _ = Describe("ReadConfig", func() {
	Describe("CustomTypeName", func() {
		It("Should return the HTTP read config gorp type name", func() {
			Expect(v4.ReadConfig{}.CustomTypeName()).To(Equal("http_read_config"))
		})
	})
})

var _ = Describe("WriteConfig", func() {
	Describe("CustomTypeName", func() {
		It("Should return the HTTP write config gorp type name", func() {
			Expect(v4.WriteConfig{}.CustomTypeName()).To(Equal("http_write_config"))
		})
	})
})
