// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package segment

import (
	"sync"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// Executor runs f, at once or on another goroutine. It must run every f it gets.
type Executor func(f func())

// GoExecutor runs each f on a new goroutine.
func GoExecutor(f func()) { go f() }

// Pool syncs segment handles on its executor, at most its width at once across every
// caller. A Pool is safe for concurrent use.
type Pool struct {
	exec  Executor
	slots chan struct{}
}

// NewPool returns a pool of width syncs at once, at least 1, that runs each on exec.
func NewPool(width int, exec Executor) (*Pool, error) {
	if width < 1 || exec == nil {
		return nil, errors.Wrapf(
			validate.ErrValidation,
			"sync pool needs a width of at least 1 and an executor, not %d and %t",
			width,
			exec != nil,
		)
	}
	return &Pool{exec: exec, slots: make(chan struct{}, width)}, nil
}

// Sync syncs each handle of hs, and returns once every sync returned. It returns the
// errors of the failed syncs, in the order of hs.
func (p *Pool) Sync(hs []*Handle) error {
	var wg sync.WaitGroup
	errs := make([]error, len(hs))
	wg.Add(len(hs))
	for i, h := range hs {
		p.slots <- struct{}{}
		p.exec(func() {
			defer func() {
				<-p.slots
				wg.Done()
			}()
			errs[i] = h.Sync()
		})
	}
	wg.Wait()
	return errors.Join(errs...)
}
