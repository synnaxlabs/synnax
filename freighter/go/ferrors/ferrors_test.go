// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ferrors_test

import (
	"context"
	"github.com/cockroachdb/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter/ferrors"
	. "github.com/synnaxlabs/x/testutil"
)

const (
	MyCustomErrorType     ferrors.Type = "my-custom-error"
	UnregisteredErrorType ferrors.Type = "unregistered-error"
)

var (
	MyCustomErrorOne = ferrors.Typed(errors.New("one"), MyCustomErrorType)
	MyCustomErrorTwo = ferrors.Typed(errors.New("two"), MyCustomErrorType)
)

func encodeMyCustomError(ctx context.Context, err error) (ferrors.Payload, bool) {
	v, ok := err.(ferrors.Error)
	if !ok || v.FreighterType() != MyCustomErrorType {
		return ferrors.Payload{}, false
	}
	return ferrors.Payload{
		Type: MyCustomErrorType,
		Data: v.Error(),
	}, true
}

func decodeMyCustomError(ctx context.Context, encoded ferrors.Payload) (error, bool) {
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
		ferrors.Register(encodeMyCustomError, decodeMyCustomError)
	})
	Describe("Encode", func() {
		It("Should encode a custom error type into a payload", func() {
			pld := ferrors.Encode(MyCustomErrorOne)
			Expect(pld.Type).To(Equal(MyCustomErrorType))
			Expect(pld.Data).To(Equal(MyCustomErrorOne.Error()))
		})
		It("Should encode an unknown error using cockroachdb's errors package", func() {
			pld := ferrors.Encode(errors.New("unknown"))
			Expect(pld.Type).To(Equal(ferrors.TypeRoach))
		})
		It("Should return an unknown error type if the error type is not registered", func() {
			pld := ferrors.Encode(ferrors.Typed(errors.New("unknown"), UnregisteredErrorType))
			Expect(pld.Type).To(Equal(ferrors.TypeUnknown))
		})
	})
	Describe("Decode", func() {
		It("Should decode a custom error type from a payload", func() {
			pld := ferrors.Encode(MyCustomErrorOne)
			err := ferrors.Decode(pld)
			Expect(err).To(Equal(MyCustomErrorOne))
		})
		It("Should decode a nil error from a TypeNil typed payload", func() {
			pld := ferrors.Encode(nil)
			err := ferrors.Decode(pld)
			Expect(err).To(BeNil())
		})
		It("Should decode an unknown error using cockroachdb's errors package", func() {
			pld := ferrors.Encode(errors.New("unknown"))
			pld2 := &ferrors.Payload{}
			pld2.Unmarshal(pld.Error())
			err := ferrors.Decode(*pld2)
			Expect(err).To(HaveOccurredAs(errors.New("unknown")))
		})
	})
})
