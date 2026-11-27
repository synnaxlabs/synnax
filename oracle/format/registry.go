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
	"sync"

	"github.com/synnaxlabs/x/errors"
	"golang.org/x/sync/semaphore"
)

// File is one (path, content) pair. The path is absolute and is used both
// for routing (extension → formatter chain) and for tooling config
// discovery (--stdin-filepath, --assume-filename).
type File struct {
	Path    string
	Content []byte
}

// Result is the outcome of formatting one File: the canonical bytes, or the error that
// stopped its formatter chain.
type Result struct {
	Content []byte
	Err     error
}

// Registry routes files to formatters by file extension. A given extension
// may have multiple formatters; they run in the order they were registered
// (e.g. eslint --fix first, then prettier).
type Registry struct {
	byExt map[string][]batchFormatter
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{byExt: make(map[string][]batchFormatter)}
}

// Register appends a formatter for the given extension. The extension must
// include the leading dot (e.g. ".go", ".cpp"). When a file matches multiple
// formatters, they run sequentially in registration order, each receiving
// the previous formatter's output.
func (r *Registry) Register(ext string, f formatter) {
	r.registerBatch(ext, eachFile{f: f})
}

func (r *Registry) registerBatch(ext string, f batchFormatter) {
	r.byExt[ext] = append(r.byExt[ext], f)
}

// FormatBatch formats files and returns one Result per file, in input order. The files
// of one extension pass through each formatter of its chain as one batch, and at most
// workers tool processes run at once; pass 0 for GOMAXPROCS. Files with no registered
// formatter pass through unchanged. A file whose chain fails gets the error in its
// Result, and the other files still format.
func (r *Registry) FormatBatch(
	ctx context.Context,
	files []File,
	workers int,
) []Result {
	if workers <= 0 {
		workers = runtime.GOMAXPROCS(0)
	}
	sem := semaphore.NewWeighted(int64(workers))
	out := make([]Result, len(files))
	byExt := make(map[string][]int)
	for i, f := range files {
		out[i].Content = f.Content
		ext := filepath.Ext(f.Path)
		if _, ok := r.byExt[ext]; ok {
			byExt[ext] = append(byExt[ext], i)
		}
	}
	var wg sync.WaitGroup
	for ext, indices := range byExt {
		wg.Go(func() { formatChain(ctx, r.byExt[ext], files, indices, out, sem) })
	}
	wg.Wait()
	return out
}

// formatChain runs each formatter of chain over the files at indices that have not
// failed yet and writes the results to out.
func formatChain(
	ctx context.Context,
	chain []batchFormatter,
	files []File,
	indices []int,
	out []Result,
	sem *semaphore.Weighted,
) {
	for _, f := range chain {
		live := make([]int, 0, len(indices))
		batch := make([]File, 0, len(indices))
		for _, i := range indices {
			if out[i].Err == nil {
				live = append(live, i)
				batch = append(
					batch,
					File{Path: files[i].Path, Content: out[i].Content},
				)
			}
		}
		for j, res := range f.formatBatch(ctx, batch, sem) {
			i := live[j]
			if res.Err != nil {
				res = Result{Err: errors.Wrapf(res.Err, "format %s", files[i].Path)}
			}
			out[i] = res
		}
	}
}

// eachFile runs a per-file formatter once for each file of a batch.
type eachFile struct{ f formatter }

func (e eachFile) formatBatch(
	ctx context.Context,
	files []File,
	sem *semaphore.Weighted,
) []Result {
	out := make([]Result, len(files))
	var wg sync.WaitGroup
	for i, file := range files {
		if err := sem.Acquire(ctx, 1); err != nil {
			out[i].Err = err
			continue
		}
		wg.Go(func() {
			defer sem.Release(1)
			out[i].Content, out[i].Err = e.f.Format(ctx, file.Content, file.Path)
		})
	}
	wg.Wait()
	return out
}
