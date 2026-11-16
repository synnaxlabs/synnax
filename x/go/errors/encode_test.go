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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
)

const myCustomErrorType string = "my-custom-error"

var (
	errMyCustomOne = errors.New("one")
	errMyCustomTwo = errors.New("two")
)

func encodeMyCustomError(_ *errors.Codec, err error) (errors.Payload, bool) {
	if errors.Is(err, errMyCustomOne) {
		return errors.Payload{
			Type: myCustomErrorType,
			Data: errMyCustomOne.Error(),
		}, true
	}
	if errors.Is(err, errMyCustomTwo) {
		return errors.Payload{
			Type: myCustomErrorType,
			Data: errMyCustomTwo.Error(),
		}, true
	}
	return errors.Payload{}, false
}

func decodeMyCustomError(_ *errors.Codec, encoded errors.Payload) (error, bool) {
	if encoded.Type != myCustomErrorType {
		return nil, false
	}
	switch encoded.Data {
	case errMyCustomOne.Error():
		return errMyCustomOne, true
	case errMyCustomTwo.Error():
		return errMyCustomTwo, true
	}
	panic("unknown error")
}

var _ = Describe("Codec", func() {
	var codec *errors.Codec
	BeforeEach(func() {
		ShouldNotLeakGoroutines()
		codec = errors.NewCodec(errors.Provider{
			Encode: encodeMyCustomError,
			Decode: decodeMyCustomError,
		})
	})
	Describe("Encode", func() {
		Context("Internal is true", func() {
			It(
				"Should encode a custom error type into a payload",
				func() {
					pld := codec.Encode(errMyCustomOne, true)
					Expect(pld.Type).To(Equal(myCustomErrorType))
					Expect(pld.Data).To(Equal(errMyCustomOne.Error()))
				},
			)
			It(
				"Should encode an unknown error using cockroachdb's errors package",
				func() {
					pld := codec.Encode(errors.New("unknown"), true)
					Expect(pld.Type).To(Equal(errors.TypeRoach))
				},
			)
		})
		Context("Internal is false", func() {
			It(
				"Should encode a custom error type into a payload",
				func() {
					pld := codec.Encode(errMyCustomOne, false)
					Expect(pld.Type).To(Equal(myCustomErrorType))
					Expect(pld.Data).To(Equal(errMyCustomOne.Error()))
				},
			)
			It(
				"Should encode an unknown error into a human readable string",
				func() {
					pld := codec.Encode(errors.New("unknown"), false)
					Expect(pld.Type).To(Equal(errors.TypeUnknown))
					Expect(pld.Data).To(Equal("unknown"))
				},
			)
			It("Should return a bare payload unchanged", func() {
				pld := errors.Payload{Type: "relayed", Data: "data"}
				Expect(codec.Encode(pld, false)).To(Equal(pld))
			})
			It("Should keep the wrap message of a wrapped payload", func() {
				pld := errors.Payload{Type: "relayed", Data: "data"}
				Expect(codec.Encode(errors.Wrap(pld, "ctx"), false)).To(Equal(
					errors.Payload{
						Type: errors.TypeUnknown,
						Data: "ctx: relayed---data",
					},
				))
			})
		})
	})
	Describe("Expected", func() {
		It("Should return true for a registered error", func() {
			Expect(
				codec.Expected(errors.Wrap(errMyCustomOne, "ctx")),
			).To(BeTrue())
		})
		It("Should return true for a join of registered errors", func() {
			Expect(
				codec.Expected(errors.Join(errMyCustomOne, errMyCustomTwo)),
			).To(BeTrue())
		})
		It(
			"Should return false for a join with an unknown error",
			func() {
				Expect(
					codec.Expected(errors.Join(errMyCustomOne, errors.New("x"))),
				).To(BeFalse())
			},
		)
		It("Should return false for an unknown error", func() {
			Expect(codec.Expected(errors.New("unknown"))).To(BeFalse())
		})
		It("Should return false for a nil error", func() {
			Expect(codec.Expected(nil)).To(BeFalse())
		})
	})
	Describe("Decode", func() {
		Context("Internal is true", func() {
			It(
				"Should decode a custom error type from a payload",
				func() {
					pld := codec.Encode(errMyCustomOne, true)
					Expect(codec.Decode(pld)).To(Equal(errMyCustomOne))
				},
			)
			It(
				"Should decode a nil error from a TypeNil typed payload",
				func() {
					pld := codec.Encode(nil, true)
					Expect(codec.Decode(pld)).To(Succeed())
				},
			)
			It(
				"Should decode an unknown error using cockroachdb's errors package",
				func() {
					pld := codec.Encode(errors.New("unknown"), true)
					Expect(codec.Decode(pld)).To(MatchError("unknown"))
				},
			)
		})
		Context("Internal is false", func() {
			It(
				"Should decode a custom error type from a payload",
				func() {
					pld := codec.Encode(errMyCustomOne, false)
					Expect(codec.Decode(pld)).To(Equal(errMyCustomOne))
				},
			)
			It(
				"Should decode a nil error from a TypeNil typed payload",
				func() {
					pld := codec.Encode(nil, false)
					Expect(codec.Decode(pld)).To(Succeed())
				},
			)
			It(
				"Should decode an unknown error into a human readable string",
				func() {
					pld := codec.Encode(errors.New("unknown"), false)
					Expect(codec.Decode(pld)).To(MatchError("unknown"))
				},
			)
			It(
				"Should decode an error with no type into a human readable string",
				func() {
					Expect(codec.Decode(errors.Payload{Data: "cat"})).
						To(MatchError(ContainSubstring("cat")))
				},
			)
			It(
				"Should decode a completely empty payload into a nil error",
				func() {
					Expect(codec.Decode(errors.Payload{})).To(Succeed())
				},
			)
		})
	})

	Describe("ParsePayload", func() {
		It("Should parse the text of Payload.Error", func() {
			p := errors.Payload{Type: "sy.validation", Data: "bad field"}
			Expect(errors.ParsePayload(p.Error())).To(Equal(p))
		})

		It("Should parse text in any other form as an unknown payload", func() {
			Expect(errors.ParsePayload("connection reset")).To(Equal(errors.Payload{
				Type: errors.TypeUnknown,
				Data: "connection reset",
			}))
		})

		It("Should parse text with more than one separator as unknown", func() {
			Expect(errors.ParsePayload("a---b---c")).To(Equal(errors.Payload{
				Type: errors.TypeUnknown,
				Data: "a---b---c",
			}))
		})
	})
})
