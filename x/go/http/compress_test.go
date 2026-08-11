// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package http_test

import (
	"bytes"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xhttp "github.com/synnaxlabs/x/http"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Compress", func() {
	all := []xhttp.Compression{xhttp.Zstd, xhttp.Brotli, xhttp.Gzip, xhttp.Deflate}

	DescribeTable("ContentEncoding", func(c xhttp.Compression, expected string) {
		Expect(c.ContentEncoding()).To(Equal(expected))
	},
		Entry("gzip", xhttp.Gzip, "gzip"),
		Entry("brotli", xhttp.Brotli, "br"),
		Entry("zstd", xhttp.Zstd, "zstd"),
		Entry("deflate", xhttp.Deflate, "deflate"),
	)

	Describe("Round Trip", func() {
		DescribeTable("should return the original body", func(c xhttp.Compression) {
			body := []byte(strings.Repeat("synnax telemetry ", 500))
			Expect(c.Decompress(MustSucceed(c.Compress(body)), 0)).To(Equal(body))
		},
			Entry("gzip", xhttp.Gzip),
			Entry("brotli", xhttp.Brotli),
			Entry("zstd", xhttp.Zstd),
			Entry("deflate", xhttp.Deflate),
		)

		DescribeTable("should handle an empty body", func(c xhttp.Compression) {
			Expect(c.Decompress(MustSucceed(c.Compress(nil)), 0)).To(BeEmpty())
		},
			Entry("gzip", xhttp.Gzip),
			Entry("brotli", xhttp.Brotli),
			Entry("zstd", xhttp.Zstd),
			Entry("deflate", xhttp.Deflate),
		)

		// Concurrent use exercises the pooled writers and readers, where a leaked or
		// unreset instance would corrupt another caller's output.
		DescribeTable("should be safe under concurrent use", func(c xhttp.Compression) {
			body := []byte(strings.Repeat("concurrent ", 200))
			results := make(chan []byte, 16)
			for range cap(results) {
				go func() {
					defer GinkgoRecover()
					results <- MustSucceed(
						c.Decompress(MustSucceed(c.Compress(body)), 0),
					)
				}()
			}
			for range cap(results) {
				Eventually(results).Should(Receive(Equal(body)))
			}
		},
			Entry("gzip", xhttp.Gzip),
			Entry("brotli", xhttp.Brotli),
			Entry("zstd", xhttp.Zstd),
			Entry("deflate", xhttp.Deflate),
		)
	})

	Describe("Decompress", func() {
		DescribeTable(
			"should return ErrBodyTooLarge when the body expands past maxSize",
			func(c xhttp.Compression) {
				compressed := MustSucceed(
					c.Compress(bytes.Repeat([]byte("a"), 1_000_000)),
				)
				Expect(c.Decompress(compressed, 1024)).Error().To(SatisfyAll(
					MatchError(xhttp.ErrBodyTooLarge),
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring("1024")),
				))
			},
			Entry("gzip", xhttp.Gzip),
			Entry("brotli", xhttp.Brotli),
			Entry("zstd", xhttp.Zstd),
			Entry("deflate", xhttp.Deflate),
		)

		DescribeTable(
			"should accept a body sitting exactly on maxSize",
			func(c xhttp.Compression) {
				body := bytes.Repeat([]byte("a"), 1024)
				Expect(
					c.Decompress(MustSucceed(c.Compress(body)), 1024),
				).To(Equal(body))
			},
			Entry("gzip", xhttp.Gzip),
			Entry("brotli", xhttp.Brotli),
			Entry("zstd", xhttp.Zstd),
			Entry("deflate", xhttp.Deflate),
		)

		DescribeTable("should reject a body that is not valid under the encoding",
			func(c xhttp.Compression, message string) {
				Expect(c.Decompress([]byte("not compressed at all"), 0)).Error().
					To(MatchError(ContainSubstring(message)))
			},
			// Brotli is omitted: its format has no magic number, so a short arbitrary
			// byte string can decode as valid brotli rather than failing.
			Entry("gzip", xhttp.Gzip, "invalid header"),
			Entry("zstd", xhttp.Zstd, "magic number"),
			Entry("deflate", xhttp.Deflate, "invalid header"),
		)
	})

	Describe("ResolveCompression", func() {
		DescribeTable("should match the declared encoding",
			func(contentEncoding string, expected xhttp.Compression) {
				Expect(
					xhttp.ResolveCompression(contentEncoding, all),
				).To(Equal(expected))
			},
			Entry("gzip", "gzip", xhttp.Gzip),
			Entry("brotli", "br", xhttp.Brotli),
			Entry("zstd", "zstd", xhttp.Zstd),
			Entry("deflate", "deflate", xhttp.Deflate),
			Entry("uppercase", "GZIP", xhttp.Gzip),
			Entry("padded", "  br  ", xhttp.Brotli),
		)

		DescribeTable("should report an uncompressed body as no compression",
			func(contentEncoding string) {
				Expect(xhttp.ResolveCompression(contentEncoding, all)).To(BeNil())
			},
			Entry("empty", ""),
			Entry("identity", "identity"),
		)

		It("should return ErrUnsupportedContentEncoding for an unknown encoding",
			func() {
				Expect(xhttp.ResolveCompression("snappy", all)).Error().To(SatisfyAll(
					MatchError(xhttp.ErrUnsupportedContentEncoding),
					MatchError(ContainSubstring("snappy")),
				))
			})

		It("should return ErrUnsupportedContentEncoding when no options are offered",
			func() {
				Expect(xhttp.ResolveCompression("gzip", nil)).Error().
					To(MatchError(xhttp.ErrUnsupportedContentEncoding))
			})
	})

	Describe("NegotiateCompression", func() {
		DescribeTable("should pick the encoding the peer ranks highest",
			func(acceptEncoding string, expected xhttp.Compression) {
				Expect(MustBeOk(
					xhttp.NegotiateCompression(acceptEncoding, all),
				)).To(Equal(expected))
			},
			Entry("server preference wins a tie", "gzip, br, zstd", xhttp.Zstd),
			Entry("only offer available", "gzip", xhttp.Gzip),
			Entry("explicit quality wins", "zstd;q=0.1, gzip;q=0.9", xhttp.Gzip),
			Entry("wildcard matches anything", "*", xhttp.Zstd),
			Entry(
				"wildcard fills in for unlisted",
				"gzip;q=0.2, *;q=0.8",
				xhttp.Zstd,
			),
			Entry("malformed quality falls back to 1", "gzip;q=abc", xhttp.Gzip),
			Entry("padded tokens", " br ,  gzip ", xhttp.Brotli),
			Entry("uppercase tokens", "GZIP", xhttp.Gzip),
		)

		DescribeTable("should select nothing",
			func(acceptEncoding string, options []xhttp.Compression) {
				_, ok := xhttp.NegotiateCompression(acceptEncoding, options)
				Expect(ok).To(BeFalse())
			},
			Entry("absent header", "", all),
			Entry("whitespace header", "   ", all),
			Entry("identity only", "identity", all),
			Entry("no options offered", "gzip", nil),
			Entry(
				"every offer rejected",
				"gzip;q=0, br;q=0, zstd;q=0, deflate;q=0",
				all,
			),
			Entry("wildcard rejected", "*;q=0", all),
			Entry(
				"named offer rejected under an accepting wildcard",
				"zstd;q=0, br;q=0, gzip;q=0, deflate;q=0, *;q=0",
				all,
			),
		)

		It("should prefer an explicit quality over the wildcard", func() {
			Expect(MustBeOk(
				xhttp.NegotiateCompression("gzip;q=1.0, *;q=0", all),
			)).To(Equal(xhttp.Gzip))
		})
	})
})
