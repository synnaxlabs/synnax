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

	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/http"
)

// Files is a flat namespace of file name to file contents. It carries no directories
// and no nesting: every name is a leaf.
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
// name order, so equal Files always encode to equal bytes. Encoding any other value,
// and a Marshaler that fails, returns encoding.ErrEncode.
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
	zw := zip.NewWriter(w)
	for _, name := range slices.Sorted(maps.Keys(files)) {
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
