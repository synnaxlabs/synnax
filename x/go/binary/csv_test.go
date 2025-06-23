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

type marshaller struct {
	records [][]string
}

func (m marshaller) MarshalCSV() ([][]string, error) {
	return m.records, nil
}

func (m *marshaller) UnmarshalCSV(records [][]string) error {
	m.records = records
	return nil
}

var _ CSVMarshaler = marshaller{}
var _ CSVUnmarshaler = (*marshaller)(nil)

var _ = Describe("CSV", func() {
	var records [][]string
	BeforeEach(func() {
		records = [][]string{{"a", "b"}, {"c", "d"}}
	})
	Describe("MarshalCSV", func() {
		It("should marshal the records", func() {
			m := &marshaller{records: records}
			encoded, err := MarshalCSV(m)
			Expect(err).To(BeNil())
			Expect(encoded).To(Equal(records))
		})
		It("should return an error if the value does not implement CSVMarshaler", func() {
			m := struct{}{}
			_, err := MarshalCSV(m)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("does not implement CSVMarshaler"))
		})
	})
	Describe("UnmarshalCSV", func() {
		It("should unmarshal the records", func() {
			u := marshaller{}
			err := UnmarshalCSV(records, &u)
			Expect(err).To(BeNil())
			Expect(u.records).To(Equal(records))
		})
		It("should return an error if the value does not implement CSVUnmarshaler", func() {
			v := struct{}{}
			err := UnmarshalCSV(records, &v)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("does not implement CSVUnmarshaler"))
		})
	})
	Describe("Codec", func() {
		DescribeTableSubtree("Encoding and decoding valid CSV", func(records [][]string, encoded []byte) {
			var codec *CSVCodec
			BeforeEach(func() {
				codec = &CSVCodec{}
			})
			Describe("Encoding", func() {
				var marshaler *marshaller
				BeforeEach(func() {
					marshaler = &marshaller{records: records}
				})
				It("Regular", func() {
					data, err := codec.Encode(context.Background(), marshaler)
					Expect(err).To(BeNil())
					Expect(data).To(Equal(encoded))
				})
				It("Stream", func() {
					err := codec.EncodeStream(context.Background(), io.Discard, marshaler)
					Expect(err).To(BeNil())
				})
			})
			Describe("Decoding", func() {
				var unmarshaler *marshaller
				BeforeEach(func() {
					unmarshaler = &marshaller{}
				})
				It("Regular", func() {
					err := codec.Decode(context.Background(), encoded, unmarshaler)
					Expect(err).To(BeNil())
					Expect(unmarshaler.records).To(Equal(records))
				})
				It("Stream", func() {
					err := codec.DecodeStream(context.Background(), bytes.NewReader(encoded), unmarshaler)
					Expect(err).To(BeNil())
					Expect(unmarshaler.records).To(Equal(records))
				})
			})
		},
			Entry("basic", [][]string{{"a", "b"}, {"c", "d"}}, []byte("a,b\r\nc,d")),
			Entry("double quotes", [][]string{{"a", "b"}, {"\"", "d"}}, []byte("a,b\r\n\"\"\"\",d")),
			Entry("empty", [][]string{}, []byte("")),
			Entry("single row", [][]string{{"a", "b"}}, []byte("a,b")),
			Entry("commas", [][]string{{"a", "b"}, {",", "d"}}, []byte("a,b\r\n\",\",d")),
		)
		Describe("Data with CRLFs", func() {
			// These tests only encode data because the standard library's CSV reader
			// automatically converts CRLFs in data to LFs. See
			// https://linear.app/synnax/issue/SY-2638/add-csv-data-exporting-to-server.
			// The decoding tests just make sure that no errors are returned. Since we
			// don't call Decode or DecodeStream for CSVs, we don't need to worry about
			// this.
			var (
				codec   *CSVCodec
				encoded []byte
			)
			BeforeEach(func() {
				codec = &CSVCodec{}
				encoded = []byte("a,b\r\n\"\r\n\",d")
			})
			Describe("Encoding", func() {
				var marshaler *marshaller
				BeforeEach(func() {
					marshaler = &marshaller{records: [][]string{{"a", "b"}, {"\r\n", "d"}}}
				})
				It("Regular", func() {
					data, err := codec.Encode(context.Background(), marshaler)
					Expect(err).To(BeNil())
					Expect(data).To(Equal(encoded))
				})
				It("Stream", func() {
					err := codec.EncodeStream(context.Background(), io.Discard, marshaler)
					Expect(err).To(BeNil())
				})
			})
			Describe("Decoding", func() {
				var unmarshaler *marshaller
				BeforeEach(func() {
					unmarshaler = &marshaller{}
				})
				It("Regular", func() {
					err := codec.Decode(context.Background(), encoded, unmarshaler)
					Expect(err).To(BeNil())
				})
				It("Stream", func() {
					err := codec.DecodeStream(context.Background(), bytes.NewReader(encoded), unmarshaler)
					Expect(err).To(BeNil())
				})
			})
		})
		Describe("Row length errors", func() {
			var codec *CSVCodec
			BeforeEach(func() {
				codec = &CSVCodec{}
			})
			Describe("Encoding", func() {
				var marshaler *marshaller
				BeforeEach(func() {
					marshaler = &marshaller{records: [][]string{{"a", "b"}, {"c"}}}
				})
				It("Regular", func() {
					_, err := codec.Encode(context.Background(), marshaler)
					Expect(err).To(HaveOccurred())
				})
				It("Stream", func() {
					err := codec.EncodeStream(context.Background(), io.Discard, marshaler)
					Expect(err).To(HaveOccurred())
				})
			})
			Describe("Decoding", func() {
				var unmarshaler *marshaller
				var encoded []byte
				BeforeEach(func() {
					unmarshaler = &marshaller{}
					encoded = []byte("a,b\r\nc")
				})
				It("Regular", func() {
					err := codec.Decode(context.Background(), encoded, unmarshaler)
					Expect(err).To(HaveOccurred())
				})
				It("Stream", func() {
					err := codec.DecodeStream(context.Background(), bytes.NewReader(encoded), unmarshaler)
					Expect(err).To(HaveOccurred())
				})
			})
		})
	})
})
