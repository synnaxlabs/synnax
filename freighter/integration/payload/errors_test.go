// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package payload_test

import (
	"context"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter/integration/payload"
	"github.com/synnaxlabs/x/errors"
)

var _ = Describe("Errors", func() {
	var ctx context.Context
	BeforeEach(func() {
		ctx = context.Background()
	})
	Describe("Encoding", func() {
		It("should encode errors", func() {
			err := payload.Error{Code: 1, Message: "test error"}
			pld := errors.Encode(ctx, err, false)
			Expect(pld.Type).To(Equal("integration.error"))
			Expect(pld.Data).To(Equal("1,test error"))
		})
		It("should not encode errors that are not of type integration.error", func() {
			err := errors.New("")
			pld := errors.Encode(ctx, err, false)
			Expect(pld.Type).To(Equal(errors.TypeUnknown))
			Expect(pld.Data).To(BeEmpty())
		})
	})
	Describe("Decoding", func() {
		It("should decode errors", func() {
			pld := errors.Payload{Type: "integration.error", Data: "1,test error"}
			err := errors.Decode(ctx, pld)
			Expect(err).To(BeAssignableToTypeOf(payload.Error{}))
			Expect(err.(payload.Error).Code).To(Equal(1))
			Expect(err.(payload.Error).Message).To(Equal("test error"))
		})
		It("should not decode errors that are not of type integration.error", func() {
			pld := errors.Payload{Type: "unknown", Data: "1,test error"}
			err := errors.Decode(ctx, pld)
			Expect(err).To(Not(BeAssignableToTypeOf(payload.Error{})))
		})
		It("should return an error if the data is not in the correct format", func() {
			pld := errors.Payload{
				Type: "integration.error",
				Data: "1,test error, with multiple parts",
			}
			err := errors.Decode(ctx, pld)
			Expect(err.Error()).To(Equal("unexpected error format"))
		})
		It("should return an error if the code is not a valid integer", func() {
			pld := errors.Payload{Type: "integration.error", Data: "invalid,test error"}
			err := errors.Decode(ctx, pld)
			Expect(err).To(BeAssignableToTypeOf(&strconv.NumError{}))
		})
	})
})
