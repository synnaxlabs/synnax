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

var _ Codec = (*CSVCodec)(nil)

type CSVCodec struct{}

type CSVMarshaler interface {
	MarshalCSV() ([][]string, error)
}

type CSVUnmarshaler interface {
	UnmarshalCSV([][]string) error
}

func MarshalCSV(value any) ([][]string, error) {
	if m, ok := value.(CSVMarshaler); ok {
		return m.MarshalCSV()
	}
	return nil, errors.Newf("%T does not implement CSVMarshaler", value)
}

func UnmarshalCSV(data [][]string, value any) error {
	if u, ok := value.(CSVUnmarshaler); ok {
		return u.UnmarshalCSV(data)
	}
	return errors.Newf("%T does not implement CSVUnmarshaler", value)
}

func validateCSV(records [][]string) error {
	rowLengths := len(records[0])
	for i, row := range records {
		if len(row) != rowLengths {
			return errors.Newf("all rows must have the same length. Row %d has length %d, expected %d", i, len(row), rowLengths)
		}
	}
	return nil
}

func (c *CSVCodec) Encode(ctx context.Context, value any) ([]byte, error) {
	records, err := MarshalCSV(value)
	if err != nil {
		return nil, sugarEncodingErr(value, err)
	}
	if len(records) == 0 {
		return []byte{}, nil
	}
	if err := validateCSV(records); err != nil {
		return nil, sugarEncodingErr(value, err)
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

func (c *CSVCodec) EncodeStream(ctx context.Context, w io.Writer, value any) error {
	b, err := c.Encode(ctx, value)
	if err != nil {
		return sugarEncodingErr(value, err)
	}
	_, err = w.Write(b)
	return sugarEncodingErr(value, err)
}

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

func (c *CSVCodec) DecodeStream(ctx context.Context, r io.Reader, value any) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return sugarDecodingErr(data, value, err)
	}
	return c.Decode(ctx, data, value)
}
