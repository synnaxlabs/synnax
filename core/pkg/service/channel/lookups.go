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
	"time"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/errors"
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

const (
	// compileTries is the number of compiles that CompileConsistent runs.
	compileTries = 5
	// compileBackoff is the wait after the first changed compile. It doubles after
	// each.
	compileBackoff = 10 * time.Millisecond
	// changedTries is the number of calls that RetryChanged makes.
	changedTries = 3
	// changedBackoff is the wait after the first ErrLookupsChanged that RetryChanged
	// sees. It doubles after each.
	changedBackoff = 100 * time.Millisecond
)

// ErrLookupsChanged is returned by CompileConsistent when a write changed what each of
// its compiles read.
var ErrLookupsChanged = errors.New(
	"channels or modules changed during each compile; try again",
)

// CompileConsistent returns the result of compile, which reads through snap and records
// each lookup it makes in lookups. It passes db, so a long compile delays no write, and
// then runs the lookups again in one short snapshot. A changed answer means that a
// write committed during the compile, so it waits and compiles again. After five
// changed tries, it returns ErrLookupsChanged.
func CompileConsistent[T any](
	ctx context.Context,
	db *gorp.DB,
	compile func(snap gorp.Snapshot, lookups *Lookups) (T, error),
) (T, error) {
	return retry(ctx, compileTries, compileBackoff, func() (T, error) {
		var zero T
		lookups := &Lookups{}
		res, err := compile(db, lookups)
		var unchanged bool
		if checkErr := db.WithSnapshot(ctx, func(snap gorp.Snapshot) (err error) {
			unchanged, err = lookups.Unchanged(ctx, snap)
			return err
		}); checkErr != nil {
			return zero, checkErr
		}
		if !unchanged {
			return zero, ErrLookupsChanged
		}
		return res, err
	})
}

// RetryChanged returns the result of f, calling f again after a growing wait each time
// it returns ErrLookupsChanged. After three such calls, it returns ErrLookupsChanged.
// The caller must hold no transaction, since writes must commit during the waits.
func RetryChanged[T any](ctx context.Context, f func() (T, error)) (T, error) {
	return retry(ctx, changedTries, changedBackoff, f)
}

// retry calls f up to tries times while it returns ErrLookupsChanged, waiting backoff
// before the second call and doubling the wait before each one after.
func retry[T any](
	ctx context.Context,
	tries int,
	backoff time.Duration,
	f func() (T, error),
) (T, error) {
	var zero T
	for try := range tries {
		if try > 0 {
			select {
			case <-ctx.Done():
				return zero, ctx.Err()
			case <-time.After(backoff):
			}
			backoff *= 2
		}
		res, err := f()
		if !errors.Is(err, ErrLookupsChanged) {
			return res, err
		}
	}
	return zero, ErrLookupsChanged
}
