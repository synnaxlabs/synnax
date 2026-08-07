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
	azip "archive/zip"
	"bytes"
	"context"
	"io"
	"maps"
	"slices"

	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/errors"
	xhttp "github.com/synnaxlabs/x/http"
)

// Files is a flat namespace of file name to file contents. It carries no directories
// and no nesting: every name is a leaf.
type Files = map[string][]byte

// ContentType is the HTTP content type of a zip archive.
const ContentType = "application/zip"

// Encoder encodes a Files value into a zip archive with one entry per file. Entries are
// written in sorted name order, so equal Files always encode to equal bytes. Encoding
// any other value returns encoding.ErrEncode.
var Encoder xhttp.Encoder = encoder{}

type encoder struct{}

func (encoder) ContentType() string { return ContentType }

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
		return encoding.SugarEncodingErr(value, errors.New("value is not zip.Files"))
	}
	zw := azip.NewWriter(w)
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
