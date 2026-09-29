// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package icd parses interface control documents (DBC files and field tables) into
// library entries, and merges imported entries into an existing library.
package icd

import (
	"github.com/synnaxlabs/synnax/pkg/service/library/versions"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// Format is the file format of an interface control document.
type Format string

const (
	// FormatDBC is a Vector CAN database file.
	FormatDBC Format = "dbc"
	// FormatCSV is a field table in comma-separated values.
	FormatCSV Format = "csv"
	// FormatXLSX is a field table in the first sheet of an Excel workbook.
	FormatXLSX Format = "xlsx"
)

// Parse parses data in the given format into library entries. Every entry and field
// gets a new key, so references between them resolve. It returns an error scoped to
// the "format" path when the format is unknown, and one scoped to the "data" path,
// naming the line or row, when data is invalid.
func Parse(format Format, data []byte) ([]versions.Entry, error) {
	switch format {
	case FormatDBC:
		return ParseDBC(data)
	case FormatCSV:
		return ParseCSV(data)
	case FormatXLSX:
		return ParseXLSX(data)
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
