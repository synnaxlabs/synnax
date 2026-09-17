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

	"github.com/synnaxlabs/oracle/format"
	"github.com/synnaxlabs/oracle/pipeline"
	"github.com/synnaxlabs/x/telem"
)

// CacheGate validates that the sync cache is internally consistent with what is
// actually on disk.
//
// The sync cache stores two hashes per generated file: the raw plugin output and the
// canonical post-format output. On the next sync, if the raw hash matches AND the
// on-disk file hashes to the cached canonical value, sync skips the file. The
// canonical-hash check is what makes the cache safe across formatter version bumps and
// hand edits; without it sync would silently keep stale bytes in place.
//
// This gate proves the cache is not lying: for every produced file where the cache
// contains an entry, it verifies (1) the file exists and (2) the on-disk bytes hash to
// the cached canonical value. A mismatch means the cache says "skip" on a file whose
// contents no longer match what sync claimed it wrote, which would let stale output
// survive a sync with no warning.
type cacheGate struct{ cache *format.Cache }

// NewCacheGate returns a Checker that reports the cache coherent. A nil cache skips
// the gate, for a caller that loaded none.
func NewCacheGate(cache *format.Cache) Checker { return cacheGate{cache: cache} }

func (cacheGate) Name() string { return "cache" }

func (g cacheGate) Run(_ context.Context, res *pipeline.Result, env Env) GateReport {
	start := telem.Now()
	r := GateReport{Gate: g.Name(), Status: StatusPass}
	if g.cache == nil {
		r.Status = StatusSkipped
		r.Elapsed = telem.Since(start)
		return r
	}

	for _, files := range res.Outputs {
		for _, f := range files {
			entry, hit := g.cache.Lookup(f.Path)
			if !hit {
				continue
			}
			if entry.Raw != format.Hash(f.Content) {
				continue
			}
			abs := filepath.Join(env.RepoRoot, f.Path)
			existing, err := os.ReadFile(abs)
			if err != nil {
				r.Findings = append(r.Findings, Finding{
					Path:     f.Path,
					Severity: SeverityError,
					Message:  "cache says fresh but file does not exist on disk",
					FixHint:  "delete .oracle/sync-cache.json and re-run `oracle sync`",
				})
				continue
			}
			if format.Hash(existing) != entry.Canonical {
				r.Findings = append(r.Findings, Finding{
					Path:     f.Path,
					Severity: SeverityError,
					Message:  "on-disk content does not match cached canonical hash; sync would skip this file",
					FixHint:  "delete .oracle/sync-cache.json and re-run `oracle sync`",
				})
			}
		}
	}
	if len(r.Findings) > 0 {
		r.Status = StatusFail
	}
	r.Elapsed = telem.Since(start)
	return r
}
