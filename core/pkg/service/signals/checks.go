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

	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// checkCommits is how many commits of checks a change channel holds. A streamer that
// falls further behind sends none of their samples.
const checkCommits = 64

// commitChecks holds the checks that a change channel made at publish for the samples
// of its recent commits, so that each streamer reads a check rather than the record. It
// holds the last checkCommits commits with checks. It is safe for concurrent use.
type commitChecks[I comparable] struct {
	mu sync.Mutex
	// commits holds the checks of each commit, oldest first.
	commits []map[I]Check
}

func (c *commitChecks[I]) add(checks map[I]Check) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.commits) == checkCommits {
		copy(c.commits, c.commits[1:])
		c.commits = c.commits[:len(c.commits)-1]
	}
	c.commits = append(c.commits, checks)
}

// get returns the check of the sample named id, or a check that refuses every subject
// when the check is gone.
func (c *commitChecks[I]) get(id I) Check {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, checks := range slices.Backward(c.commits) {
		if check, ok := checks[id]; ok {
			return check
		}
	}
	return refuse
}

// getSample is get for checks named by the bytes of their samples. It does not copy
// sample.
func getSample(c *commitChecks[string], sample []byte) Check {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, checks := range slices.Backward(c.commits) {
		if check, ok := checks[string(sample)]; ok {
			return check
		}
	}
	return refuse
}

func refuse(func(resource.ID) bool) bool { return false }
