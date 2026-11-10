// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"context"
	"sync"

	"github.com/synnaxlabs/gorp"
)

// Lookups records the lookups that one compile makes, so that CompileConsistent can run
// them again. It is safe for concurrent use.
type Lookups struct {
	mu     sync.Mutex
	checks []func(context.Context, gorp.Snapshot) (bool, error)
}

// Record adds a lookup to l. check runs the lookup again in snap and reports whether it
// gives the same answer.
func (l *Lookups) Record(check func(context.Context, gorp.Snapshot) (bool, error)) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.checks = append(l.checks, check)
}

// Unchanged runs each lookup of l again in snap and reports whether every one gives the
// same answer. A lookup that fails returns its error.
func (l *Lookups) Unchanged(ctx context.Context, snap gorp.Snapshot) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, check := range l.checks {
		if same, err := check(ctx, snap); !same || err != nil {
			return false, err
		}
	}
	return true, nil
}

// compileTries is the number of compiles that CompileConsistent runs with no snapshot.
const compileTries = 3

// CompileConsistent returns the result of compile, which reads through snap and records
// each lookup it makes in lookups. It first passes db, so a long compile delays no
// write, and then runs the lookups again in one short snapshot. A changed answer means
// that a write committed during the compile, so it compiles again. After three changed
// tries, it compiles inside a snapshot, so compile must not write.
func CompileConsistent[T any](
	ctx context.Context,
	db *gorp.DB,
	compile func(snap gorp.Snapshot, lookups *Lookups) (T, error),
) (T, error) {
	for range compileTries {
		lookups := &Lookups{}
		res, err := compile(db, lookups)
		var unchanged bool
		if checkErr := db.WithSnapshot(ctx, func(snap gorp.Snapshot) (err error) {
			unchanged, err = lookups.Unchanged(ctx, snap)
			return err
		}); checkErr != nil {
			var zero T
			return zero, checkErr
		}
		if unchanged {
			return res, err
		}
	}
	var res T
	err := db.WithSnapshot(ctx, func(snap gorp.Snapshot) (err error) {
		res, err = compile(snap, &Lookups{})
		return err
	})
	return res, err
}
