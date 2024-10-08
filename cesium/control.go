// Copyright 2023 Synnax Labs, Inc.
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
	"github.com/google/uuid"
	"github.com/synnaxlabs/cesium/internal/controller"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
)

type ControlUpdate struct {
	Transfers []controller.Transfer `json:"transfers"`
}

// ConfigureControlUpdateChannel configures a channel to be the update channel for the
// database. If the channel is not found, it is created.
func (db *DB) ConfigureControlUpdateChannel(ctx context.Context, key ChannelKey) error {
	if db.closed.Load() {
		return errDBClosed
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	ch, err := db.retrieveChannel(ctx, key)
	if errors.Is(err, core.ErrChannelNotFound) {
		ch.Key = key
		ch.DataType = telem.StringT
		ch.Virtual = true
		if err = db.createChannel(ch); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	if ch.DataType != telem.StringT || !ch.Virtual {
		return errors.New("control update channel must be a string virtual channel.")
	}

	w, err := db.newStreamWriter(ctx, WriterConfig{
		ControlSubject: control.Subject{
			Name: "cesium_internal_control_digest",
			Key:  uuid.New().String(),
		},
		Start:    telem.Now(),
		Channels: []ChannelKey{key},
	})
	db.digests.key = key
	if err != nil {
		return err
	}
	db.digests.inlet, db.digests.outlet = confluence.Attach[WriterRequest, WriterResponse](w, 100)
	sCtx, _ := signal.Isolated()
	w.Flow(
		sCtx,
		confluence.CloseOutputInletsOnExit(),
		confluence.CancelOnFail(),
		confluence.RecoverWithErrOnPanic(),
	)
	return nil
}

func (db *DB) updateControlDigests(
	ctx context.Context,
	u ControlUpdate,
) error {
	if !db.digestsConfigured() {
		return nil
	}
	return signal.SendUnderContext(
		ctx,
		db.digests.inlet.Inlet(),
		WriterRequest{Command: WriterWrite, Frame: db.ControlUpdateToFrame(ctx, u)},
	)
}

func (db *DB) closeControlDigests() {
	// We need to do a careful mutex lock covers both the digests key check and
	// conditional re-set to 0. After that, we need to unlock the mutex so that the
	// control digest writer can be properly shut down, as it internally needs to
	// acquire a read lock on the mutex.
	db.mu.Lock()
	if db.digests.key == 0 {
		db.mu.Unlock()
		return
	}
	db.digests.key = 0
	db.mu.Unlock()
	db.digests.inlet.Close()
	confluence.Drain(db.digests.outlet)
	return
}

func (db *DB) digestsConfigured() bool { return db.digests.key != 0 }

// ControlStates returns the leading control entity in each unary and virtual channel
// in the Cesium database at the snapshot at which ControlStates is called: the
// controlState may change during the call.
func (db *DB) ControlStates() (u ControlUpdate) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	if !db.digestsConfigured() {
		return
	}
	u.Transfers = make([]controller.Transfer, 0, len(db.unaryDBs)+len(db.virtualDBs))
	for _, d := range db.unaryDBs {
		if s := d.LeadingControlState(); s != nil {
			u.Transfers = append(u.Transfers, controller.Transfer{To: s})
		}
	}
	for _, d := range db.virtualDBs {
		if s := d.LeadingControlState(); s != nil {
			u.Transfers = append(u.Transfers, controller.Transfer{To: s})
		}
	}
	return u
}

func (db *DB) ControlUpdateToFrame(ctx context.Context, u ControlUpdate) Frame {
	d, err := EncodeControlUpdate(ctx, u)
	if err != nil {
		panic(err)
	}
	return Frame{
		Keys:   []ChannelKey{db.digests.key},
		Series: []telem.Series{d},
	}
}

func EncodeControlUpdate(ctx context.Context, u ControlUpdate) (s telem.Series, err error) {
	s.DataType = telem.StringT
	s.Data, err = (&binary.JSONCodec{}).Encode(ctx, u)
	return s, err
}

func DecodeControlUpdate(ctx context.Context, s telem.Series) (ControlUpdate, error) {
	var u ControlUpdate
	return u, (&binary.JSONCodec{}).Decode(ctx, s.Data, &u)
}
