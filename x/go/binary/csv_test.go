// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package binary

import (
	"bytes"
	"context"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type marshaller struct{ records [][]string }

var (
	_ CSVMarshaler   = (*marshaller)(nil)
	_ CSVUnmarshaler = (*marshaller)(nil)
)

func (m marshaller) MarshalCSV() ([][]string, error) { return m.records, nil }

func (m *marshaller) UnmarshalCSV(records [][]string) error {
	m.records = records
	return nil
}

var _ = Describe("CSV", func() {
	records := [][]string{{"a", "b"}, {"c", "d"}}
	Describe("MarshalCSV", func() {
		DescribeTable("should marshal the records", func(records [][]string) {
			m := &marshaller{records: records}
			Expect(MarshalCSV(m)).To(Equal(records))
		},
			Entry("basic", [][]string{{"a", "b"}, {"c", "d"}}),
			Entry("zero length", [][]string{}),
			Entry("zero length rows", [][]string{{}, {}}),
		)
		DescribeTable("should return an error", func(records [][]string) {
			m := &marshaller{records: records}
			Expect(MarshalCSV(m)).Error().To(HaveOccurred())
		},
			Entry("different lengths", [][]string{{"a", "b"}, {"c"}}),
			Entry("first row has zero length but the rest do not", [][]string{{}, {"c", "d"}}),
		)
		It("should return an error if the value does not implement CSVMarshaler", func() {
			Expect(MarshalCSV(struct{}{})).Error().To(HaveOccurred())
		})
	})
	Describe("UnmarshalCSV", func() {
		DescribeTable("should unmarshal the records", func(records [][]string) {
			u := marshaller{}
			Expect(UnmarshalCSV(records, &u)).To(Succeed())
			Expect(u.records).To(Equal(records))
		},
			Entry("basic", [][]string{{"a", "b"}, {"c", "d"}}),
			Entry("zero length", [][]string{}),
			Entry("zero length rows", [][]string{{}, {}}),
		)
		It("should return an error if the value does not implement CSVUnmarshaler", func() {
			Expect(UnmarshalCSV(records, &struct{}{})).Error().To(HaveOccurred())
		})
	})
	Describe("Codec", func() {
		DescribeTableSubtree("Encoding and decoding valid CSV", func(records [][]string, encoded []byte) {
			codec := &CSVCodec{}
			Describe("Encoding", func() {
				marshaler := &marshaller{records: records}
				It("Regular", func() {
					Expect(codec.Encode(context.Background(), marshaler)).To(Equal(encoded))
				})
				It("Stream", func() {
					Expect(codec.EncodeStream(context.Background(), io.Discard, marshaler)).To(Succeed())
				})
			})
			Describe("Decoding", func() {
				unmarshaler := &marshaller{}
				It("Regular", func() {
					Expect(codec.Decode(context.Background(), encoded, unmarshaler)).To(Succeed())
					Expect(unmarshaler.records).To(Equal(records))
				})
				It("Stream", func() {
					Expect(codec.DecodeStream(context.Background(), bytes.NewReader(encoded), unmarshaler)).To(Succeed())
					Expect(unmarshaler.records).To(Equal(records))
				})
			})
		},
			Entry("basic", [][]string{{"a", "b"}, {"c", "d"}}, []byte("a,b\r\nc,d\r\n")),
			Entry("double quotes", [][]string{{"a", "b"}, {"\"", "d"}}, []byte("a,b\r\n\"\"\"\",d\r\n")),
			Entry("empty", [][]string{}, []byte("")),
			Entry("single row", [][]string{{"a", "b"}}, []byte("a,b\r\n")),
			Entry("commas", [][]string{{"a", "b"}, {",", "d"}}, []byte("a,b\r\n\",\",d\r\n")),
		)
		Describe("Data with CRLFs", func() {
			// These tests only encode data because the standard library's CSV reader
			// automatically converts CRLFs in data to LFs. See
			// https://linear.app/synnax/issue/SY-2639/stop-stripping-of-cr-in-decoding-csvs.
			// The decoding tests just make sure that no errors are returned. Since we
			// don't call Decode or DecodeStream for CSVs, we don't need to worry about
			// this yet.
			codec := &CSVCodec{}
			encoded := []byte("a,b\r\n\"\r\n\",d\r\n")
			Describe("Encoding", func() {
				marshaler := &marshaller{records: [][]string{{"a", "b"}, {"\r\n", "d"}}}
				It("Regular", func() {
					Expect(codec.Encode(context.Background(), marshaler)).To(Equal(encoded))
				})
				It("Stream", func() {
					Expect(codec.EncodeStream(context.Background(), io.Discard, marshaler)).To(Succeed())
				})
			})
			Describe("Decoding", func() {
				unmarshaler := &marshaller{}
				It("Regular", func() {
					Expect(codec.Decode(context.Background(), encoded, unmarshaler)).To(Succeed())
				})
				It("Stream", func() {
					Expect(codec.DecodeStream(context.Background(), bytes.NewReader(encoded), unmarshaler)).To(Succeed())
				})
			})
		})
		Describe("Row length errors", func() {
			codec := &CSVCodec{}
			Describe("Encoding", func() {
				marshaler := &marshaller{records: [][]string{{"a", "b"}, {"c"}}}
				It("Regular", func() {
					Expect(codec.Encode(context.Background(), marshaler)).Error().To(HaveOccurred())
				})
				It("Stream", func() {
					Expect(codec.EncodeStream(context.Background(), io.Discard, marshaler)).To(HaveOccurred())
				})
			})
			Describe("Decoding", func() {
				unmarshaler := &marshaller{}
				encoded := []byte("a,b\r\nc")
				It("Regular", func() {
					Expect(codec.Decode(context.Background(), encoded, unmarshaler)).To(HaveOccurred())
				})
				It("Stream", func() {
					Expect(codec.DecodeStream(context.Background(), bytes.NewReader(encoded), unmarshaler)).To(HaveOccurred())
				})
			})
		})
	})
	Describe("CSVRecords", func() {
		var data = [][]string{{"a", "b"}, {"c", "d"}}
		Describe("MarshalCSV", func() {
			It("should marshal the records", func() {
				records := CSVRecords(data)
				Expect(MarshalCSV(records)).To(Equal(data))
			})
		})
		Describe("UnmarshalCSV", func() {
			It("should unmarshal the records", func() {
				var records CSVRecords
				Expect(UnmarshalCSV(data, &records)).To(Succeed())
				Expect(records).To(Equal(CSVRecords(data)))
			})
		})
	})
	Describe("NewCSVRecords", func() {
		DescribeTable("should create a new CSVRecords", func(rows int, cols int, expected CSVRecords) {
			records := NewCSVRecords(rows, cols)
			Expect(records).To(Equal(expected))
		},
			Entry("basic", 2, 2, CSVRecords{{"", ""}, {"", ""}}),
			Entry("zero rows", 0, 2, CSVRecords{}),
			Entry("zero columns", 2, 0, CSVRecords{{}, {}}),
			Entry("zero rows and columns", 0, 0, CSVRecords{}),
		)
		It("should error if the number of rows is negative", func() {
			Expect(func() { NewCSVRecords(-1, 2) }).To(Panic())
		})
		It("should error if the number of columns is negative", func() {
			Expect(func() { NewCSVRecords(2, -1) }).To(Panic())
		})
	})
})
