// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package version

// Heartbeat tracks information about the age of a process such as a node or thread.
// Heartbeat is useful for situations where two states must be compared and merged.
type Heartbeat struct {
	// Generation is incremented every time the process is restarted. This
	// value is typically persisted to disk in some fashion.
	Generation uint32
	// Version is every time the process alters its state. This value is ephemeral
	// and is reset to 0 when the process restarts.
	Version uint32
}

// Increment increments the Heartbeat.Version.
func (h Heartbeat) Increment() Heartbeat { h.Version++; return h }

// Decrement decrements Heartbeat.Version.
func (h Heartbeat) Decrement() Heartbeat { h.Version--; return h }

// Restart increments Heartbeat.Generation and resets Heartbeat.Version.
func (h Heartbeat) Restart() Heartbeat { h.Generation++; h.Version = 0; return h }

// OlderThan returns true if the Heartbeat generation or version is greater than other.
// It's important to note that an older heartbeat means a 'newer' version.
func (h Heartbeat) OlderThan(other Heartbeat) bool {
	return h.Generation > other.Generation ||
		(h.Generation == other.Generation && h.Version > other.Version)
}

// YoungerThan returns true if the Heartbeat generation or version is less than other.
// It's important to note that a younger heartbeat mans an 'older' version.
func (h Heartbeat) YoungerThan(other Heartbeat) bool {
	return h.Generation < other.Generation ||
		(h.Generation == other.Generation && h.Version < other.Version)
}
