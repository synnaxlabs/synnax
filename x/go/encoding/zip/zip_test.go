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
			Expect(read(MustSucceed(xzip.Encoder.Encode(files)))).To(Equal(files))
		})
		It("Should encode an empty file map into an empty archive", func(
			ctx SpecContext,
		) {
			Expect(
				read(MustSucceed(xzip.Encoder.Encode(xzip.Files{}))),
			).To(BeEmpty())
		})
		It("Should preserve an empty file's contents", func(ctx SpecContext) {
			b := MustSucceed(xzip.Encoder.Encode(xzip.Files{"empty.json": {}}))
			Expect(read(b)).To(HaveKeyWithValue("empty.json", BeEmpty()))
		})
		It("Should encode equal file maps to equal bytes", func(ctx SpecContext) {
			first := xzip.Files{"a.json": []byte("1"), "b.json": []byte("2")}
			second := xzip.Files{"b.json": []byte("2"), "a.json": []byte("1")}
			Expect(xzip.Encoder.Encode(first)).
				To(Equal(MustSucceed(xzip.Encoder.Encode(second))))
		})
		DescribeTable("Should reject a value that is not a file map",
			func(ctx SpecContext, value any) {
				Expect(xzip.Encoder.Encode(value)).Error().
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
			Expect(read(MustSucceed(xzip.Encoder.Encode(files)))).To(Equal(files))
		})
		DescribeTable("Should reject an invalid entry name",
			func(ctx SpecContext, name, reason string) {
				Expect(xzip.Encoder.Encode(xzip.Files{name: []byte("1")})).Error().
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
			Expect(xzip.Encoder.EncodeStream(&buf, files)).To(Succeed())
			Expect(read(buf.Bytes())).To(Equal(files))
		})
		It("Should reject a value that is not a file map", func(ctx SpecContext) {
			var buf bytes.Buffer
			Expect(xzip.Encoder.EncodeStream(&buf, "valve")).
				To(MatchError(ContainSubstring("failed to encode")))
		})
		It("Should write nothing when an entry name is invalid", func(
			ctx SpecContext,
		) {
			files := xzip.Files{"a.json": []byte("1"), `nested\b.json`: []byte("2")}
			var buf bytes.Buffer
			Expect(xzip.Encoder.EncodeStream(&buf, files)).
				To(MatchError(validate.ErrValidation))
			Expect(buf.Bytes()).To(BeEmpty())
		})
	})
})
