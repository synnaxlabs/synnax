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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xzip "github.com/synnaxlabs/x/encoding/zip"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// maxDecodedSize mirrors the Codec's 64 MiB decompression cap.
const maxDecodedSize = 64 << 20

// write packs entries into an archive without the Codec's validation, so specs can
// craft archives the Codec refuses to produce.
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

var _ = Describe("Codec", func() {
	Describe("ContentType", func() {
		It("Should report the zip content type", func() {
			Expect(xzip.Codec.ContentType()).To(Equal("application/zip"))
		})
	})
	Describe("Encode", func() {
		It("Should encode every file into a readable archive", func(ctx SpecContext) {
			files := xzip.Files{
				"manifest.json": []byte(`{"version":2}`),
				"valve.json":    []byte(`{"name":"valve"}`),
			}
			Expect(read(MustSucceed(xzip.Codec.Encode(ctx, files)))).To(Equal(files))
		})
		It("Should encode an empty file map into an empty archive", func(
			ctx SpecContext,
		) {
			Expect(
				read(MustSucceed(xzip.Codec.Encode(ctx, xzip.Files{}))),
			).To(BeEmpty())
		})
		It("Should preserve an empty file's contents", func(ctx SpecContext) {
			b := MustSucceed(xzip.Codec.Encode(ctx, xzip.Files{"empty.json": {}}))
			Expect(read(b)).To(HaveKeyWithValue("empty.json", BeEmpty()))
		})
		It("Should encode equal file maps to equal bytes", func(ctx SpecContext) {
			first := xzip.Files{"a.json": []byte("1"), "b.json": []byte("2")}
			second := xzip.Files{"b.json": []byte("2"), "a.json": []byte("1")}
			Expect(xzip.Codec.Encode(ctx, first)).
				To(Equal(MustSucceed(xzip.Codec.Encode(ctx, second))))
		})
		DescribeTable("Should reject a value that is not a file map",
			func(ctx SpecContext, value any) {
				Expect(xzip.Codec.Encode(ctx, value)).Error().
					To(MatchError(ContainSubstring("failed to encode")))
			},
			Entry("a struct", struct{ Name string }{Name: "valve"}),
			Entry("a string map", map[string]string{"a.json": "1"}),
			Entry("a byte slice", []byte("valve")),
		)
		It("Should encode nested entry paths", func(ctx SpecContext) {
			files := xzip.Files{
				"manifest.json":                  []byte(`{"version":1}`),
				"propulsion/pressurization.json": []byte(`{"name":"press"}`),
				"propulsion/tanks/lox.json":      []byte(`{"name":"lox"}`),
			}
			Expect(read(MustSucceed(xzip.Codec.Encode(ctx, files)))).To(Equal(files))
		})
		DescribeTable("Should reject an invalid entry name",
			func(ctx SpecContext, name, reason string) {
				Expect(xzip.Codec.Encode(ctx, xzip.Files{name: []byte("1")})).Error().
					To(SatisfyAll(
						MatchError(validate.ErrValidation),
						MatchError(ContainSubstring(reason)),
					))
			},
			Entry("an empty name", "", "file name is empty"),
			Entry("a backslash", `nested\valve.json`, "holds a backslash"),
			Entry("a leading slash", "/valve.json", "not a valid relative path"),
			Entry("a trailing slash", "nested/", "not a valid relative path"),
			Entry("a doubled slash", "nested//valve.json", "not a valid relative path"),
			Entry("the current directory", ".", "not a valid relative path"),
			Entry("the parent directory", "..", "not a valid relative path"),
			Entry(
				"a parent segment",
				"nested/../valve.json",
				"not a valid relative path",
			),
			Entry("a current segment", "./valve.json", "not a valid relative path"),
		)
	})
	Describe("EncodeStream", func() {
		It("Should write the archive to the writer", func(ctx SpecContext) {
			files := xzip.Files{"valve.json": []byte(`{"name":"valve"}`)}
			var buf bytes.Buffer
			Expect(xzip.Codec.EncodeStream(ctx, &buf, files)).To(Succeed())
			Expect(read(buf.Bytes())).To(Equal(files))
		})
		It("Should reject a value that is not a file map", func(ctx SpecContext) {
			var buf bytes.Buffer
			Expect(xzip.Codec.EncodeStream(ctx, &buf, "valve")).
				To(MatchError(ContainSubstring("failed to encode")))
		})
		It("Should write nothing when an entry name is invalid", func(
			ctx SpecContext,
		) {
			files := xzip.Files{"a.json": []byte("1"), `nested\b.json`: []byte("2")}
			var buf bytes.Buffer
			Expect(xzip.Codec.EncodeStream(ctx, &buf, files)).
				To(MatchError(validate.ErrValidation))
			Expect(buf.Bytes()).To(BeEmpty())
		})
	})
	Describe("Decode", func() {
		It("Should decode every entry into the file map", func(ctx SpecContext) {
			files := xzip.Files{
				"manifest.json": []byte(`{"version":2}`),
				"valve.json":    []byte(`{"name":"valve"}`),
			}
			var decoded xzip.Files
			Expect(xzip.Codec.Decode(
				ctx, MustSucceed(xzip.Codec.Encode(ctx, files)), &decoded,
			)).To(Succeed())
			Expect(decoded).To(Equal(files))
		})
		It("Should decode an empty archive into an empty file map", func(
			ctx SpecContext,
		) {
			var decoded xzip.Files
			Expect(xzip.Codec.Decode(
				ctx, MustSucceed(xzip.Codec.Encode(ctx, xzip.Files{})), &decoded,
			)).To(Succeed())
			Expect(decoded).To(BeEmpty())
		})
		It("Should preserve an empty file's contents", func(ctx SpecContext) {
			var decoded xzip.Files
			Expect(xzip.Codec.Decode(
				ctx,
				MustSucceed(xzip.Codec.Encode(ctx, xzip.Files{"empty.json": {}})),
				&decoded,
			)).To(Succeed())
			Expect(decoded).To(HaveKeyWithValue("empty.json", BeEmpty()))
		})
		It("Should reject a value that is not a file map pointer", func(
			ctx SpecContext,
		) {
			b := write([2]string{"valve.json", "1"})
			var wrong map[string]string
			Expect(xzip.Codec.Decode(ctx, b, &wrong)).
				To(MatchError(ContainSubstring("failed to decode")))
		})
		It("Should reject bytes that are not a zip archive", func(ctx SpecContext) {
			var decoded xzip.Files
			Expect(xzip.Codec.Decode(ctx, []byte("not a zip"), &decoded)).
				To(MatchError(ContainSubstring("failed to decode")))
		})
		It("Should reject an archive that repeats an entry name", func(
			ctx SpecContext,
		) {
			b := write([2]string{"valve.json", "1"}, [2]string{"valve.json", "2"})
			var decoded xzip.Files
			Expect(xzip.Codec.Decode(ctx, b, &decoded)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("repeats an earlier entry")),
			))
		})
		DescribeTable("Should reject an invalid entry name",
			func(ctx SpecContext, name, reason string) {
				var decoded xzip.Files
				Expect(xzip.Codec.Decode(ctx, write([2]string{name, "1"}), &decoded)).
					To(SatisfyAll(
						MatchError(validate.ErrValidation),
						MatchError(ContainSubstring(reason)),
					))
			},
			Entry("an empty name", "", "file name is empty"),
			Entry("a backslash", `nested\valve.json`, "holds a backslash"),
			Entry("the parent directory", "..", "not a valid relative path"),
		)
		It("Should reject a parent segment beside a root entry", func(
			ctx SpecContext,
		) {
			b := write(
				[2]string{"nested/../valve.json", "1"},
				[2]string{"other.json", "2"},
			)
			var decoded xzip.Files
			Expect(xzip.Codec.Decode(ctx, b, &decoded)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("not a valid relative path")),
			))
		})
		It("Should decode a nested entry beside a root entry", func(ctx SpecContext) {
			b := write([2]string{"a.json", "1"}, [2]string{"nested/b.json", "2"})
			var decoded xzip.Files
			Expect(xzip.Codec.Decode(ctx, b, &decoded)).To(Succeed())
			Expect(decoded).To(Equal(xzip.Files{
				"a.json":        []byte("1"),
				"nested/b.json": []byte("2"),
			}))
		})
		It("Should unwrap a root directory all entries share", func(ctx SpecContext) {
			b := write(
				[2]string{"grp/", ""},
				[2]string{"grp/manifest.json", `{"version":2}`},
				[2]string{"grp/valve.json", `{"name":"valve"}`},
				[2]string{"__MACOSX/._grp", "junk"},
				[2]string{"__MACOSX/grp/._manifest.json", "junk"},
				[2]string{"grp/.DS_Store", "junk"},
			)
			var decoded xzip.Files
			Expect(xzip.Codec.Decode(ctx, b, &decoded)).To(Succeed())
			Expect(decoded).To(Equal(xzip.Files{
				"manifest.json": []byte(`{"version":2}`),
				"valve.json":    []byte(`{"name":"valve"}`),
			}))
		})
		It("Should unwrap nested shared root directories", func(ctx SpecContext) {
			b := write([2]string{"outer/inner/valve.json", "1"})
			var decoded xzip.Files
			Expect(xzip.Codec.Decode(ctx, b, &decoded)).To(Succeed())
			Expect(decoded).To(HaveKeyWithValue("valve.json", []byte("1")))
		})
		It("Should skip macOS metadata beside root entries", func(ctx SpecContext) {
			b := write(
				[2]string{"valve.json", "1"},
				[2]string{".DS_Store", "junk"},
				[2]string{"._valve.json", "junk"},
			)
			var decoded xzip.Files
			Expect(xzip.Codec.Decode(ctx, b, &decoded)).To(Succeed())
			Expect(decoded).To(Equal(xzip.Files{"valve.json": []byte("1")}))
		})
		It("Should reject contents decompressing past the cap", func(
			ctx SpecContext,
		) {
			// Two half-budget entries plus one byte: small compressed, over the cap
			// decompressed, and neither entry alone crosses it.
			half := strings.Repeat("0", maxDecodedSize/2)
			b := write([2]string{"a.json", half}, [2]string{"b.json", half + "0"})
			var decoded xzip.Files
			Expect(xzip.Codec.Decode(ctx, b, &decoded)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("decompress past")),
			))
		})
		It("Should decode contents exactly at the cap", func(ctx SpecContext) {
			b := write([2]string{"a.json", strings.Repeat("0", maxDecodedSize)})
			var decoded xzip.Files
			Expect(xzip.Codec.Decode(ctx, b, &decoded)).To(Succeed())
			Expect(decoded["a.json"]).To(HaveLen(maxDecodedSize))
		})
		It("Should skip a directory entry", func(ctx SpecContext) {
			b := write([2]string{"nested/", ""}, [2]string{"valve.json", "1"})
			var decoded xzip.Files
			Expect(xzip.Codec.Decode(ctx, b, &decoded)).To(Succeed())
			Expect(decoded).To(Equal(xzip.Files{"valve.json": []byte("1")}))
		})
	})
	Describe("DecodeStream", func() {
		It("Should decode the archive from the reader", func(ctx SpecContext) {
			files := xzip.Files{"valve.json": []byte(`{"name":"valve"}`)}
			var decoded xzip.Files
			Expect(xzip.Codec.DecodeStream(
				ctx,
				bytes.NewReader(MustSucceed(xzip.Codec.Encode(ctx, files))),
				&decoded,
			)).To(Succeed())
			Expect(decoded).To(Equal(files))
		})
	})
})

var _ = DescribeTable("ParentDir",
	func(name, expected string) {
		Expect(xzip.ParentDir(name)).To(Equal(expected))
	},
	Entry("root entry", "manifest.json", ""),
	Entry("nested entry", "propulsion/pressurization.json", "propulsion"),
	Entry("deeply nested entry", "a/b/c.json", "a/b"),
	Entry("directory prefix", "a/b", "a"),
	Entry("empty name", "", ""),
)
