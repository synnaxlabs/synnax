// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"context"
	"slices"
	"strings"

	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/synnax/pkg/service/arc/directory"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// pathOf returns the import path of a: the names of its directories below Arcs, then
// its own name.
func (w Writer) pathOf(ctx context.Context, a Arc) (string, error) {
	segments := []string{a.Base().Name}
	parent := a.Base().Directory
	for {
		if parent == nil {
			return "", errors.Newf("arc %s is not below Arcs", a.Base().Key)
		}
		if *parent == w.root {
			break
		}
		var d directory.Directory
		if err := w.directories.NewRetrieve().
			Where(directory.MatchKeys(*parent)).
			Entry(&d).
			Exec(ctx, w.tx); err != nil {
			return "", err
		}
		segments = append(segments, d.Name)
		parent = d.Parent
	}
	slices.Reverse(segments)
	return strings.Join(segments, "."), nil
}

// importsOf returns the import paths of a. A graph Arc or a text that does not parse
// has none.
func (w Writer) importsOf(a Arc) []string {
	src, ok := a.Base().Source.Variant.(TextSource)
	if !ok {
		return nil
	}
	prog, diag := parser.Parse(src.Text.Materialize().Raw, w.parser)
	if diag != nil {
		return nil
	}
	entries := parser.Imports(prog)
	paths := make([]string, len(entries))
	for i, entry := range entries {
		paths[i] = entry.Path
	}
	return paths
}

// importersOf returns the key of every Arc that imports the module at path, directly
// or through another module, each once.
func (w Writer) importersOf(ctx context.Context, path string) ([]Key, error) {
	var arcs []Arc
	if err := w.table.NewRetrieve().Entries(&arcs).Exec(ctx, w.tx); err != nil {
		return nil, err
	}
	byPath := make(map[string][]Arc)
	for _, a := range arcs {
		for _, p := range w.importsOf(a) {
			byPath[p] = append(byPath[p], a)
		}
	}
	var (
		found []Key
		seen  = set.New[Key]()
		queue = []string{path}
	)
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		for _, a := range byPath[p] {
			key := a.Base().Key
			if seen.Contains(key) {
				continue
			}
			seen.Add(key)
			found = append(found, key)
			if a.Type() != TypeModule {
				continue
			}
			modulePath, err := w.pathOf(ctx, a)
			if err != nil {
				return nil, err
			}
			queue = append(queue, modulePath)
		}
	}
	return found, nil
}

// resyncImporting schedules a resync of every Arc with an import once w.tx commits. A
// path change may resolve or break any of them.
func (w Writer) resyncImporting(ctx context.Context) error {
	var arcs []Arc
	if err := w.table.NewRetrieve().Entries(&arcs).Exec(ctx, w.tx); err != nil {
		return err
	}
	var keys []Key
	for _, a := range arcs {
		if len(w.importsOf(a)) > 0 {
			keys = append(keys, a.Base().Key)
		}
	}
	w.resyncs.Add(w.tx, keys...)
	return nil
}
