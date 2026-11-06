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
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Errors", func() {
	Describe("PathedError", func() {
		It("Should add a path to an error", func() {
			base := errors.New("cat")
			pathed := validate.PathedError(base, "field")
			Expect(pathed).To(MatchError(ContainSubstring("field: cat")))
		})
		It("Should append nested paths correctly", func() {
			base := errors.New("cat")
			first := validate.PathedError(base, "first")
			parent := validate.PathedError(first, "parent")
			Expect(parent).To(MatchError(ContainSubstring("parent.first: cat")))
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
		It("Should return nil when the error is nil", func() {
			Expect(validate.PathedError(nil, "field")).To(Succeed())
		})

		Describe("Encoding + Decoding", func() {
			It("Should correctly encode and decode", func(ctx SpecContext) {
				base := errors.New("cat")
				pathed := validate.PathedError(base, "field")
				encoded := errors.Encode(ctx, pathed, false)
				decoded := errors.Decode(ctx, encoded)
				Expect(decoded).To(MatchError(ContainSubstring("field: cat")))
			})

			It("Should have the correct encoded representation", func(ctx SpecContext) {
				base := errors.New("cat")
				pathed := validate.PathedError(base, "field")
				encoded := errors.Encode(ctx, pathed, false)
				Expect(encoded.Type).To(Equal("sy.validation.path"))
				Expect(
					encoded.Data,
				).To(Equal("{\"error\":{\"type\":\"unknown\",\"data\":\"cat\"},\"path\":[\"field\"]}"))
			})

			It(
				"Should correctly encode and decode nested paths",
				func(ctx SpecContext) {
					base := errors.New("cat")
					first := validate.PathedError(base, "first")
					parent := validate.PathedError(first, "parent")
					encoded := errors.Encode(ctx, parent, false)
					decoded := errors.Decode(ctx, encoded)
					Expect(
						decoded,
					).To(MatchError(ContainSubstring("parent.first: cat")))
				},
			)
		})
	})

	Describe("Encoding + Decoding", func() {
		It("Should send the message of a wrap", func(ctx SpecContext) {
			err := errors.Wrap(validate.ErrValidation, "channel name already exists")
			encoded := errors.Encode(ctx, err, false)
			Expect(encoded).To(Equal(errors.Payload{
				Type: "sy.validation",
				Data: "channel name already exists",
			}))
			Expect(errors.Decode(ctx, encoded)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError("channel name already exists: validation error"),
			))
		})

		It("Should send the formatted message of a wrapf", func(ctx SpecContext) {
			err := errors.Wrapf(validate.ErrValidation, "rate of %d%% is too high", 50)
			encoded := errors.Encode(ctx, err, false)
			Expect(encoded.Data).To(Equal("rate of 50% is too high"))
			Expect(errors.Decode(ctx, encoded)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError("rate of 50% is too high: validation error"),
			))
		})

		It("Should send the message of a wrap of a wrap", func(ctx SpecContext) {
			err := errors.Wrap(errors.Wrap(validate.ErrValidation, "inner"), "outer")
			encoded := errors.Encode(ctx, err, false)
			Expect(encoded.Data).To(Equal("outer: inner"))
			Expect(errors.Decode(ctx, encoded)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError("outer: inner: validation error"),
			))
		})

		DescribeTable(
			"Should keep the message of a derived sentinel",
			func(ctx SpecContext, err error, msg string) {
				encoded := errors.Encode(ctx, err, false)
				Expect(encoded.Data).To(Equal(msg))
				Expect(errors.Decode(ctx, encoded)).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(msg+": validation error"),
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

		It("Should send the message of a path error", func(ctx SpecContext) {
			err := validate.PathedError(validate.ErrRequired, "name")
			encoded := errors.Encode(ctx, err, false)
			Expect(encoded).To(Equal(errors.Payload{
				Type: "sy.validation.path",
				Data: `{"error":{"type":"sy.validation","data":"required"},` +
					`"path":["name"]}`,
			}))
			Expect(errors.Decode(ctx, encoded)).To(SatisfyAll(
				BeAssignableToTypeOf(validate.PathError{}),
				MatchError("name: required: validation error"),
			))
		})

		It("Should send the message of each joined error", func(ctx SpecContext) {
			err := errors.Join(
				errors.Wrap(validate.ErrValidation, "first"),
				errors.Wrap(validate.ErrValidation, "second"),
			)
			encoded := errors.Encode(ctx, err, false)
			Expect(encoded.Data).To(Equal("first\nsecond"))
			Expect(errors.Decode(ctx, encoded)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError("first\nsecond: validation error"),
			))
		})

		It("Should send an empty message for the bare sentinel", func(ctx SpecContext) {
			encoded := errors.Encode(ctx, validate.ErrValidation, false)
			Expect(encoded.Data).To(BeEmpty())
			Expect(errors.Decode(ctx, encoded)).To(SatisfyAll(
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
