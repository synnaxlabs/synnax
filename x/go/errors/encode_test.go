// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package errors_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
)

const MyCustomErrorType string = "my-custom-error"

var (
	MyCustomErrorOne = errors.New("one")
	MyCustomErrorTwo = errors.New("two")
)

func encodeMyCustomError(ctx context.Context, err error) (errors.Payload, bool) {
	if errors.Is(err, MyCustomErrorOne) {
		return errors.Payload{
			Type: MyCustomErrorType,
			Data: MyCustomErrorOne.Error(),
		}, true
	}
	if errors.Is(err, MyCustomErrorTwo) {
		return errors.Payload{
			Type: MyCustomErrorType,
			Data: MyCustomErrorTwo.Error(),
		}, true
	}
	return errors.Payload{}, false
}

func decodeMyCustomError(ctx context.Context, encoded errors.Payload) (error, bool) {
	if encoded.Type != MyCustomErrorType {
		return nil, false
	}
	switch encoded.Data {
	case MyCustomErrorOne.Error():
		return MyCustomErrorOne, true
	case MyCustomErrorTwo.Error():
		return MyCustomErrorTwo, true
	}
	panic("unknown error")
}

var _ = Describe("Ferrors", Ordered, func() {
	BeforeAll(func() {
		errors.Register(encodeMyCustomError, decodeMyCustomError)
	})
	Describe("Encode", func() {
		Context("Internal is true", func() {
			It("Should encode a custom error type into a payload", func() {
				pld := errors.Encode(ctx, MyCustomErrorOne, true)
				Expect(pld.Type).To(Equal(MyCustomErrorType))
				Expect(pld.Data).To(Equal(MyCustomErrorOne.Error()))
			})
			It("Should encode an unknown error using cockroachdb's errors package", func() {
				pld := errors.Encode(ctx, errors.New("unknown"), true)
				Expect(pld.Type).To(Equal(errors.TypeRoach))
			})
		})
		Context("Internal is false", func() {
			It("Should encode a custom error type into a payload", func() {
				pld := errors.Encode(ctx, MyCustomErrorOne, false)
				Expect(pld.Type).To(Equal(MyCustomErrorType))
				Expect(pld.Data).To(Equal(MyCustomErrorOne.Error()))
			})
			It("Should encode an unknown error into a human readable string", func() {
				pld := errors.Encode(ctx, errors.New("unknown"), false)
				Expect(pld.Type).To(Equal(errors.TypeUnknown))
				Expect(pld.Data).To(Equal("unknown"))
			})
		})

	})
	Describe("Decode", func() {
		Context("Internal is true", func() {
			It("Should decode a custom error type from a payload", func() {
				pld := errors.Encode(ctx, MyCustomErrorOne, true)
				err := errors.Decode(ctx, pld)
				Expect(err).To(Equal(MyCustomErrorOne))
			})
			It("Should decode a nil error from a TypeNil typed payload", func() {
				pld := errors.Encode(ctx, nil, true)
				err := errors.Decode(ctx, pld)
				Expect(err).To(BeNil())
			})
			It("Should decode an unknown error using cockroachdb's errors package", func() {
				pld := errors.Encode(ctx, errors.New("unknown"), true)
				pld2 := &errors.Payload{}
				pld2.Unmarshal(pld.Error())
				err := errors.Decode(ctx, *pld2)
				Expect(err).To(HaveOccurredAs(errors.New("unknown")))
			})
		})
		Context("Internal is false", func() {
			It("Should decode a custom error type from a payload", func() {
				pld := errors.Encode(ctx, MyCustomErrorOne, false)
				err := errors.Decode(ctx, pld)
				Expect(err).To(Equal(MyCustomErrorOne))
			})
			It("Should decode a nil error from a TypeNil typed payload", func() {
				pld := errors.Encode(ctx, nil, false)
				err := errors.Decode(ctx, pld)
				Expect(err).To(BeNil())
			})
			It("Should decode an unknown error into a human readable string", func() {
				pld := errors.Encode(ctx, errors.New("unknown"), false)
				err := errors.Decode(ctx, pld)
				pld2 := &errors.Payload{}
				pld2.Unmarshal(pld.Error())
				Expect(err).To(HaveOccurredAs(errors.New("unknown")))
			})
			It("Should decode an error with no type into a human readable string", func() {
				err := errors.Decode(ctx, errors.Payload{Data: "cat"})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("cat"))
			})
			It("Should decode a completely empty payload into a nil error", func() {
				err := errors.Decode(ctx, errors.Payload{})
				Expect(err).To(BeNil())
			})
		})
	})
})
