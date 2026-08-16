// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package zip encodes a flat namespace of named files into a zip archive.
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
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// Files is a flat namespace of file name to file contents. It carries no directories
// and no nesting: every name is a leaf, which the Codec enforces.
type Files = map[string][]byte

// Codec encodes a Files value into a zip archive with one entry per file and decodes
// such an archive back into a *Files. Entries are written in sorted name order, so
// equal Files always encode to equal bytes. A value of another type fails, and a file
// name that is not a leaf, or an archive holding two entries under one name, returns
// an error wrapping validate.ErrValidation.
var Codec http.Codec = codec{}

type codec struct{}

func (codec) ContentType() string { return "application/zip" }

func (c codec) Encode(ctx context.Context, value any) ([]byte, error) {
	var buf bytes.Buffer
	if err := c.EncodeStream(ctx, &buf, value); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (codec) EncodeStream(_ context.Context, w io.Writer, value any) error {
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
		if err := validateLeaf(name); err != nil {
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

func (codec) Decode(_ context.Context, data []byte, value any) error {
	files, ok := value.(*Files)
	if !ok {
		return encoding.SugarDecodingError(
			data, value, errors.New("value is not *zip.Files"),
		)
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return encoding.SugarDecodingError(data, value, err)
	}
	decoded := make(Files, len(zr.File))
	names := make(set.Set[string], len(zr.File))
	for _, f := range zr.File {
		if err := validateLeaf(f.Name); err != nil {
			return err
		}
		if names.Contains(f.Name) {
			return errors.Wrapf(
				validate.ErrValidation, "file name %q repeats an earlier entry", f.Name,
			)
		}
		names.Add(f.Name)
		rc, err := f.Open()
		if err != nil {
			return encoding.SugarDecodingError(data, value, err)
		}
		contents, err := io.ReadAll(rc)
		if err != nil {
			err = errors.Combine(err, rc.Close())
			return encoding.SugarDecodingError(data, value, err)
		}
		if err := rc.Close(); err != nil {
			return encoding.SugarDecodingError(data, value, err)
		}
		decoded[f.Name] = contents
	}
	*files = decoded
	return nil
}

func (c codec) DecodeStream(ctx context.Context, r io.Reader, value any) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return encoding.SugarDecodingError(nil, value, err)
	}
	return c.Decode(ctx, data, value)
}

// validateLeaf returns an error wrapping validate.ErrValidation when name cannot be a
// flat archive entry: an empty name, a name holding a path separator, or a name
// addressing a directory. It names the offender itself rather than going through
// encoding.SugarEncodingError, which reports the whole file map and drops the reason.
func validateLeaf(name string) error {
	switch {
	case name == "":
		return errors.Wrap(validate.ErrValidation, "file name is empty")
	case strings.ContainsAny(name, `/\`):
		return errors.Wrapf(
			validate.ErrValidation, "file name %q holds a path separator", name,
		)
	case name == "." || name == "..":
		return errors.Wrapf(
			validate.ErrValidation, "file name %q addresses a directory", name,
		)
	}
	return nil
}
