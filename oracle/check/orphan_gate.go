// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package check

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/synnaxlabs/oracle/format"
	"github.com/synnaxlabs/oracle/pipeline"
	"github.com/synnaxlabs/x/set"
)

// OrphanMarker is the substring oracle puts in every generated filename
// to mark it as machine-managed. The orphan walk uses this to distinguish
// generated artifacts from hand-written code that happens to live in the
// same directory.
const OrphanMarker = ".gen."

// OrphanGate detects generated files that exist on disk but no schema
// produces any longer. Two signals drive detection:
//
//   - The sync cache: every previously-synced output path is recorded.
//     If the cache knows about a path the current pipeline run did not
//     emit, that path is orphaned. The cache catches "this schema used
//     to produce types.gen.go but no longer does."
//
//   - A walk of every directory containing at least one current output:
//     any file whose basename contains `.gen.` and is not in the
//     produced set is reported. This catches files written before the
//     cache existed and files copied around without updating the cache.
//
// The two signals are unioned, so a file caught by either is reported
// once. The fix hint always points at sync, which is responsible for
// pruning - though sync today only prunes the cache, not the disk;
// removing that asymmetry is a separate change.
type OrphanGate struct {
	cache *format.Cache
	// Ignores is a set of basename glob patterns the gate skips during
	// the directory walk. Use it to declare paths owned by another
	// generation system (today: the migrate plugin, which has its own
	// command and is not part of the regular check pipeline). Cache
	// signals also skip ignored paths.
	//
	// Examples: "migrate_auto.gen.*", "*.frozen.gen.go".
	Ignores []string
}

// NewOrphanGate constructs the gate. The cache is the same one sync
// reads/writes; passing nil disables the cache signal.
func NewOrphanGate(cache *format.Cache) *OrphanGate {
	return &OrphanGate{cache: cache}
}

// WithIgnores returns a copy of the gate with the supplied basename glob
// patterns added to the Ignores list.
func (g *OrphanGate) WithIgnores(patterns ...string) *OrphanGate {
	cp := *g
	cp.Ignores = append(append([]string(nil), g.Ignores...), patterns...)
	return &cp
}

func (g OrphanGate) ignored(rel string) bool {
	base := filepath.Base(rel)
	for _, pat := range g.Ignores {
		if ok, _ := filepath.Match(pat, base); ok {
			return true
		}
	}
	return false
}

func (OrphanGate) Name() string { return "orphans" }

func (g OrphanGate) Run(_ context.Context, p *pipeline.Result, env Env) GateReport {
	start := time.Now()
	r := GateReport{Gate: g.Name(), Status: StatusPass}

	produced := set.New[string]()
	dirs := set.New[string]()
	for _, files := range p.Outputs {
		for _, f := range files {
			produced.Add(f.Path)
			dirs.Add(filepath.Dir(f.Path))
		}
	}

	orphans := set.New[string]()

	// Signal 1: cache references a path that no longer appears in the
	// produced set. The cache only learns paths that successfully synced
	// in a prior run, so any cache entry not in produced is by
	// definition stale.
	if g.cache != nil {
		for _, key := range g.cache.EntryKeys() {
			if produced.Contains(key) {
				continue
			}
			if g.ignored(key) {
				continue
			}
			orphans.Add(key)
		}
	}

	// Signal 2: walk every directory that currently has at least one
	// produced output (no recursion - a `.gen.` file under a sub
	// directory only counts as managed if that sub directory itself
	// has a current output). This is what keeps migration directories,
	// which sit under managed dirs but have their own generation
	// pipeline, from being misclassified as orphans of the regular
	// sync pipeline.
	for d := range dirs {
		entries, err := os.ReadDir(filepath.Join(env.RepoRoot, d))
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			if !strings.Contains(entry.Name(), OrphanMarker) {
				continue
			}
			rel := filepath.ToSlash(filepath.Join(d, entry.Name()))
			if produced.Contains(rel) {
				continue
			}
			if g.ignored(rel) {
				continue
			}
			orphans.Add(rel)
		}
	}

	for path := range orphans {
		// If a path is in the cache but the on-disk file no longer
		// exists (someone deleted it manually), don't flag it - that
		// is just a dirty cache, which the cache gate handles.
		abs := filepath.Join(env.RepoRoot, path)
		if _, err := os.Stat(abs); err != nil {
			continue
		}
		r.Findings = append(r.Findings, Finding{
			Path:     path,
			Severity: SeverityError,
			Message:  "generated file is no longer produced by any schema",
			FixHint:  "delete the file or restore the schema; then run `oracle sync`",
		})
	}
	if len(r.Findings) > 0 {
		r.Status = StatusFail
	}
	r.Elapsed = time.Since(start)
	return r
}
