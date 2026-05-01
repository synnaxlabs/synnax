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
	"time"

	"github.com/synnaxlabs/oracle/format"
	"github.com/synnaxlabs/oracle/pipeline"
)

// CacheGate validates that the sync cache is internally consistent with
// what the pipeline currently produces.
//
// The sync cache stores SHA-256 of the *raw* (pre-format) plugin bytes
// keyed by repo-relative path. On the next sync, if the same path has
// the same raw hash, sync skips formatting and writing entirely under
// the assumption that the on-disk file must already be canonical.
//
// That assumption holds only when nothing has poisoned the cache. A
// poisoned cache - cache entry says "fresh" but the on-disk file has
// been edited by hand or written by an older formatter - causes sync to
// silently leave broken output in place. This gate catches that:
//
//   - For every produced file, compare cache.LookupRaw against the
//     hash of the *current* raw plugin output. A mismatch is just a
//     cache miss (sync will reformat next run); not a finding.
//   - For every produced file where the cache *does* match, verify the
//     on-disk file is non-empty and exists. A cache hit on a missing
//     file is poisoned: sync will skip and never repair.
//
// The gate cannot validate the on-disk *bytes* without re-running the
// formatter chain, which is what the generated gate already does. So
// this gate stays cheap and focuses on the failure mode the generated
// gate cannot catch: cache says "skip" on a path that doesn't exist or
// is empty.
type CacheGate struct {
	cache *format.Cache
}

// NewCacheGate constructs the gate. nil cache makes the gate a no-op
// (skip-style), used by callers that did not load a cache.
func NewCacheGate(cache *format.Cache) *CacheGate {
	return &CacheGate{cache: cache}
}

func (CacheGate) Name() string { return "cache" }

func (g CacheGate) Run(_ context.Context, p *pipeline.Result, env Env) GateReport {
	start := time.Now()
	r := GateReport{Gate: g.Name(), Status: StatusPass}
	if g.cache == nil {
		r.Status = StatusSkipped
		r.Elapsed = time.Since(start)
		return r
	}

	for _, files := range p.Outputs {
		for _, f := range files {
			rawHash := format.Hash(f.Content)
			cached, hit := g.cache.LookupRaw(f.Path)
			if !hit {
				continue
			}
			if cached != rawHash {
				continue
			}
			abs := filepath.Join(env.RepoRoot, f.Path)
			info, err := os.Stat(abs)
			if err != nil {
				r.Findings = append(r.Findings, Finding{
					Path:     f.Path,
					Severity: SeverityError,
					Message:  "cache says fresh but file does not exist on disk",
					FixHint:  "delete .oracle/sync-cache.json and re-run `oracle sync`",
				})
				continue
			}
			if info.Size() == 0 {
				r.Findings = append(r.Findings, Finding{
					Path:     f.Path,
					Severity: SeverityError,
					Message:  "cache says fresh but file is empty",
					FixHint:  "delete .oracle/sync-cache.json and re-run `oracle sync`",
				})
			}
		}
	}
	if len(r.Findings) > 0 {
		r.Status = StatusFail
	}
	r.Elapsed = time.Since(start)
	return r
}
