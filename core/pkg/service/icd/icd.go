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

// parse parses data in the given format into ICD messages. Every message and field gets
// a new key, so references between fields resolve. It returns an error scoped to the
// "format" path when the format is unknown, and one scoped to the "data" path, naming
// the line or row, when data is invalid.
func parse(format ImportFormat, data []byte) ([]Message, error) {
	switch format {
	case ImportFormatDBC:
		return parseDBC(data)
	case ImportFormatCSV:
		return parseCSV(data)
	case ImportFormatXLSX:
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
