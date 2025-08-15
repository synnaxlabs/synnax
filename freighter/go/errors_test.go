// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package freighter_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/errors"
)

var _ = Describe("Errors", func() {
	var ctx context.Context
	BeforeEach(func() {
		ctx = context.Background()
	})
	Describe("Encoding", func() {
		It("Should encode EOF", func() {
			pld := errors.Encode(ctx, freighter.EOF, false)
			Expect(pld.Type).To(Equal("freighter.eof"))
			Expect(pld.Data).To(ContainSubstring("EOF"))
		})
		It("Should encode ErrStreamClosed", func() {
			pld := errors.Encode(ctx, freighter.ErrStreamClosed, false)
			Expect(pld.Type).To(Equal("freighter.stream_closed"))
			Expect(pld.Data).To(ContainSubstring("stream closed"))
		})
		It("Shouldn't encode unknown errors", func() {
			pld := errors.Encode(ctx, errors.New("unknown error"), false)
			Expect(pld.Type).To(Equal(errors.TypeUnknown))
			Expect(pld.Data).To(ContainSubstring("unknown error"))
		})
	})
	Describe("Decoding", func() {
		It("Should decode EOF", func() {
			pld := errors.Payload{Type: "freighter.eof"}
			err := errors.Decode(ctx, pld)
			Expect(err).To(MatchError(freighter.EOF))
			Expect(err.Error()).To(Equal("EOF"))
		})
		It("Should decode ErrStreamClosed", func() {
			pld := errors.Payload{Type: "freighter.stream_closed"}
			err := errors.Decode(ctx, pld)
			Expect(err).To(MatchError(freighter.ErrStreamClosed))
			Expect(err.Error()).To(ContainSubstring("stream closed"))
		})
		It("should decode unknown freighter error", func() {
			pld := errors.Payload{Type: "freighter.unknown", Data: "unknown error"}
			err := errors.Decode(ctx, pld)
			Expect(err.Error()).To(Equal("unknown error"))
		})
		It("shouldn't decode unknown error", func() {
			pld := errors.Payload{Type: "unknown", Data: "unknown error"}
			err := errors.Decode(ctx, pld)
			Expect(err.Error()).To(Equal("unknown error"))
		})
	})
})
