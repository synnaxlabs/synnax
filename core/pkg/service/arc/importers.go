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
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/arc/directory"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"go.uber.org/zap"
)

// pathOf returns the import path of a in snap: the names of its directories below Arcs,
// then its own name.
func (s *Service) pathOf(
	ctx context.Context,
	snap gorp.Snapshot,
	a Arc,
) (string, error) {
	segments := []string{a.Base().Name}
	parent := a.Base().Directory
	for {
		if parent == nil {
			return "", errors.Newf("arc %s is not below Arcs", a.Base().Key)
		}
		if *parent == s.root {
			break
		}
		var d directory.Directory
		if err := s.directories.NewRetrieve().
			Where(directory.MatchKeys(*parent)).
			Entry(&d).
			Exec(ctx, snap); err != nil {
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
func (s *Service) importsOf(a Arc) []string {
	src, ok := a.Base().Source.Variant.(TextSource)
	if !ok {
		return nil
	}
	cfg := parser.Config{AllowDashedNames: s.AllowDashedNames()}
	prog, diag := parser.Parse(src.Text.Materialize().Raw, cfg)
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

// importersOf returns the key of every Arc in snap that imports the module at path,
// directly or through another module, each once.
func (s *Service) importersOf(
	ctx context.Context,
	snap gorp.Snapshot,
	path string,
) ([]Key, error) {
	var arcs []Arc
	if err := s.table.NewRetrieve().Entries(&arcs).Exec(ctx, snap); err != nil {
		return nil, err
	}
	byPath := make(map[string][]Arc)
	for _, a := range arcs {
		for _, p := range s.importsOf(a) {
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
			modulePath, err := s.pathOf(ctx, snap, a)
			if err != nil {
				return nil, err
			}
			queue = append(queue, modulePath)
		}
	}
	return found, nil
}

// resyncImporting schedules a resync of every Arc with an import, since a path change
// may resolve or break any of them. It runs on the path-sync debouncer and reads
// committed state with no lock.
func (s *Service) resyncImporting(ctx context.Context, _ struct{}) {
	var arcs []Arc
	if err := s.table.NewRetrieve().Entries(&arcs).Exec(ctx, s.cfg.DB); err != nil {
		if failed(err) {
			s.cfg.L.Error("failed to resync importing arcs", zap.Error(err))
		}
		return
	}
	for _, a := range arcs {
		if len(s.importsOf(a)) > 0 {
			s.taskSync.Trigger(a.Base().Key)
		}
	}
}
