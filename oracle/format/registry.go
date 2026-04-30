// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package format

import (
	"context"
	"path/filepath"
	"runtime"

	"github.com/synnaxlabs/x/errors"
	"golang.org/x/sync/errgroup"
)

// File is one (path, content) pair. The path is absolute and is used both
// for routing (extension → formatter chain) and for tooling config
// discovery (--stdin-filepath, --assume-filename).
type File struct {
	Path    string
	Content []byte
}

// Registry routes files to formatters by file extension. A given extension
// may have multiple formatters; they run in the order they were registered
// (e.g. prettier first, then eslint --fix).
type Registry struct {
	byExt map[string][]Formatter
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{byExt: make(map[string][]Formatter)}
}

// Register appends a formatter for the given extension. The extension must
// include the leading dot (e.g. ".go", ".cpp"). When a file matches multiple
// formatters, they run sequentially in registration order, each receiving
// the previous formatter's output.
func (r *Registry) Register(ext string, f Formatter) {
	r.byExt[ext] = append(r.byExt[ext], f)
}

// Format runs all formatters registered for the file's extension,
// chaining their output. Returns content unchanged if no formatter is
// registered.
func (r *Registry) Format(ctx context.Context, content []byte, absPath string) ([]byte, error) {
	ext := filepath.Ext(absPath)
	formatters, ok := r.byExt[ext]
	if !ok {
		return content, nil
	}
	current := content
	for _, f := range formatters {
		out, err := f.Format(ctx, current, absPath)
		if err != nil {
			return nil, errors.Wrapf(err, "format %s", absPath)
		}
		current = out
	}
	return current, nil
}

// Has reports whether at least one formatter is registered for the given
// extension.
func (r *Registry) Has(ext string) bool {
	_, ok := r.byExt[ext]
	return ok
}

// FormatBatch formats a slice of files in parallel and returns a new slice
// with the canonical bytes. Worker count defaults to GOMAXPROCS; pass 0 for
// the default. Files with no registered formatter pass through unchanged.
//
// On the first error the context is cancelled and the partial results are
// discarded; the caller receives only the wrapped error.
func (r *Registry) FormatBatch(ctx context.Context, files []File, workers int) ([]File, error) {
	if workers <= 0 {
		workers = runtime.GOMAXPROCS(0)
	}
	out := make([]File, len(files))
	eg, ctx := errgroup.WithContext(ctx)
	eg.SetLimit(workers)
	for i, f := range files {
		i, f := i, f
		eg.Go(func() error {
			if err := ctx.Err(); err != nil {
				return err
			}
			content, err := r.Format(ctx, f.Content, f.Path)
			if err != nil {
				return errors.Wrapf(err, "format %s", f.Path)
			}
			out[i] = File{Path: f.Path, Content: content}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	return out, nil
}
