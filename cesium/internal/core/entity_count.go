// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package core

import "sync"

type entityCount struct {
	sync.RWMutex
	openIterators  int
	openWriters    int
	totalWriters   uint32
	totalIterators uint32
}

func (c *entityCount) addWriter() (uint32, func()) {
	c.Lock()
	c.openWriters += 1
	c.totalWriters += 1
	c.Unlock()
	return c.totalWriters, func() {
		c.Lock()
		c.openWriters -= 1
		c.Unlock()
	}
}

func (c *entityCount) addIterator() func() {
	c.Lock()
	c.openIterators += 1
	c.totalIterators += 1
	c.Unlock()
	return c.totalIterators, func() {
		c.openIterators -= 1
	}
}
