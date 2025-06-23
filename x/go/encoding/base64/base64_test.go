// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package base64_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/encoding/base64"
)

var _ = Describe(
	"Base64", func() {
		Describe(
			"MustDecode", func() {
				Context(
					"Valid Base64", func() {
						DescribeTable(
							"it should decode the string",
							func(encoded, decoded string) {
								Expect(base64.MustDecode(encoded)).To(Equal(decoded))
							}, Entry("empty", "", ""),
							Entry("simple", "aGVsbG8=", "hello"),
							Entry("complex", "dGVzdA==", "test"),
							Entry(
								"with emojis and padding", "IPCfp5F0a8+JIA==", " 🧑tkω ",
							),
						)
					},
				)
				Context(
					"invalid base64a", func() {
						DescribeTable(
							"it should panic when it can't decode",
							func(encoded string) {
								Expect(func() { base64.MustDecode(encoded) }).To(Panic())
							},
							Entry("with space", " "),
							Entry("with invalid characters", "!"),
							Entry("with invalid padding", "abc"),
						)
					},
				)
			},
		)
	},
)
