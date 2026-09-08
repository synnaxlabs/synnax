// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium

import (
	"context"

	"github.com/synnaxlabs/cesium/internal/control"
	"github.com/synnaxlabs/x/observe"
)

type (
	// ControlState is the state of control over a single channel: the subject holding
	// authority over it, and the level of that authority.
	ControlState = control.State
	// ControlTransfer is a transition of control over a single channel. A nil From is
	// an acquire; a nil To is a release.
	ControlTransfer = control.Transfer
)

type ControlUpdate struct {
	Transfers []ControlTransfer `json:"transfers"`
}

// OnControlUpdate registers handler to receive every control transfer the DB
// arbitrates, in the order the DB applies them. Handlers run inline while the DB holds
// its read lock, so a handler must hand the update off: blocking on it stalls every
// writer, and calling back into the DB deadlocks it, since the second read lock cannot
// be taken while a writer waits. The returned function unregisters the handler.
func (db *DB) OnControlUpdate(
	handler func(context.Context, ControlUpdate),
) observe.Disconnect {
	return db.controlUpdates.OnChange(handler)
}

// ControlStates returns the leading control resource in each unary and virtual channel
// in the Cesium database at the snapshot at which ControlStates is called: the
// controlState may change during the call.
func (db *DB) ControlStates() (u ControlUpdate) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	u.Transfers = make(
		[]control.Transfer,
		0,
		len(db.mu.dbs.unary)+len(db.mu.dbs.virtual),
	)
	for _, d := range db.mu.dbs.unary {
		if s := d.LeadingControlState(); s != nil {
			u.Transfers = append(u.Transfers, control.Transfer{To: s})
		}
	}
	for _, d := range db.mu.dbs.virtual {
		if s := d.LeadingControlState(); s != nil {
			u.Transfers = append(u.Transfers, control.Transfer{To: s})
		}
	}
	return u
}
