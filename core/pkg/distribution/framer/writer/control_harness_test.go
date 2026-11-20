// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package writer_test

import (
	"context"
	"sync"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/x/confluence"
	xcontrol "github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	xtime "github.com/synnaxlabs/x/time"
)

// The control specs ran against Cesium writers before control moved to the framer.
// testDB and storageWriter give framer writers the Cesium API those specs use, so their
// assertions run unchanged.

// testDB is a Cesium DB whose writers are framer writers. It holds the records of the
// virtual channels, which Cesium does not store, and resolves them for the framer.
type testDB struct {
	*cesium.DB
	svc     *writer.Service
	writes  *confluence.Stream[relay.Response]
	virtual map[channel.Key]channel.Channel
	mu      sync.Mutex
}

// storageConfig is the configuration of a Cesium writer before control moved to the
// framer.
type storageConfig struct {
	ErrOnUnauthorized *bool
	ControlSubject    xcontrol.Subject
	Channels          []cesium.ChannelKey
	Authorities       []xcontrol.Authority
	Start             telem.TimeStamp
	Mode              writer.Mode
}

// controlUpdate is a batch of control transfers in Cesium keys.
type controlUpdate struct {
	Transfers []xcontrol.Transfer[cesium.ChannelKey]
}

var testKeys atomic.Uint32

// GenerateChannelKey returns a channel key that no other spec uses.
func GenerateChannelKey() cesium.ChannelKey {
	return uint32(channel.NewKey(1, channel.LocalKey(60_000+testKeys.Add(1))))
}

// mustOpenDBOnFS opens a Cesium DB on fsys with a writer service over it. The DB closes
// when the spec or container that opens it ends.
func mustOpenDBOnFS(ctx context.Context, fsys fs.FS) *testDB {
	GinkgoHelper()
	db := DeferClose(MustSucceed(cesium.Open(ctx, "", cesium.WithFS(fsys))))
	StartRun(db.Run)
	writes := confluence.NewStream[relay.Response](10_000)
	svc := MustSucceed(writer.NewService(writer.ServiceConfig{
		TS:     db,
		Writes: writes,
		Clock:  xtime.Real,
	}))
	tdb := &testDB{
		DB:      db,
		svc:     svc,
		writes:  writes,
		virtual: make(map[channel.Key]channel.Channel),
	}
	svc.SetChannelResolver(tdb)
	return tdb
}

// CreateChannel creates the stored channels of chs in Cesium and keeps the records of
// the virtual ones.
func (db *testDB) CreateChannel(ctx context.Context, chs ...cesium.Channel) error {
	var stored []cesium.Channel
	db.mu.Lock()
	for _, ch := range chs {
		if !ch.Virtual {
			stored = append(stored, ch)
			continue
		}
		key := channel.Key(ch.Key)
		db.virtual[key] = channel.Channel{
			Name:        ch.Name,
			Leaseholder: key.Lease(),
			LocalKey:    key.LocalKey(),
			DataType:    ch.DataType,
			Virtual:     true,
		}
	}
	db.mu.Unlock()
	if len(stored) == 0 {
		return nil
	}
	return db.DB.CreateChannel(ctx, stored...)
}

// Resolve implements writer.ChannelResolver.
func (db *testDB) Resolve(
	_ context.Context,
	keys channel.Keys,
) (map[channel.Key]writer.Facts, error) {
	db.mu.Lock()
	defer db.mu.Unlock()
	facts := make(map[channel.Key]writer.Facts, len(keys))
	for _, k := range keys {
		if r, ok := db.virtual[k]; ok {
			facts[k] = writer.FactsOf(r)
		}
	}
	return facts, nil
}

// OpenWriter opens a framer writer on the channels of cfg.
func (db *testDB) OpenWriter(
	ctx context.Context,
	cfg storageConfig,
) (*storageWriter, error) {
	// The writer runs until it closes, as a Cesium writer did, not until the node that
	// opens it ends.
	w, err := db.svc.Open(context.WithoutCancel(ctx), writer.Config{
		ControlSubject:    cfg.ControlSubject,
		Keys:              channel.KeysFromUint32(cfg.Channels),
		Start:             cfg.Start,
		Authorities:       cfg.Authorities,
		ErrOnUnauthorized: cfg.ErrOnUnauthorized,
		Mode:              cfg.Mode,
		EnableAutoCommit:  new(false),
		Sync:              new(true),
	})
	if err != nil {
		return nil, err
	}
	db.mu.Lock()
	virtual := lo.CountBy(cfg.Channels, func(k cesium.ChannelKey) bool {
		_, ok := db.virtual[channel.Key(k)]
		return ok
	})
	db.mu.Unlock()
	relays := 0
	if cfg.Mode != writer.ModePersistOnly {
		relays = lo.Ternary(virtual > 0, 1, 0) +
			lo.Ternary(virtual < len(cfg.Channels), 1, 0)
	}
	return &storageWriter{
		w:       w,
		writes:  db.writes,
		relays:  relays,
		virtual: virtual == len(cfg.Channels),
	}, nil
}

