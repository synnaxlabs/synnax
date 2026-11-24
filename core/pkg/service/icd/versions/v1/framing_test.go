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
	"encoding/json/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v1"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// decodeValidate decodes data into a T and returns the error of its Validate.
func decodeValidate[T interface{ Validate() error }](data string) error {
	GinkgoHelper()
	var v T
	Expect(json.Unmarshal([]byte(data), &v)).To(Succeed())
	return v.Validate()
}

var _ = Describe("Framing", func() {
	It("Should read and write the framing of an ICD", func() {
		var ic v1.ICD
		Expect(json.Unmarshal(
			[]byte(
				`{"name": "gps", "framing": {"type": "delimited", "delimiter": "\r\n"}}`,
			),
			&ic,
		)).To(Succeed())
		Expect(ic.Framing).ToNot(BeNil())
		Expect(ic.Framing.Variant).To(Equal(v1.DelimitedFraming{Delimiter: "\r\n"}))
		Expect(string(MustSucceed(json.Marshal(ic)))).To(ContainSubstring(`"framing":`))
	})

	It("Should leave out the framing of an ICD that has none", func() {
		Expect(string(MustSucceed(json.Marshal(v1.ICD{Name: "gps"})))).ToNot(
			ContainSubstring("framing"),
		)
	})

	It("Should read and write a length size as its schema value", func() {
		var f v1.Framing
		Expect(json.Unmarshal(
			[]byte(`{"type": "sync", "sync": "qg==", "length_size": "four"}`), &f,
		)).To(Succeed())
		sync, ok := f.Variant.(v1.SyncFraming)
		Expect(ok).To(BeTrue())
		Expect(sync.LengthSize).To(Equal(v1.LengthSizeFour))
		Expect(
			string(MustSucceed(json.Marshal(f))),
		).To(ContainSubstring(`"length_size":"four"`))
	})

	It("Should refuse an unknown framing type when it decodes", func() {
		var f v1.Framing
		Expect(json.Unmarshal([]byte(`{"type": "bogus"}`), &f)).To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(`Framing: unknown type "bogus"`)),
		))
	})

	It("Should refuse a framing with no variant", func() {
		Expect(v1.Framing{}.Validate()).To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("type: required")),
		))
	})

	DescribeTable(
		"Should refuse an invalid framing with the path of its field",
		func(data, want string) {
			Expect(decodeValidate[v1.Framing](data)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(want)),
			))
		},
		Entry(
			"an empty sync sequence",
			`{"type": "sync", "sync": ""}`,
			"sync: must be at least 1 bytes long",
		),
		Entry(
			"an unknown length size",
			`{"type": "sync", "sync": "qg==", "length_size": "three"}`,
			"length_size: invalid length_size: three",
		),
		Entry(
			"a max length of 0",
			`{"type": "sync", "sync": "qg==", "max_length": 0}`,
			"max_length: must be greater than or equal to 1",
		),
		Entry(
			"a max length of 65537",
			`{"type": "sync", "sync": "qg==", "max_length": 65537}`,
			"max_length: must be less than or equal to 65536",
		),
		Entry(
			"an unknown byte order",
			`{"type": "sync", "sync": "qg==", "byte_order": "middle_endian"}`,
			"byte_order: invalid byte_order: middle_endian",
		),
		Entry(
			"an unknown checksum byte order",
			`{"type": "sync", "sync": "qg==", "checksum": `+
				`{"algorithm": "crc32", "byte_order": "middle_endian"}}`,
			"checksum.byte_order: invalid byte_order: middle_endian",
		),
		Entry(
			"a checksum with no algorithm",
			`{"type": "sync", "sync": "qg==", "checksum": {}}`,
			"checksum.algorithm: invalid algorithm: ",
		),
		Entry(
			"an unknown checksum algorithm",
			`{"type": "sync", "sync": "qg==", "checksum": {"algorithm": "crc8"}}`,
			"checksum.algorithm: invalid algorithm: crc8",
		),
		Entry(
			"a fixed length of 0",
			`{"type": "fixed", "length": 0}`,
			"length: must be greater than or equal to 1",
		),
		Entry(
			"a fixed length of 65537",
			`{"type": "fixed", "length": 65537}`,
			"length: must be less than or equal to 65536",
		),
		Entry(
			"an empty delimiter",
			`{"type": "delimited", "delimiter": ""}`,
			"delimiter: required",
		),
	)
})
