// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package state

// AuthorityChange represents a buffered authority change request.
// It is produced by set_authority nodes during reactive execution and consumed
// by the runtime task when flushing state.
type AuthorityChange struct {
	// Channel is the specific channel to change authority for.
	// If nil, the change applies to all write channels.
	Channel *uint32
	// Authority is the new authority value (0-255).
	Authority uint8
}

// State buffers authority change requests produced during
// reactive execution for later flushing.
type State struct {
	changes []AuthorityChange
}

// Set buffers an authority change request.
// If channelKey is nil, the change applies to all write channels.
func (b *State) Set(channelKey *uint32, authority uint8) {
	b.changes = append(b.changes, AuthorityChange{
		Channel:   channelKey,
		Authority: authority,
	})
}

// Flush returns and clears all buffered authority changes.
func (b *State) Flush() []AuthorityChange {
	if len(b.changes) == 0 {
		return nil
	}
	changes := b.changes
	b.changes = nil
	return changes
}
