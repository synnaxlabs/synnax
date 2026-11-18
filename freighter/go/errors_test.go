// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package freighter_test

import (
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/errors"
)

var _ = Describe("ErrorProvider", func() {
	codec := errors.NewCodec(freighter.ErrorProvider())
	DescribeTable("Should round-trip a freighter error",
		func(err error, errType string) {
			pld := codec.Encode(err, false)
			Expect(pld.Type).To(Equal(errType))
			Expect(codec.Decode(pld)).To(MatchError(err))
		},
		Entry("EOF", io.EOF, "freighter.eof"),
		Entry("stream closed", freighter.ErrStreamClosed, "freighter.stream_closed"),
	)
	It("Should decode another freighter type as an error with its data", func() {
		Expect(codec.Decode(errors.Payload{Type: "freighter.other", Data: "cat"})).
			To(MatchError("cat"))
	})
	It("Should leave other errors to other providers", func() {
		Expect(codec.Encode(errors.New("cat"), false).Type).
			To(Equal(errors.TypeUnknown))
	})
})
