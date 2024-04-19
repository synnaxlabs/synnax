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
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/cesium/internal/controller"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
)

type ControlUpdate struct {
	Transfers []controller.Transfer `json:"transfers"`
}

func (db *DB) ConfigureControlUpdateChannel(ctx context.Context, key ChannelKey) error {
	ch, err := db.RetrieveChannel(ctx, key)
	if errors.Is(err, core.ChannelNotFound) {
		ch.Key = key
		ch.DataType = telem.StringT
		ch.Virtual = true
		if err := db.CreateChannel(ctx, ch); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	db.digests.key = key
	w, err := db.NewStreamWriter(ctx, WriterConfig{
		ControlSubject: control.Subject{Name: "cesium_internal_control_digest"},
		Start:          telem.Now(),
		Channels:       []ChannelKey{key},
	})
	if err != nil {
		return err
	}
	db.digests.inlet, db.digests.outlet = confluence.Attach[WriterRequest, WriterResponse](w, 100)
	sCtx, _ := signal.Isolated()
	w.Flow(sCtx, confluence.CloseInletsOnExit())
	return nil
}

func (db *DB) updateControlDigests(
	ctx context.Context,
	u ControlUpdate,
) {
	if db.digests.key == 0 {
		return
	}
	db.digests.inlet.Inlet() <- WriterRequest{Command: WriterWrite, Frame: db.ControlUpdateToFrame(ctx, u)}
}

func (db *DB) closeControlDigests() {
	if db.digests.key != 0 {
		db.digests.key = 0
		db.digests.inlet.Close()
		confluence.Drain(db.digests.outlet)
	}
}

func (db *DB) ControlStates() (u ControlUpdate) {
	if db.digests.key == 0 {
		return
	}
	u.Transfers = make([]controller.Transfer, 0, len(db.unaryDBs)+len(db.virtualDBs))
	for _, d := range db.unaryDBs {
		s := d.LeadingControlState()
		if s != nil {
			u.Transfers = append(u.Transfers, controller.Transfer{To: s})
		}
	}
	for _, d := range db.virtualDBs {
		s := d.LeadingControlState()
		if s != nil {
			u.Transfers = append(u.Transfers, controller.Transfer{To: d.LeadingControlState()})
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
	s.Data, err = (&binary.JSONEncoderDecoder{}).Encode(ctx, u)
	return s, err
}

func DecodeControlUpdate(ctx context.Context, s telem.Series) (ControlUpdate, error) {
	var u ControlUpdate
	return u, (&binary.JSONEncoderDecoder{}).Decode(ctx, s.Data, &u)
}
