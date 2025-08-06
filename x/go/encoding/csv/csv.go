// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package csv

import (
	"context"
	"io"

	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/encoding/csv/internal/csv"
	"github.com/synnaxlabs/x/errors"
)

// Encoder is an encoder that encodes data to the CSV format.
var Encoder = &encoder{}

// Marshaler is a type that can marshal itself to a CSV representation.
type Marshaler interface{ MarshalCSV() ([][]string, error) }

type encoder struct{}

var _ binary.Encoder = (*encoder)(nil)

// Encode encodes a value to its CSV representation in bytes. The value must either
// implement the Marshaler interface or be a [][]string or []string.
func (enc *encoder) Encode(ctx context.Context, v any) ([]byte, error) {
	return binary.WrapStreamEncoder(enc, ctx, v)
}

// EncodeStream encodes a value to a CSV representation in bytes and writes it to a
// writer. The value must either implement the Marshaler interface or be a [][]string or
// []string.
func (enc *encoder) EncodeStream(_ context.Context, w io.Writer, v any) error {
	records, err := marshalCSV(v)
	if err != nil {
		return binary.SugarEncodingErr(v, err)
	}
	csvWriter := csv.NewWriter(w)
	csvWriter.UseCRLF = true
	err = csvWriter.WriteAll(records)
	return binary.SugarEncodingErr(v, err)
}

func marshalCSV(v any) ([][]string, error) {
	var records [][]string
	switch target := v.(type) {
	case Marshaler:
		var err error
		if records, err = target.MarshalCSV(); err != nil {
			return nil, err
		}
	case [][]string:
		records = target
	case []string:
		records = [][]string{target}
	default:
		return nil, errors.Newf("%T does not implement CSVMarshaler", v)
	}
	if len(records) == 0 {
		return records, nil
	}
	rowLength := len(records[0])
	for i, row := range records {
		if len(row) != rowLength {
			return nil, errors.Newf(
				"all rows must have the same length. Row %d has length %d, expected %d",
				i,
				len(row),
				rowLength,
			)
		}
	}
	return records, nil
}
