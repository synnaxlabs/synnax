// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package writer

import (
	"sync"

	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/errors"
)

var (
	// ErrChannelInUse is returned by Service.ClaimDelete when a writer holds one of the
	// channels.
	ErrChannelInUse = errors.New("channel in use")
	// ErrChannelDeleting is returned when a writer opens on a channel that a delete
	// claims.
	ErrChannelDeleting = errors.New("channel is being deleted")
)

// claims counts the open writers and the delete claims of each channel with storage.
type claims struct {
	mu       sync.Mutex
	writers  map[channel.Key]int
	deleting map[channel.Key]int
}

func newClaims() *claims {
	return &claims{
		writers:  make(map[channel.Key]int),
		deleting: make(map[channel.Key]int),
	}
}

// hold records a writer on keys and returns the function that releases it. It fails
// when a delete claims one of keys.
func (c *claims) hold(keys channel.Keys) (release func(), err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, k := range keys {
		if c.deleting[k] > 0 {
			return nil, errors.Wrapf(ErrChannelDeleting, "channel %v", k)
		}
	}
	for _, k := range keys {
		c.writers[k]++
	}
	return c.releaser(c.writers, keys), nil
}

// claim records a delete claim on keys and returns the function that releases it. It
// fails when a writer holds one of keys.
func (c *claims) claim(keys channel.Keys) (release func(), err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, k := range keys {
		if open := c.writers[k]; open > 0 {
			return nil, errors.Wrapf(
				ErrChannelInUse,
				"there are %d unclosed writers on channel %v",
				open,
				k,
			)
		}
	}
	for _, k := range keys {
		c.deleting[k]++
	}
	return c.releaser(c.deleting, keys), nil
}

// releaser returns a function that decrements the count of each of keys in counts
// once.
func (c *claims) releaser(counts map[channel.Key]int, keys channel.Keys) func() {
	return sync.OnceFunc(func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		for _, k := range keys {
			if counts[k]--; counts[k] == 0 {
				delete(counts, k)
			}
		}
	})
}
