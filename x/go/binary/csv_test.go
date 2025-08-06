// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package binary_test

import (
	"context"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/errors"
	xio "github.com/synnaxlabs/x/io"
)

type marshaller struct{ records [][]string }

var _ binary.CSVMarshaler = (*marshaller)(nil)

func (m marshaller) MarshalCSV() ([][]string, error) { return m.records, nil }

type errMarshaller struct{}

var _ binary.CSVMarshaler = (*errMarshaller)(nil)

var errTest = errors.New("test")

func (m errMarshaller) MarshalCSV() ([][]string, error) { return nil, errTest }

var _ = Describe("CSVEncoder", func() {
	var ctx context.Context
	BeforeEach(func() {
		ctx = context.Background()
	})
	DescribeTableSubtree("Valid CSVs", func(v any, encoded []byte) {
		It("should encode the records", func() {
			Expect(binary.CSVEncoder.Encode(ctx, v)).To(Equal(encoded))
		})
		It("should encode the records to a stream", func() {
			Expect(binary.CSVEncoder.EncodeStream(ctx, io.Discard, v)).To(Succeed())
		})
	},
		Entry("basic", [][]string{{"a", "b"}, {"c", "d"}}, []byte("a,b\r\nc,d\r\n")),
		Entry("value that is a []string", []string{"a", "b"}, []byte("a,b\r\n")),
		Entry("value that implements CSVMarshaler",
			marshaller{records: [][]string{{"a", "b"}, {"c", "d"}}},
			[]byte("a,b\r\nc,d\r\n"),
		),
		Entry("empty", [][]string{}, []byte("")),
		Entry("empty string", []string{}, []byte("\r\n")),
		Entry("zero length rows", [][]string{{}, {}}, []byte("\r\n\r\n")),
		Entry("single row", [][]string{{"a", "b"}}, []byte("a,b\r\n")),
		Entry("double quotes",
			[][]string{{"a", "b"}, {"\"", "d"}},
			[]byte("a,b\r\n\"\"\"\",d\r\n"),
		),
		Entry("commas",
			[][]string{{"a", "b"}, {",", "d"}},
			[]byte("a,b\r\n\",\",d\r\n"),
		),
		Entry("newlines",
			[][]string{{"\r", "\n"}, {"\r\n", "\n\r"}},
			[]byte("\"\r\",\"\n\"\r\n\"\r\n\",\"\n\r\"\r\n"),
		),
		Entry("fields contain newlines",
			[]string{"hey\r\nthere", "Hello\nWorld"},
			[]byte("\"hey\r\nthere\",\"Hello\nWorld\"\r\n"),
		),
	)
	Describe("with a failing writer", func() {
		It("should return an error", func() {
			Expect(binary.CSVEncoder.EncodeStream(
				ctx,
				xio.FailWriter,
				[][]string{{"a", "b"}},
			)).Error().To(HaveOccurred())
		})
		It("should not return an error if writing empty records", func() {
			Expect(binary.CSVEncoder.EncodeStream(
				ctx,
				xio.FailWriter,
				[][]string{},
			)).To(Succeed())
		})
	})
	DescribeTableSubtree("Encoding errors", func(v any) {
		It("should return an error", func() {
			Expect(binary.CSVEncoder.Encode(ctx, v)).Error().To(HaveOccurred())
		})
		It("should return an error for the stream", func() {
			Expect(binary.CSVEncoder.EncodeStream(ctx, io.Discard, v)).
				Error().To(HaveOccurred())
		})
	},
		Entry("different lengths", [][]string{{"a", "b"}, {"c"}}),
		Entry("first row has zero length but others don't", [][]string{{}, {"c", "d"}}),
		Entry("value does not implement CSVMarshaler", [][]struct{}{{}}),
		Entry("value implements CSVMarshaler but returns an error", errMarshaller{}),
	)
	Describe("Encoding panics", func() {
		It("should panic if passing in the nil constant", func() {
			Expect(func() { binary.CSVEncoder.Encode(ctx, nil) }).To(Panic())
		})
		It("should panic if passing in nil to the stream encoder", func() {
			Expect(func() { binary.CSVEncoder.EncodeStream(ctx, io.Discard, nil) }).
				To(Panic())
		})
	})
})
