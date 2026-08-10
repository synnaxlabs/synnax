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
)

// Files is a flat namespace of file name to file contents. It carries no directories
// and no nesting: every name is a leaf, which the Encoder enforces.
type Files = map[string][]byte

// Marshaler is implemented by types that convert themselves into a flat file namespace.
// The Encoder accepts such a type in place of a Files value, so a caller can hand a
// domain type to the encoder and keep the file layout with the type it belongs to.
type Marshaler interface {
	// MarshalZIP returns the files the value encodes to.
	MarshalZIP() (Files, error)
}

// Encoder encodes a Files value into a zip archive with one entry per file. It also
// accepts a Marshaler, which it marshals to Files first. Entries are written in sorted
// name order, so equal Files always encode to equal bytes. Encoding any other value, a
// Marshaler that fails, and a file name that is not a leaf all return
// encoding.ErrEncode.
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
	var files Files
	switch value := value.(type) {
	case Files:
		files = value
	case Marshaler:
		var err error
		files, err = value.MarshalZIP()
		if err != nil {
			return encoding.SugarEncodingErr(value, err)
		}
	default:
		return encoding.SugarEncodingErr(value, errors.New(
			"value is not zip.Files and does not implement zip.Marshaler",
		))
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
			return encoding.SugarEncodingErr(value, err)
		}
		if _, err = f.Write(files[name]); err != nil {
			return encoding.SugarEncodingErr(value, err)
		}
	}
	return encoding.SugarEncodingErr(value, zw.Close())
}

// validateLeaf returns an error wrapping encoding.ErrEncode when name cannot be a flat
// archive entry: an empty name, a name holding a path separator, or a name addressing a
// directory. It names the offender itself rather than going through
// encoding.SugarEncodingErr, which reports the whole file map and drops the reason.
func validateLeaf(name string) error {
	switch {
	case name == "":
		return errors.Wrap(encoding.ErrEncode, "file name is empty")
	case strings.ContainsAny(name, `/\`):
		return errors.Wrapf(
			encoding.ErrEncode, "file name %q holds a path separator", name,
		)
	case name == "." || name == "..":
		return errors.Wrapf(
			encoding.ErrEncode, "file name %q addresses a directory", name,
		)
	}
	return nil
}
