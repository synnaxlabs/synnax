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

// maxDecodedSize caps the total decompressed size of an archive Decode accepts,
// guarding against zip bombs. Enforced while reading, so a lying size header cannot
// bypass it.
const maxDecodedSize = 64 << 20

// Codec encodes a Files value into a zip archive with one entry per file and decodes
// such an archive back into a *Files. Entries are written in sorted name order, so
// equal Files always encode to equal bytes. Decode tolerates archives re-zipped by
// desktop archivers: it skips directory entries and macOS metadata (__MACOSX trees,
// ._* files, .DS_Store) and unwraps root directories all entries share. A value of
// another type fails, and a remaining name that is not a leaf, two entries under one
// name, or contents decompressing past a fixed 64 MiB cap, returns an error wrapping
// validate.ErrValidation.
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
	entries := make([]*zip.File, 0, len(zr.File))
	names := make([]string, 0, len(zr.File))
	for _, f := range zr.File {
		if strings.HasSuffix(f.Name, "/") || junk(f.Name) {
			continue
		}
		entries = append(entries, f)
		names = append(names, f.Name)
	}
	stripSharedRoot(names)
	decoded := make(Files, len(entries))
	seen := make(set.Set[string], len(entries))
	remaining := int64(maxDecodedSize)
	for i, f := range entries {
		name := names[i]
		if err := validateLeaf(name); err != nil {
			return err
		}
		if seen.Contains(name) {
			return errors.Wrapf(
				validate.ErrValidation, "file name %q repeats an earlier entry", name,
			)
		}
		seen.Add(name)
		rc, err := f.Open()
		if err != nil {
			return encoding.SugarDecodingError(data, value, err)
		}
		// The extra byte past the budget distinguishes an archive exactly at the limit
		// from one over it.
		contents, err := io.ReadAll(io.LimitReader(rc, remaining+1))
		if err != nil {
			err = errors.Combine(err, rc.Close())
			return encoding.SugarDecodingError(data, value, err)
		}
		if err := rc.Close(); err != nil {
			return encoding.SugarDecodingError(data, value, err)
		}
		if int64(len(contents)) > remaining {
			return errors.Wrapf(
				validate.ErrValidation,
				"archive contents decompress past the %d byte limit", maxDecodedSize,
			)
		}
		remaining -= int64(len(contents))
		decoded[name] = contents
	}
	*files = decoded
	return nil
}

// junk reports whether name is archiver metadata rather than content: the __MACOSX
// tree, AppleDouble ._* files, and .DS_Store.
func junk(name string) bool {
	if name == "__MACOSX" || strings.HasPrefix(name, "__MACOSX/") {
		return true
	}
	base := name[strings.LastIndexByte(name, '/')+1:]
	return base == ".DS_Store" || strings.HasPrefix(base, "._")
}

// stripSharedRoot trims directory prefixes shared by every name in place: while all
// names sit under one top-level directory, that directory is dropped. Desktop
// archivers produce this shape by zipping a folder rather than its contents.
func stripSharedRoot(names []string) {
	for len(names) > 0 {
		i := strings.IndexByte(names[0], '/')
		if i < 0 {
			return
		}
		root := names[0][:i+1]
		for _, name := range names[1:] {
			if !strings.HasPrefix(name, root) {
				return
			}
		}
		for j, name := range names {
			names[j] = strings.TrimPrefix(name, root)
		}
	}
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
