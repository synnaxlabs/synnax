// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package noop_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/noop"
)

var _ = Describe("Noop", func() {
	Describe("UnaryServer", func() {
		var server noop.UnaryServer[string, string]

		BeforeEach(func() {
			server = noop.UnaryServer[string, string]{}
		})

		It("should satisfy the freighter.UnaryServer interface", func() {
			var _ freighter.UnaryServer[string, string] = &server
		})

		It("should accept middleware without panicking", func() {
			Expect(func() {
				server.Use(freighter.MiddlewareFunc(func(
					ctx freighter.Context,
					next freighter.Next,
				) (freighter.Context, error) {
					return next(ctx)
				}))
			}).ToNot(Panic())
		})

		It("should accept multiple middleware without panicking", func() {
			mw := freighter.MiddlewareFunc(func(
				ctx freighter.Context,
				next freighter.Next,
			) (freighter.Context, error) {
				return next(ctx)
			})
			Expect(func() { server.Use(mw, mw, mw) }).ToNot(Panic())
		})

		It("should accept a handler without panicking", func() {
			Expect(func() {
				server.BindHandler(func(ctx context.Context, req string) (string, error) {
					return req, nil
				})
			}).ToNot(Panic())
		})

		It("should return a report with empty protocol and encodings", func() {
			report := server.Report()
			Expect(report).To(HaveKeyWithValue("protocol", ""))
			Expect(report).To(HaveKeyWithValue("encodings", BeNil()))
		})
	})

	Describe("StreamServer", func() {
		var server noop.StreamServer[string, string]

		BeforeEach(func() {
			server = noop.StreamServer[string, string]{}
		})

		It("should satisfy the freighter.StreamServer interface", func() {
			var _ freighter.StreamServer[string, string] = &server
		})

		It("should accept middleware without panicking", func() {
			Expect(func() {
				server.Use(freighter.MiddlewareFunc(func(
					ctx freighter.Context,
					next freighter.Next,
				) (freighter.Context, error) {
					return next(ctx)
				}))
			}).ToNot(Panic())
		})

		It("should accept multiple middleware without panicking", func() {
			mw := freighter.MiddlewareFunc(func(
				ctx freighter.Context,
				next freighter.Next,
			) (freighter.Context, error) {
				return next(ctx)
			})
			Expect(func() { server.Use(mw, mw, mw) }).ToNot(Panic())
		})

		It("should accept a handler without panicking", func() {
			Expect(func() {
				server.BindHandler(func(
					ctx context.Context,
					stream freighter.ServerStream[string, string],
				) error {
					return nil
				})
			}).ToNot(Panic())
		})

		It("should return a report with empty protocol and encodings", func() {
			report := server.Report()
			Expect(report).To(HaveKeyWithValue("protocol", ""))
			Expect(report).To(HaveKeyWithValue("encodings", BeNil()))
		})
	})
})
