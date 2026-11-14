// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package icd stores interface control documents (ICDs): the messages a device sends
// and receives on a bus or byte stream. It checks each ICD and imports messages from
// DBC files and field tables.
package icd

import (
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// Format is the file format of an interface control document.
type Format string

const (
	// FormatDBC is a Vector CAN database file.
	FormatDBC Format = "dbc"
	// FormatCSV is a field table in comma-separated values. The first row is a header
	// that names each column, in any case: message, id, extended, length, field,
	// start_bit, bit_length, byte_order, signed, float, scale, offset, units, min, and
	// max. The message, field, start_bit, and bit_length columns are required. Each
	// other row is one binary field, and the rows of one message form one message, with
	// a CAN identifier when an id is present.
	FormatCSV Format = "csv"
	// FormatXLSX is a field table, laid out as for FormatCSV, in the first sheet of an
	// Excel workbook. A numeric cell is read as its value, whatever its number format.
	FormatXLSX Format = "xlsx"
)

// parse parses data in the given format into ICD messages. Every message and field gets
// a new key, so references between fields resolve. It returns an error scoped to the
// "format" path when the format is unknown, and one scoped to the "data" path, naming
// the line or row, when data is invalid.
func parse(format Format, data []byte) ([]Message, error) {
	switch format {
	case FormatDBC:
		return parseDBC(data)
	case FormatCSV:
		return parseCSV(data)
	case FormatXLSX:
		return parseXLSX(data)
	}
	return nil, validate.PathedError(
		errors.Wrapf(validate.ErrValidation, "unsupported format %q", format),
		"format",
	)
}

func dataErrorf(format string, args ...any) error {
	return validate.PathedError(
		errors.Wrapf(validate.ErrValidation, format, args...),
		"data",
	)
}
