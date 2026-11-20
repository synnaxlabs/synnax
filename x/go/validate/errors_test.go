// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package validate_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

type fieldError struct{ msg string }

func (e *fieldError) Error() string { return e.msg }

var _ = Describe("Errors", func() {
	Describe("PathedError", func() {
		It("Should add a path to an error", func() {
			base := errors.New("cat")
			Expect(
				validate.PathedError(base, "field"),
			).To(MatchError(ContainSubstring("field: cat")))
		})
		It("Should append nested paths correctly", func() {
			base := errors.New("cat")
			first := validate.PathedError(base, "first")
			Expect(
				validate.PathedError(first, "parent"),
			).To(MatchError(ContainSubstring("parent.first: cat")))
		})
		It("Should prefix every member of a joined error independently", func() {
			joined := errors.Join(
				validate.PathedError(errors.New("cat"), "level"),
				validate.PathedError(errors.New("dog"), "notation"),
			)
			parent := validate.PathedError(joined, "title")
			Expect(parent).To(MatchError(ContainSubstring("title.level: cat")))
			Expect(parent).To(MatchError(ContainSubstring("title.notation: dog")))
		})
		It("Should match the error at the path", func() {
			base := errors.Wrap(validate.ErrValidation, "cat")
			pathed := validate.PathedError(
				validate.PathedError(base, "first"),
				"parent",
			)
			Expect(pathed).To(MatchError(validate.ErrValidation))
			Expect(pathed).To(MatchError(base))
		})
		It("Should return nil when the error is nil", func() {
			Expect(validate.PathedError(nil, "field")).To(Succeed())
		})

		Describe("Encoding + Decoding", func() {
			codec := errors.NewCodec(query.ErrorProvider(), validate.ErrorProvider())
			It("Should correctly encode and decode", func() {
				base := errors.New("cat")
				pathed := validate.PathedError(base, "field")
				encoded := codec.Encode(pathed, false)
				Expect(
					codec.Decode(encoded),
				).To(MatchError(ContainSubstring("field: cat")))
			})

			It("Should have the correct encoded representation", func() {
				base := errors.New("cat")
				pathed := validate.PathedError(base, "field")
				encoded := codec.Encode(pathed, false)
				Expect(encoded.Type).To(Equal("sy.validation.path"))
				Expect(
					encoded.Data,
				).To(Equal("{\"error\":{\"type\":\"unknown\",\"data\":\"cat\"},\"path\":[\"field\"]}"))
			})

			It(
				"Should encode a path error around a query error as that error",
				func() {
					base := errors.Wrap(query.ErrNotFound, "cat")
					encoded := codec.Encode(
						validate.PathedError(base, "field"),
						false,
					)
					Expect(encoded.Type).To(Equal("sy.query.not_found"))
					Expect(
						codec.Decode(encoded),
					).To(MatchError(query.ErrNotFound))
				},
			)

			It(
				"Should correctly encode and decode nested paths",
				func() {
					base := errors.New("cat")
					first := validate.PathedError(base, "first")
					parent := validate.PathedError(first, "parent")
					encoded := codec.Encode(parent, false)
					Expect(
						codec.Decode(encoded),
					).To(MatchError(ContainSubstring("parent.first: cat")))
				},
			)
		})
	})

	Describe("Encoding + Decoding", func() {
		codec := errors.NewCodec(query.ErrorProvider(), validate.ErrorProvider())
		It("Should send the message of a wrap", func() {
			err := errors.Wrap(validate.ErrValidation, "channel name already exists")
			encoded := codec.Encode(err, false)
			Expect(encoded).To(Equal(errors.Payload{
				Type: "sy.validation",
				Data: "channel name already exists",
			}))
			Expect(codec.Decode(encoded)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError("channel name already exists"),
			))
		})

		It("Should send the formatted message of a wrapf", func() {
			err := errors.Wrapf(validate.ErrValidation, "rate of %d%% is too high", 50)
			encoded := codec.Encode(err, false)
			Expect(encoded.Data).To(Equal("rate of 50% is too high"))
			Expect(codec.Decode(encoded)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError("rate of 50% is too high"),
			))
		})

		It("Should send the message of a wrap of a wrap", func() {
			err := errors.Wrap(errors.Wrap(validate.ErrValidation, "inner"), "outer")
			encoded := codec.Encode(err, false)
			Expect(encoded.Data).To(Equal("outer: inner"))
			Expect(codec.Decode(encoded)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError("outer: inner"),
			))
		})

		DescribeTable(
			"Should keep the message of a derived sentinel",
			func(err error, msg string) {
				encoded := codec.Encode(err, false)
				Expect(encoded.Data).To(Equal(msg))
				Expect(codec.Decode(encoded)).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(msg),
				))
			},
			Entry(
				"required",
				errors.Wrap(validate.ErrRequired, "name"),
				"name: required",
			),
			Entry(
				"conversion",
				errors.Wrap(validate.ErrConversion, "rate"),
				"rate: conversion error",
			),
			Entry(
				"invalid type",
				validate.NewInvalidTypeError("int", "string"),
				"expected int but received string: invalid type",
			),
		)

		It("Should send the message of a path error", func() {
			err := validate.PathedError(validate.ErrRequired, "name")
			encoded := codec.Encode(err, false)
			Expect(encoded).To(Equal(errors.Payload{
				Type: "sy.validation.path",
				Data: `{"error":{"type":"sy.validation","data":"required"},` +
					`"path":["name"]}`,
			}))
			Expect(codec.Decode(encoded)).To(SatisfyAll(
				BeAssignableToTypeOf(validate.PathError{}),
				MatchError("name: required"),
			))
		})

		It("Should send the message of each joined error", func() {
			err := errors.Join(
				errors.Wrap(validate.ErrValidation, "first"),
				errors.Wrap(validate.ErrValidation, "second"),
			)
			encoded := codec.Encode(err, false)
			Expect(encoded.Data).To(Equal("first\nsecond"))
			Expect(codec.Decode(encoded)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError("first\nsecond"),
			))
		})

		It("Should send a join with a typed nil member", func() {
			err := errors.Join(
				(*fieldError)(nil),
				errors.Wrap(validate.ErrValidation, "second"),
			)
			Expect(codec.Encode(err, false)).To(Equal(errors.Payload{
				Type: "sy.validation",
				Data: "<nil>\nsecond",
			}))
		})

		It("Should send the message of the bare sentinel", func() {
			encoded := codec.Encode(validate.ErrValidation, false)
			Expect(encoded.Data).To(Equal("validation error"))
			Expect(codec.Decode(encoded)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError("validation error"),
			))
		})
	})

	Describe("InvalidTypeError", func() {
		It("Should format the error message correctly", func() {
			err := validate.NewInvalidTypeError("cat", "dog")
			Expect(err).To(MatchError(validate.ErrInvalidType))
			Expect(
				err,
			).To(MatchError(ContainSubstring("expected cat but received dog")))
		})
	})
})
