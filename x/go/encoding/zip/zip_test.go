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
	azip "archive/zip"
	"bytes"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/encoding/zip"
	. "github.com/synnaxlabs/x/testutil"
)

// read unpacks b into the file map it was encoded from.
func read(b []byte) zip.Files {
	GinkgoHelper()
	r := MustSucceed(azip.NewReader(bytes.NewReader(b), int64(len(b))))
	files := make(zip.Files, len(r.File))
	for _, f := range r.File {
		rc := MustSucceed(f.Open())
		files[f.Name] = MustSucceed(io.ReadAll(rc))
		Expect(rc.Close()).To(Succeed())
	}
	return files
}

var _ = Describe("Zip", func() {
	Describe("ContentType", func() {
		It("Should report the zip content type", func() {
			Expect(zip.Encoder.ContentType()).To(Equal("application/zip"))
		})
	})
	Describe("Encode", func() {
		It("Should encode every file into a readable archive", func(ctx SpecContext) {
			files := zip.Files{
				"manifest.json": []byte(`{"version":2}`),
				"valve.json":    []byte(`{"name":"valve"}`),
			}
			Expect(read(MustSucceed(zip.Encoder.Encode(ctx, files)))).To(Equal(files))
		})
		It("Should encode an empty file map into an empty archive", func(
			ctx SpecContext,
		) {
			Expect(
				read(MustSucceed(zip.Encoder.Encode(ctx, zip.Files{}))),
			).To(BeEmpty())
		})
		It("Should preserve an empty file's contents", func(ctx SpecContext) {
			b := MustSucceed(zip.Encoder.Encode(ctx, zip.Files{"empty.json": {}}))
			Expect(read(b)).To(HaveKeyWithValue("empty.json", BeEmpty()))
		})
		It("Should encode equal file maps to equal bytes", func(ctx SpecContext) {
			first := zip.Files{"a.json": []byte("1"), "b.json": []byte("2")}
			second := zip.Files{"b.json": []byte("2"), "a.json": []byte("1")}
			Expect(zip.Encoder.Encode(ctx, first)).
				To(Equal(MustSucceed(zip.Encoder.Encode(ctx, second))))
		})
		DescribeTable("Should reject a value that is not a file map",
			func(ctx SpecContext, value any) {
				Expect(zip.Encoder.Encode(ctx, value)).Error().
					To(MatchError(encoding.ErrEncode))
			},
			Entry("a struct", struct{ Name string }{Name: "valve"}),
			Entry("a string map", map[string]string{"a.json": "1"}),
			Entry("a byte slice", []byte("valve")),
		)
	})
	Describe("EncodeStream", func() {
		It("Should write the archive to the writer", func(ctx SpecContext) {
			files := zip.Files{"valve.json": []byte(`{"name":"valve"}`)}
			var buf bytes.Buffer
			Expect(zip.Encoder.EncodeStream(ctx, &buf, files)).To(Succeed())
			Expect(read(buf.Bytes())).To(Equal(files))
		})
		It("Should reject a value that is not a file map", func(ctx SpecContext) {
			var buf bytes.Buffer
			Expect(zip.Encoder.EncodeStream(ctx, &buf, "valve")).
				To(MatchError(encoding.ErrEncode))
		})
	})
})
