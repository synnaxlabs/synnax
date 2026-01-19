// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package signal

// Census allows the caller to examine the state of the signal context.
type Census interface {
	// Routines returns a slice of RoutineInfo for all routines forked by the
	// context.
	Routines() []RoutineInfo
}

// Routines implement the Census interface.
func (c *core) Routines() []RoutineInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.routines()
}

func (c *core) routines() []RoutineInfo {
	info := make([]RoutineInfo, len(c.mu.routines))
	for i, r := range c.mu.routines {
		info[i] = r.info()
	}
	return info
}
