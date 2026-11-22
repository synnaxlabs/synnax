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
	"encoding/base64"
	"encoding/json/v2"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/synnax/pkg/service/bus/versions/v0"
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
	It("Should read and write a length size as its schema value", func() {
		var f v0.Framing
		Expect(json.Unmarshal(
			[]byte(`{"type": "sync", "sync": "qg==", "length_size": 4}`), &f,
		)).To(Succeed())
		sync, ok := f.Variant.(v0.SyncFraming)
		Expect(ok).To(BeTrue())
		Expect(sync.LengthSize).To(Equal(v0.LengthSizeFour))
		Expect(
			string(MustSucceed(json.Marshal(f))),
		).To(ContainSubstring(`"length_size":4`))
	})

	DescribeTable(
		"Should refuse an invalid framing with the path of its field",
		func(data, want string) {
			Expect(decodeValidate[v0.Framing](data)).To(SatisfyAll(
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
			"a length size of 3",
			`{"type": "sync", "sync": "qg==", "length_size": 3}`,
			"length_size: invalid length_size: LengthSize(3)",
		),
		Entry(
			"a max length of 0",
			`{"type": "sync", "sync": "qg==", "max_length": 0}`,
			"max_length: must be greater than or equal to 1",
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

// pollOf returns the JSON of a poll whose query holds size bytes.
func pollOf(size int) string {
	query := base64.StdEncoding.EncodeToString(make([]byte, size))
	return fmt.Sprintf(`{"query": %q, "rate": 1}`, query)
}

var _ = Describe("Poll", func() {
	It("Should accept a query of 65536 bytes", func() {
		Expect(decodeValidate[v0.Poll](pollOf(65536))).To(Succeed())
	})

	DescribeTable(
		"Should refuse an invalid poll with the path of its field",
		func(data, want string) {
			Expect(decodeValidate[v0.Poll](data)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(want)),
			))
		},
		Entry(
			"an empty query",
			`{"query": "", "rate": 1}`,
			"query: must be at least 1 bytes long",
		),
		Entry(
			"a query of 65537 bytes",
			pollOf(65537),
			"query: must be at most 65536 bytes long",
		),
		Entry(
			"a rate of 0",
			`{"query": "qg==", "rate": 0}`,
			"rate: must be greater than 0",
		),
	)
})
