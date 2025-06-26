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
	"encoding/csv"
	"io"

	"github.com/synnaxlabs/x/errors"
)

// CSVMarshaler is a type that can marshal itself to a CSV representation.
type CSVMarshaler interface {
	MarshalCSV() ([][]string, error)
}

// MarshalCSV marshals a value to a CSV representation. If each row in the CSV
// representation has a different length, an error will be returned.
func MarshalCSV(value any) ([][]string, error) {
	m, ok := value.(CSVMarshaler)
	if !ok {
		return nil, errors.Newf("%T does not implement CSVMarshaler", value)
	}
	records, err := m.MarshalCSV()
	if err != nil {
		return nil, err
	}
	if len(records) == 0 || len(records[0]) == 0 {
		return records, nil
	}
	rowLength := len(records[0])
	for i, row := range records {
		if len(row) != rowLength {
			return nil, errors.Newf("all rows must have the same length. Row %d has length %d, expected %d", i, len(row), rowLength)
		}
	}
	return records, nil
}

// CSVUnmarshaler is a type that can unmarshal itself from a CSV representation.
type CSVUnmarshaler interface {
	UnmarshalCSV([][]string) error
}

// UnmarshalCSV unmarshals a value from a CSV representation.
func UnmarshalCSV(data [][]string, value any) error {
	if u, ok := value.(CSVUnmarshaler); ok {
		return u.UnmarshalCSV(data)
	}
	return errors.Newf("%T does not implement CSVUnmarshaler", value)
}

// CSVCodec is a codec that encodes and decodes data to and from CSV format. CSVCodec
// implements the Codec interface.
type CSVCodec struct{}

var _ Codec = (*CSVCodec)(nil)

// Encode encodes a value to its CSV representation in bytes.
func (c *CSVCodec) Encode(ctx context.Context, value any) ([]byte, error) {
	records, err := MarshalCSV(value)
	if err != nil {
		return nil, sugarEncodingErr(value, err)
	}
	if len(records) == 0 {
		return []byte{}, nil
	}
	buf := bytes.NewBuffer(nil)
	csvWriter := csv.NewWriter(buf)
	csvWriter.UseCRLF = true
	if err := csvWriter.WriteAll(records); err != nil {
		return nil, sugarEncodingErr(value, err)
	}
	if err := csvWriter.Error(); err != nil {
		return nil, sugarEncodingErr(value, err)
	}
	// trim the last CRLF
	return buf.Bytes()[:buf.Len()-2], nil
}

// EncodeStream encodes a value to a CSV representation in bytes and writes it to a
// writer.
func (c *CSVCodec) EncodeStream(ctx context.Context, w io.Writer, value any) error {
	b, err := c.Encode(ctx, value)
	if err != nil {
		return sugarEncodingErr(value, err)
	}
	_, err = w.Write(b)
	return sugarEncodingErr(value, err)
}

// Decode decodes a value from a CSV representation in bytes.
func (c *CSVCodec) Decode(ctx context.Context, data []byte, value any) error {
	csvReader := csv.NewReader(bytes.NewReader(data))
	csvReader.ReuseRecord = true
	records, err := csvReader.ReadAll()
	if records == nil {
		records = [][]string{}
	}
	if err != nil {
		return sugarDecodingErr(data, value, err)
	}
	if err := UnmarshalCSV(records, value); err != nil {
		return sugarDecodingErr(data, value, err)
	}
	return nil
}

// DecodeStream decodes a value from a CSV representation in bytes and reads it from a
// reader.
func (c *CSVCodec) DecodeStream(ctx context.Context, r io.Reader, value any) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return sugarDecodingErr(data, value, err)
	}
	return c.Decode(ctx, data, value)
}
