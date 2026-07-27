// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package snapshot

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
)

// ErrAnalysis is returned by TableLoader loaders when a snapshot's schemas no
// longer analyze under the current grammar.
var ErrAnalysis = errors.New("snapshot analysis failed")

// TableLoader returns the latest snapshot version under repoRoot and a cached,
// concurrency-safe loader that analyzes a snapshot's schemas into a resolution
// table. The loader returns (nil, nil) when the snapshot does not exist and
// wraps ErrAnalysis when its schemas no longer parse under the current
// grammar.
func TableLoader(
	ctx context.Context,
	repoRoot string,
) (int, func(version int) (*resolution.Table, error), error) {
	snapshotsDir := filepath.Join(repoRoot, "schemas", "snapshots")
	latest, err := LatestVersion(snapshotsDir)
	if err != nil {
		return 0, nil, errors.Wrap(err, "failed to read snapshot version")
	}
	type entry struct {
		table *resolution.Table
		err   error
	}
	var (
		mu    sync.Mutex
		cache = make(map[int]*entry)
	)
	load := func(version int) (*resolution.Table, error) {
		mu.Lock()
		defer mu.Unlock()
		if e, ok := cache[version]; ok {
			return e.table, e.err
		}
		e := &entry{}
		cache[version] = e
		vDir := filepath.Join(snapshotsDir, fmt.Sprintf("v%d", version))
		files, err := Files(vDir)
		if err != nil || len(files) == 0 {
			return nil, nil
		}
		loader := NewFileLoader(vDir, repoRoot)
		normalized := make([]string, 0, len(files))
		for _, f := range files {
			rel, err := filepath.Rel(vDir, f)
			if err != nil {
				e.err = errors.Wrapf(err, "failed to compute relative path for %s", f)
				return nil, e.err
			}
			normalized = append(normalized, "schemas/"+strings.TrimSuffix(rel, ".oracle"))
		}
		t, diag := analyzer.Analyze(ctx, normalized, loader)
		if diag != nil && !diag.Ok() {
			e.err = errors.Wrapf(ErrAnalysis, "v%d", version)
			return nil, e.err
		}
		e.table = t
		return t, nil
	}
	return latest, load, nil
}
