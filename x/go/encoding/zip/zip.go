// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package zip encodes a namespace of named files into a zip archive.
package zip

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"maps"
	"slices"
	"strings"

	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/http"
	"github.com/synnaxlabs/x/validate"
)

// Files maps entry names to file contents. A name is a relative path from the archive
// root in forward-slash form; directories exist only as prefixes of entry names, which
// the Encoder enforces.
type Files = map[string][]byte

// Encoder encodes a Files value into a zip archive with one entry per file. Entries are
// written in sorted name order, so equal Files always encode to equal bytes. Encoding a
// value that is not a Files fails, and an invalid entry name returns an error wrapping
// validate.ErrValidation.
var Encoder http.Encoder = encoder{}

type encoder struct{}

func (encoder) ContentType() string { return "application/zip" }

func (e encoder) Encode(ctx context.Context, value any) ([]byte, error) {
	var buf bytes.Buffer
	if err := e.EncodeStream(ctx, &buf, value); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (encoder) EncodeStream(_ context.Context, w io.Writer, value any) error {
	files, ok := value.(Files)
	if !ok {
		return encoding.SugarEncodingError(
			value, errors.New("value is not zip.Files"),
		)
	}
	// Every name is checked before the first entry is written, so a rejected archive
	// leaves no partial output on w.
	names := slices.Sorted(maps.Keys(files))
	for _, name := range names {
		if err := validateEntryName(name); err != nil {
			return err
		}
	}
	zw := zip.NewWriter(w)
	for _, name := range names {
		f, err := zw.Create(name)
		if err != nil {
			return encoding.SugarEncodingError(value, err)
		}
		if _, err = f.Write(files[name]); err != nil {
			return encoding.SugarEncodingError(value, err)
		}
	}
	return encoding.SugarEncodingError(value, zw.Close())
}

// validateEntryName returns an error wrapping validate.ErrValidation when name is not
// a relative forward-slash path of non-empty segments: an empty name, a backslash, a
// leading, trailing, or doubled slash, or a "." or ".." segment. It names the offender
// itself rather than going through encoding.SugarEncodingError, which reports the whole
// file map and drops the reason.
func validateEntryName(name string) error {
	if name == "" {
		return errors.Wrap(validate.ErrValidation, "file name is empty")
	}
	if strings.Contains(name, `\`) {
		return errors.Wrapf(
			validate.ErrValidation, "file name %q holds a backslash", name,
		)
	}
	for segment := range strings.SplitSeq(name, "/") {
		switch segment {
		case "":
			return errors.Wrapf(
				validate.ErrValidation, "file name %q holds an empty path segment", name,
			)
		case ".", "..":
			return errors.Wrapf(
				validate.ErrValidation, "file name %q addresses a directory", name,
			)
		}
	}
	return nil
}
