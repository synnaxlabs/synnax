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
	_ "embed"
	"encoding/json/v2"
	"path/filepath"
	"sync"

	"github.com/synnaxlabs/x/errors"
	"golang.org/x/sync/semaphore"
)

//go:embed typescript.mjs
var typescriptScript string

// typescript applies `eslint --fix` and then prettier to TypeScript files. It runs one
// Node process for each package (the nearest directory with a package.json), so the
// package's ESLint config and TypeScript program load once for all of its files. Each
// file gets the same bytes as the eslint and prettier CLIs on stdin would give it.
// Non-fatal lint findings are tolerated, since generated code may trip rules that have
// no auto-fix. A fatal parse error fails the file.
type typescript struct {
	// Bin is the Node binary to run.
	Bin string
}

func newTypeScript() *typescript { return &typescript{Bin: "node"} }

type typescriptFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type typescriptResult struct {
	Content string `json:"content"`
	Error   string `json:"error"`
}

func (t *typescript) formatBatch(
	ctx context.Context,
	files []File,
	sem *semaphore.Weighted,
) []Result {
	out := make([]Result, len(files))
	byDir := make(map[string][]int)
	for i, f := range files {
		dir := findProjectDir(f.Path, "package.json")
		if dir == "" {
			dir = filepath.Dir(f.Path)
		}
		byDir[dir] = append(byDir[dir], i)
	}
	var wg sync.WaitGroup
	for dir, indices := range byDir {
		if err := sem.Acquire(ctx, 1); err != nil {
			for _, i := range indices {
				out[i].Err = err
			}
			continue
		}
		wg.Go(func() {
			defer sem.Release(1)
			t.run(ctx, dir, files, indices, out)
		})
	}
	wg.Wait()
	return out
}

// run formats the files at indices in one Node process with dir as its working
// directory and writes the results to out.
func (t *typescript) run(
	ctx context.Context,
	dir string,
	files []File,
	indices []int,
	out []Result,
) {
	fail := func(err error) {
		for _, i := range indices {
			out[i].Err = err
		}
	}
	in := make([]typescriptFile, len(indices))
	for j, i := range indices {
		in[j] = typescriptFile{Path: files[i].Path, Content: string(files[i].Content)}
	}
	stdin, err := json.Marshal(in)
	if err != nil {
		fail(err)
		return
	}
	stdout, err := stdinRun{
		Name:  t.Bin,
		Args:  []string{"--input-type=module", "--eval", typescriptScript},
		Dir:   dir,
		Stdin: stdin,
	}.run(ctx)
	if err != nil {
		fail(err)
		return
	}
	var results []typescriptResult
	if err := json.Unmarshal(stdout, &results); err != nil {
		fail(errors.Wrapf(err, "parse formatter output (cwd=%s): %s", dir, stdout))
		return
	}
	if len(results) != len(indices) {
		fail(errors.Newf(
			"formatter returned %d results for %d files (cwd=%s)",
			len(results), len(indices), dir,
		))
		return
	}
	for j, i := range indices {
		if results[j].Error != "" {
			out[i].Err = errors.New(results[j].Error)
			continue
		}
		out[i].Content = []byte(results[j].Content)
	}
}