// Write writes fr through a writer of its own that starts at start, as cesium.DB.Write
// did before control moved to the framer.
func (db *testDB) Write(
	ctx context.Context,
	start telem.TimeStamp,
	fr cesium.Frame,
) error {
	w, err := db.OpenWriter(ctx, storageConfig{Start: start, Channels: fr.KeysSlice()})
	if err != nil {
		return err
	}
	_, err = w.Write(ctx, fr)
	if err == nil || errors.Is(err, xcontrol.ErrUnauthorized) {
		_, commitErr := w.Commit(ctx)
		err = errors.Join(err, commitErr)
	}
	return errors.Join(err, w.Close())
}

// ControlStates returns the state of control over each controlled channel.
func (db *testDB) ControlStates() controlUpdate {
	return controlUpdate{Transfers: lo.Map(
		db.svc.ControlStates(),
		func(s writer.ControlState, _ int) xcontrol.Transfer[cesium.ChannelKey] {
			return xcontrol.Transfer[cesium.ChannelKey]{To: toStorageState(&s)}
		},
	)}
}

// OnControlUpdate calls handler with each batch of control transfers.
func (db *testDB) OnControlUpdate(handler func(controlUpdate)) observe.Disconnect {
	return db.svc.OnControlUpdate(func(_ context.Context, u writer.ControlUpdate) {
		handler(controlUpdate{Transfers: lo.Map(
			u.Transfers,
			func(
				t xcontrol.Transfer[channel.Key],
				_ int,
			) xcontrol.Transfer[cesium.ChannelKey] {
				return xcontrol.Transfer[cesium.ChannelKey]{
					From: toStorageState(t.From),
					To:   toStorageState(t.To),
				}
			},
		)})
	})
}

func toStorageState(s *writer.ControlState) *xcontrol.State[cesium.ChannelKey] {
	if s == nil {
		return nil
	}
	return &xcontrol.State[cesium.ChannelKey]{
		Subject:   s.Subject,
		Resource:  uint32(s.Resource),
		Authority: s.Authority,
	}
}

// storageWriter gives a framer writer the API of a Cesium writer: a refusal for no
// control is an error that wraps control.ErrUnauthorized.
type storageWriter struct {
	w      *writer.Writer
	writes *confluence.Stream[relay.Response]
	// relays is the number of frames the writer streams for each write: one for its
	// stored channels and one for its virtual channels.
	relays int
	// virtual is true when the writer holds only virtual channels, which commit
	// nothing.
	virtual bool
}

// Write writes fr and returns the frame the writer streams.
func (w *storageWriter) Write(
	_ context.Context,
	fr cesium.Frame,
) (cesium.Frame, error) {
	authorized, err := w.w.Write(frame.NewFromStorage(fr))
	if err != nil {
		return cesium.Frame{}, err
	}
	var written frame.Frame
	for range w.relays {
		res := <-w.writes.Outlet()
		written = written.Extend(res.Frame)
	}
	if !authorized {
		return written.ToStorage(), errors.Wrap(
			xcontrol.ErrUnauthorized,
			"write refused",
		)
	}
	return written.ToStorage(), nil
}

// Commit commits the channels the writer controls. A commit that only follows a refused
// write succeeds, as it did in Cesium: the writer returns its end. A writer of virtual
// channels commits nothing, so its commit is never refused.
func (w *storageWriter) Commit(context.Context) (telem.TimeStamp, error) {
	end, authorized, err := w.w.Commit()
	if err != nil {
		return 0, err
	}
	if !authorized && end == 0 && !w.virtual {
		return 0, errors.Wrap(xcontrol.ErrUnauthorized, "commit refused")
	}
	return end, nil
}

// SetAuthority sets the authority of the writer over the channels of cfg.
func (w *storageWriter) SetAuthority(cfg storageConfig) error {
	return w.w.SetAuthority(writer.Config{
		Keys:        channel.KeysFromUint32(cfg.Channels),
		Authorities: cfg.Authorities,
	})
}

// EndDomain ends the domains of indexes.
func (w *storageWriter) EndDomain(
	_ context.Context,
	indexes ...cesium.ChannelKey,
) error {
	authorized, err := w.w.EndDomain(channel.KeysFromUint32(indexes))
	if err != nil {
		return err
	}
	if !authorized {
		return errors.Wrap(xcontrol.ErrUnauthorized, "end domain refused")
	}
	return nil
}

// Close closes the writer.
func (w *storageWriter) Close() error { return w.w.Close() }

// expectUnauthorized asserts that a write was refused for lack of control. It does not
// check the frame, which holds the channels that the write did not refuse.
func expectUnauthorized(_ cesium.Frame, err error) {
	GinkgoHelper()
	Expect(err).To(MatchError(xcontrol.ErrUnauthorized))
}

// virtualChannel returns a virtual channel with the given key and name.
func virtualChannel(key cesium.ChannelKey, name string) cesium.Channel {
	return cesium.Channel{
		Key:      key,
		Name:     name,
		DataType: telem.Int64T,
		Virtual:  true,
	}
}
