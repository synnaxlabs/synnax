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

// checkCommits is how many commits of checks a change channel holds. It is more than
// the frames that the relay holds for a streamer, which drops the frames past them, so
// a streamer finds the check of each sample it receives.
const checkCommits = 256

// errCheckGone is the error of a stream whose sample has no check, which fails the
// stream rather than send or drop the sample.
var errCheckGone = errors.New("[signals] - the check of a change is gone")

// commitChecks holds the checks that a change channel made at publish for the samples
// of its recent commits, by the bytes of each sample, so that each streamer reads a
// check rather than the record. It holds the last checkCommits commits with checks. It
// is safe for concurrent use.
type commitChecks struct {
	mu sync.Mutex
	// commits holds the checks of each commit, oldest first.
	commits []map[string]Check
}

func (c *commitChecks) add(checks map[string]Check) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.commits) == checkCommits {
		copy(c.commits, c.commits[1:])
		c.commits = c.commits[:len(c.commits)-1]
	}
	c.commits = append(c.commits, checks)
}

// get returns the check of sample, or errCheckGone. It does not copy sample.
func (c *commitChecks) get(sample []byte) (Check, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, checks := range slices.Backward(c.commits) {
		if check, ok := checks[string(sample)]; ok {
			return check, nil
		}
	}
	return nil, errCheckGone
}
