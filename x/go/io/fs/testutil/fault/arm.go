// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package fault

// Arm gives the disk of f the sync errors and evictions of cfg from now on: its
// SyncErrorRate, EvictionRate, and SyncErrorAt, which counts from this call. The error
// matches validate.ErrValidation as the error of New does.
func (f *FS) Arm(cfg Config) error {
	if err := cfg.validate(); err != nil {
		return err
	}
	d := f.d
	d.mu.Lock()
	defer d.mu.Unlock()
	d.cfg.SyncErrorRate, d.cfg.EvictionRate = cfg.SyncErrorRate, cfg.EvictionRate
	d.cfg.SyncErrorAt, d.syncs = cfg.SyncErrorAt, 0
	return nil
}

// Fill sets the capacity of the disk of f to the bytes that it holds now plus room.
func (f *FS) Fill(room int64) {
	d := f.d
	d.mu.Lock()
	defer d.mu.Unlock()
	d.capacity, d.capped = d.used()+max(room, 0), true
}

// Unsynced reports whether the disk of f holds a change that a crash can lose in part:
// a block of a file written since its last Sync, or a directory entry that no SyncDir
// made durable.
func (f *FS) Unsynced() bool {
	d := f.d
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.pending) > 0 || d.root.unsynced()
}

func (n *node) unsynced() bool {
	if !n.dir {
		return len(n.dirty) > 0
	}
	for _, e := range n.entries {
		if e.unsynced() {
			return true
		}
	}
	return false
}
