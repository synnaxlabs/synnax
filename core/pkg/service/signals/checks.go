// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package signals

import (
	"slices"
	"sync"

	"github.com/synnaxlabs/x/errors"
)

// errCheckGone is the error of a stream whose sample has no check, which fails the
// stream rather than send or drop the sample.
var errCheckGone = errors.New("[signals] - the check of a change is gone")

// commitChecks holds the checks that a change channel made at publish for the samples
// of its recent commits, by the bytes of each sample, so that each streamer reads a
// check rather than the record. It holds the last size commits with checks. It is safe
// for concurrent use.
type commitChecks struct {
	mu sync.Mutex
	// commits holds the checks of each commit, oldest first.
	commits []map[string]check
	size    int
}

func newCommitChecks(size int) *commitChecks { return &commitChecks{size: size} }

// add adds the checks of one commit and drops the oldest commit past the size. It keeps
// the bytes of each key, so the caller must never write them again.
func (c *commitChecks) add(checks map[string]check) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.commits) == c.size {
		copy(c.commits, c.commits[1:])
		c.commits = c.commits[:len(c.commits)-1]
	}
	c.commits = append(c.commits, checks)
}

// get returns the check of sample, or errCheckGone. It does not copy sample.
func (c *commitChecks) get(sample []byte) (check, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, checks := range slices.Backward(c.commits) {
		if check, ok := checks[string(sample)]; ok {
			return check, nil
		}
	}
	return nil, errCheckGone
}
