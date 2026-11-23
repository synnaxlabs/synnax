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
	// commits is a ring of the checks of each commit. Once it is full, next is the slot
	// of the oldest commit, which the next commit takes.
	commits []map[string]Check
	next    int
	size    int
}

func newCommitChecks(size int) *commitChecks { return &commitChecks{size: size} }

func (c *commitChecks) add(checks map[string]Check) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.commits) < c.size {
		c.commits = append(c.commits, checks)
		return
	}
	c.commits[c.next] = checks
	c.next = (c.next + 1) % c.size
}

// get returns the check of sample, or errCheckGone. It does not copy sample.
func (c *commitChecks) get(sample []byte) (Check, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	n := len(c.commits)
	for i := range n {
		if check, ok := c.commits[(c.next-1-i+n)%n][string(sample)]; ok {
			return check, nil
		}
	}
	return nil, errCheckGone
}
