// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gob_test

import (
	"bytes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/encoding/gob"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
)

type toEncode struct {
	Value int
}

var _ = Describe("Codec", func() {
	It("Should encode and decode", func(ctx SpecContext) {
		b := MustSucceed(gob.Codec.Encode(toEncode{1}))
		var d toEncode
		Expect(gob.Codec.Decode(b, &d)).To(Succeed())
		Expect(d).To(Equal(toEncode{1}))
		var d2 toEncode
		Expect(gob.Codec.DecodeStream(bytes.NewReader(b), &d2)).To(Succeed())
		Expect(d2).To(Equal(toEncode{1}))
	})
	It("Should add error info on encoding failure", func(ctx SpecContext) {
		Expect(gob.Codec.Encode(make(chan int))).Error().To(MatchError(
			SatisfyAll(
				ContainSubstring("failed to encode value"),
				ContainSubstring("kind=chan, type=chan int"),
			)),
		)
	})
	It("Should include a stack trace on encoding errors", func(ctx SpecContext) {
		_, err := gob.Codec.Encode(make(chan int))
		Expect(err).To(HaveOccurred())
		stack := errors.GetStackTrace(err)
		Expect(stack.String()).ToNot(BeEmpty())
		Expect(stack.String()).To(ContainSubstring(".go"))
	})
	It("Should include a stack trace on decoding errors", func(ctx SpecContext) {
		err := gob.Codec.Decode([]byte("invalid"), &toEncode{})
		Expect(err).To(HaveOccurred())
		stack := errors.GetStackTrace(err)
		Expect(stack.String()).ToNot(BeEmpty())
		Expect(stack.String()).To(ContainSubstring(".go"))
	})
})
