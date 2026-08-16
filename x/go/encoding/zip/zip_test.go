// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package zip_test

import (
	"archive/zip"
	"bytes"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xzip "github.com/synnaxlabs/x/encoding/zip"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// write packs entries into an archive without the Encoder's validation, so specs can
// craft archives the Encoder refuses to produce.
func write(entries ...[2]string) []byte {
	GinkgoHelper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, entry := range entries {
		f := MustSucceed(zw.Create(entry[0]))
		if entry[1] != "" {
			MustSucceed(f.Write([]byte(entry[1])))
		}
	}
	Expect(zw.Close()).To(Succeed())
	return buf.Bytes()
}

// read unpacks b into the file map it was encoded from.
func read(b []byte) xzip.Files {
	GinkgoHelper()
	r := MustSucceed(zip.NewReader(bytes.NewReader(b), int64(len(b))))
	files := make(xzip.Files, len(r.File))
	for _, f := range r.File {
		rc := MustOpen(f.Open())
		files[f.Name] = MustSucceed(io.ReadAll(rc))
	}
	return files
}

var _ = Describe("Encoder", func() {
	Describe("ContentType", func() {
		It("Should report the zip content type", func() {
			Expect(xzip.Encoder.ContentType()).To(Equal("application/zip"))
		})
	})
	Describe("Encode", func() {
		It("Should encode every file into a readable archive", func(ctx SpecContext) {
			files := xzip.Files{
				"manifest.json": []byte(`{"version":2}`),
				"valve.json":    []byte(`{"name":"valve"}`),
			}
			Expect(read(MustSucceed(xzip.Encoder.Encode(ctx, files)))).To(Equal(files))
		})
		It("Should encode an empty file map into an empty archive", func(
			ctx SpecContext,
		) {
			Expect(
				read(MustSucceed(xzip.Encoder.Encode(ctx, xzip.Files{}))),
			).To(BeEmpty())
		})
		It("Should preserve an empty file's contents", func(ctx SpecContext) {
			b := MustSucceed(xzip.Encoder.Encode(ctx, xzip.Files{"empty.json": {}}))
			Expect(read(b)).To(HaveKeyWithValue("empty.json", BeEmpty()))
		})
		It("Should encode equal file maps to equal bytes", func(ctx SpecContext) {
			first := xzip.Files{"a.json": []byte("1"), "b.json": []byte("2")}
			second := xzip.Files{"b.json": []byte("2"), "a.json": []byte("1")}
			Expect(xzip.Encoder.Encode(ctx, first)).
				To(Equal(MustSucceed(xzip.Encoder.Encode(ctx, second))))
		})
		DescribeTable("Should reject a value that is not a file map",
			func(ctx SpecContext, value any) {
				Expect(xzip.Encoder.Encode(ctx, value)).Error().
					To(MatchError(ContainSubstring("failed to encode")))
			},
			Entry("a struct", struct{ Name string }{Name: "valve"}),
			Entry("a string map", map[string]string{"a.json": "1"}),
			Entry("a byte slice", []byte("valve")),
		)
		DescribeTable("Should reject a file name that is not a leaf",
			func(ctx SpecContext, name, reason string) {
				Expect(xzip.Encoder.Encode(ctx, xzip.Files{name: []byte("1")})).Error().
					To(SatisfyAll(
						MatchError(validate.ErrValidation),
						MatchError(ContainSubstring(reason)),
					))
			},
			Entry("an empty name", "", "file name is empty"),
			Entry("a forward slash", "nested/valve.json", "holds a path separator"),
			Entry("a backslash", `nested\valve.json`, "holds a path separator"),
			Entry("a leading slash", "/valve.json", "holds a path separator"),
			Entry("the current directory", ".", "addresses a directory"),
			Entry("the parent directory", "..", "addresses a directory"),
		)
	})
	Describe("EncodeStream", func() {
		It("Should write the archive to the writer", func(ctx SpecContext) {
			files := xzip.Files{"valve.json": []byte(`{"name":"valve"}`)}
			var buf bytes.Buffer
			Expect(xzip.Encoder.EncodeStream(ctx, &buf, files)).To(Succeed())
			Expect(read(buf.Bytes())).To(Equal(files))
		})
		It("Should reject a value that is not a file map", func(ctx SpecContext) {
			var buf bytes.Buffer
			Expect(xzip.Encoder.EncodeStream(ctx, &buf, "valve")).
				To(MatchError(ContainSubstring("failed to encode")))
		})
		It("Should write nothing when a file name is not a leaf", func(
			ctx SpecContext,
		) {
			files := xzip.Files{"a.json": []byte("1"), "nested/b.json": []byte("2")}
			var buf bytes.Buffer
			Expect(xzip.Encoder.EncodeStream(ctx, &buf, files)).
				To(MatchError(validate.ErrValidation))
			Expect(buf.Bytes()).To(BeEmpty())
		})
	})
})

var _ = Describe("Decoder", func() {
	Describe("ContentType", func() {
		It("Should report the zip content type", func() {
			Expect(xzip.Decoder.ContentType()).To(Equal("application/zip"))
		})
	})
	Describe("Decode", func() {
		It("Should decode every entry into the file map", func(ctx SpecContext) {
			files := xzip.Files{
				"manifest.json": []byte(`{"version":2}`),
				"valve.json":    []byte(`{"name":"valve"}`),
			}
			var decoded xzip.Files
			Expect(xzip.Decoder.Decode(
				ctx, MustSucceed(xzip.Encoder.Encode(ctx, files)), &decoded,
			)).To(Succeed())
			Expect(decoded).To(Equal(files))
		})
		It("Should decode an empty archive into an empty file map", func(
			ctx SpecContext,
		) {
			var decoded xzip.Files
			Expect(xzip.Decoder.Decode(
				ctx, MustSucceed(xzip.Encoder.Encode(ctx, xzip.Files{})), &decoded,
			)).To(Succeed())
			Expect(decoded).To(BeEmpty())
		})
		It("Should preserve an empty file's contents", func(ctx SpecContext) {
			var decoded xzip.Files
			Expect(xzip.Decoder.Decode(
				ctx,
				MustSucceed(xzip.Encoder.Encode(ctx, xzip.Files{"empty.json": {}})),
				&decoded,
			)).To(Succeed())
			Expect(decoded).To(HaveKeyWithValue("empty.json", BeEmpty()))
		})
		It("Should reject a value that is not a file map pointer", func(
			ctx SpecContext,
		) {
			b := write([2]string{"valve.json", "1"})
			var wrong map[string]string
			Expect(xzip.Decoder.Decode(ctx, b, &wrong)).
				To(MatchError(ContainSubstring("failed to decode")))
		})
		It("Should reject bytes that are not a zip archive", func(ctx SpecContext) {
			var decoded xzip.Files
			Expect(xzip.Decoder.Decode(ctx, []byte("not a zip"), &decoded)).
				To(MatchError(ContainSubstring("failed to decode")))
		})
		It("Should reject an archive that repeats an entry name", func(
			ctx SpecContext,
		) {
			b := write([2]string{"valve.json", "1"}, [2]string{"valve.json", "2"})
			var decoded xzip.Files
			Expect(xzip.Decoder.Decode(ctx, b, &decoded)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("repeats an earlier entry")),
			))
		})
		DescribeTable("Should reject an entry name that is not a leaf",
			func(ctx SpecContext, name, reason string) {
				var decoded xzip.Files
				Expect(xzip.Decoder.Decode(ctx, write([2]string{name, "1"}), &decoded)).
					To(SatisfyAll(
						MatchError(validate.ErrValidation),
						MatchError(ContainSubstring(reason)),
					))
			},
			Entry("an empty name", "", "file name is empty"),
			Entry("a nested entry", "nested/valve.json", "holds a path separator"),
			Entry("a backslash", `nested\valve.json`, "holds a path separator"),
			Entry("the current directory", ".", "addresses a directory"),
			Entry("the parent directory", "..", "addresses a directory"),
		)
		It("Should reject a directory entry", func(ctx SpecContext) {
			var decoded xzip.Files
			Expect(xzip.Decoder.Decode(ctx, write([2]string{"nested/", ""}), &decoded)).
				To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring("holds a path separator")),
				))
		})
	})
	Describe("DecodeStream", func() {
		It("Should decode the archive from the reader", func(ctx SpecContext) {
			files := xzip.Files{"valve.json": []byte(`{"name":"valve"}`)}
			var decoded xzip.Files
			Expect(xzip.Decoder.DecodeStream(
				ctx,
				bytes.NewReader(MustSucceed(xzip.Encoder.Encode(ctx, files))),
				&decoded,
			)).To(Succeed())
			Expect(decoded).To(Equal(files))
		})
	})
})
