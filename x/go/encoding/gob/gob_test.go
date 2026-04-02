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
		codec := &gob.Codec{}
		b := MustSucceed(codec.Encode(ctx, toEncode{1}))
		var d toEncode
		Expect(codec.Decode(ctx, b, &d)).To(Succeed())
		Expect(d.Value).To(Equal(1))
		var d2 toEncode
		Expect(codec.DecodeStream(ctx, bytes.NewReader(b), &d2)).To(Succeed())
		Expect(d2.Value).To(Equal(1))
	})
	It("Should add error info on encoding failure", func(ctx SpecContext) {
		codec := &gob.Codec{}
		Expect(codec.Encode(ctx, make(chan int))).Error().To(MatchError(
			SatisfyAll(
				ContainSubstring("failed to encode value"),
				ContainSubstring("kind=chan, type=chan int"),
			),
		))
	})
	It("Should include a stack trace on encoding errors", func(ctx SpecContext) {
		codec := &gob.Codec{}
		_, err := codec.Encode(ctx, make(chan int))
		Expect(err).To(HaveOccurred())
		stack := errors.GetStackTrace(err)
		Expect(stack.String()).ToNot(BeEmpty())
		Expect(stack.String()).To(ContainSubstring(".go"))
	})
	It("Should include a stack trace on decoding errors", func(ctx SpecContext) {
		codec := &gob.Codec{}
		var d toEncode
		err := codec.Decode(ctx, []byte("invalid"), &d)
		Expect(err).To(HaveOccurred())
		stack := errors.GetStackTrace(err)
		Expect(stack.String()).ToNot(BeEmpty())
		Expect(stack.String()).To(ContainSubstring(".go"))
	})
})
