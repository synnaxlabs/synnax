// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package encoding_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/errors"
)

var errBase = errors.New("base failure")

var _ = Describe("SugarDecodingError", func() {
	It("Should return nil for a nil base error", func() {
		Expect(encoding.SugarDecodingError([]byte{1}, new(int), nil)).To(Succeed())
	})

	It("Should mark the error as a decode failure", func() {
		Expect(encoding.SugarDecodingError([]byte{1}, new(int), errBase)).To(
			MatchError(ContainSubstring("failed to decode")),
		)
	})

	It("Should retain the base error in the verbose format", func() {
		err := encoding.SugarDecodingError([]byte{1}, new(int), errBase)
		Expect(fmt.Sprintf("%+v", err)).To(ContainSubstring("base failure"))
	})

	It("Should describe the target and the raw data", func() {
		Expect(encoding.SugarDecodingError([]byte{0xAB, 0xCD}, new(int), errBase)).To(
			MatchError(ContainSubstring("kind=ptr, type=*int, data=abcd")),
		)
	})
})

var _ = Describe("SugarEncodingError", func() {
	It("Should return nil for a nil base error", func() {
		Expect(encoding.SugarEncodingError(1, nil)).To(Succeed())
	})

	It("Should mark the error as an encode failure", func() {
		Expect(encoding.SugarEncodingError(make(chan int), errBase)).To(
			MatchError(ContainSubstring("failed to encode")),
		)
	})

	It("Should retain the base error in the verbose format", func() {
		err := encoding.SugarEncodingError(make(chan int), errBase)
		Expect(fmt.Sprintf("%+v", err)).To(ContainSubstring("base failure"))
	})

	It("Should describe the value being encoded", func() {
		Expect(encoding.SugarEncodingError(make(chan int), errBase)).To(
			MatchError(ContainSubstring("kind=chan, type=chan int")),
		)
	})
})
