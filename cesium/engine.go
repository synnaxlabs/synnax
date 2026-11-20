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
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/synnaxlabs/cesium/internal/checkpoint"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/segment"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
)

const (
	// segmentVersion is the format version of new segments.
	segmentVersion = 1
	// handleLimit is the most segment files that the database holds open.
	handleLimit = 1024
	// syncWidth is the most segment and directory syncs that run at once.
	syncWidth = 32
)

// engine is the pointer log layout of a DB: the state owner, the log owner, the
// checkpointer, and the budget of the segment file handles.
type engine struct {
	fs     xfs.FS
	db     uint64
	exec   func(f func())
	state  *state.State
	log    *log.Log
	ckpt   *checkpoint.Checkpointer
	budget *segment.Budget
	// head is the generation of the head that recovery used.
	head uint64
	// switched is the generation of a log that recovery made live, whose head the
	// checkpointer writes once it runs, or zero.
	switched uint64
	// ready is closed once the orphan pass is done. A writer waits for it before it
	// creates a segment file.
	ready    chan struct{}
	shutdown sync.Once
	closeErr error
}

// openEngine recovers the database of the root of o, or makes a new one in an empty
// root. It refuses a root that holds the old layout with ErrOldLayout.
func openEngine(o *options) (*engine, error) {
	r := &recovery{fs: o.fs}
	old, err := r.list()
	if err != nil {
		return nil, err
	}
	if old && len(r.heads) == 0 {
		return nil, errors.Wrap(
			ErrOldLayout,
			"the pointer log cannot open it until the conversion exists",
		)
	}
	snapshot, err := r.pickHead()
	if err != nil {
		return nil, err
	}
	e := &engine{fs: o.fs, exec: o.executor, ready: make(chan struct{})}
	if snapshot == nil {
		if snapshot, err = e.create(o, r); err != nil {
			return nil, err
		}
	}
	e.db, e.head = r.db, r.head.Generation
	if e.state, err = state.Open(state.Config{
		Versions: set.New[uint16](segmentVersion),
		Version:  segmentVersion,
		Append:   e.append,
		Executor: e.exec,
	}, snapshot); err != nil {
		return nil, err
	}
	if err = e.recover(r); err != nil {
		return nil, err
	}
	if e.budget, err = segment.NewBudget(handleLimit, syncWidth, e.exec); err != nil {
		return nil, err
	}
	return e, nil
}

// create makes a new database in a root with no head that passes: it removes the head
// and log files that a cut-short create left, draws the identity, and writes head 1
// over the file of log 1, whose header the log writes at its start.
func (e *engine) create(o *options, r *recovery) (*state.Snapshot, error) {
	for _, g := range r.heads {
		if err := o.fs.Remove(checkpoint.Name(g)); err != nil {
			return nil, err
		}
	}
	for _, g := range r.logs {
		if err := o.fs.Remove(log.Name(g)); err != nil {
			return nil, err
		}
	}
	if err := o.fs.SyncDir(""); err != nil {
		return nil, err
	}
	db := o.random.Uint64()
	for db == 0 {
		db = o.random.Uint64()
	}
	if err := log.Prepare(o.fs, 1, log.FileSize); err != nil {
		return nil, err
	}
	snapshot := &state.Snapshot{}
	if err := checkpoint.Write(
		o.fs,
		1,
		checkpoint.Encode(db, 1, snapshot),
	); err != nil {
		return nil, err
	}
	r.db, r.head = db, checkpoint.Header{DB: db, Generation: 1}
	r.heads, r.logs = []uint64{1}, []uint64{1}
	return snapshot, nil
}

// recover replays the logs into the state, writes again what recovery read, logs a
// Restart after an unclean stop, and opens the log where recovery left it.
func (e *engine) recover(r *recovery) (err error) {
	if err = r.replay(e.state.Replay); err != nil {
		return err
	}
	if err = r.rewrite(); err != nil {
		return err
	}
	sequence, records, switched, err := r.restart()
	if err != nil {
		return err
	}
	if records != nil {
		if err = e.state.Replay(sequence, records); err != nil {
			return err
		}
	}
	if switched {
		e.switched = r.gen
	}
	cfg := log.Config{
		FS:         e.fs,
		Executor:   e.exec,
		OnDurable:  e.durable,
		Prepare:    e.prepare,
		Seals:      e.seals,
		Switched:   e.switchedTo,
		DB:         r.db,
		Generation: r.gen,
		Sequence:   r.first,
	}
	if r.fresh {
		e.log, err = log.Start(cfg)
	} else {
		e.log, err = log.Resume(cfg, r.pos)
	}
	if err != nil {
		return err
	}
	e.ckpt, err = checkpoint.New(checkpoint.Config{
		FS:       e.fs,
		DB:       r.db,
		LogSize:  log.FileSize,
		Snapshot: e.state.Snapshot,
		Prepared: e.log.Prepared,
		Executor: e.exec,
	})
	return err
}

func (e *engine) append(entry *log.Entry) { e.log.Append(entry) }

// durable hands a durable entry to the state. A refusal fails the state, whose later
// calls return the error.
func (e *engine) durable(sequence uint64, records []log.Record) {
	_ = e.state.Durable(sequence, records)
}

func (e *engine) prepare(next uint64) { e.ckpt.Prepare(next) }

func (e *engine) switchedTo(generation uint64) { e.ckpt.Switched(generation) }

func (e *engine) seals() ([]log.Record, error) {
	seals, err := e.state.SealUnheld()
	if err != nil {
		return nil, err
	}
	records := make([]log.Record, len(seals))
	for i := range seals {
		records[i] = &seals[i]
	}
	return records, nil
}

// run runs the owners until ctx is done, then closes them in order: the log, so that
// its queued entries and a clean mark reach the file, then the checkpointer, then the
// state. It runs the orphan pass first.
func (e *engine) run(ctx context.Context) error {
	owners, cancel := signal.Isolated()
	defer cancel()
	owners.Go(e.state.Run)
	owners.Go(e.log.Run)
	owners.Go(e.ckpt.Run)
	if e.switched != 0 {
		e.ckpt.Switched(e.switched)
	}
	err := e.orphans()
	close(e.ready)
	if err == nil {
		select {
		case <-ctx.Done():
		case <-owners.Done():
		}
	}
	return errors.Join(err, e.close(), errors.Skip(owners.Wait(), context.Canceled))
}

// close closes the owners once.
func (e *engine) close() error {
	e.shutdown.Do(func() {
		e.closeErr = errors.Join(e.log.Close(), e.ckpt.Close(), e.state.Close())
	})
	return e.closeErr
}

// segmentName returns the file name of segment id of key.
func segmentName(key ChannelKey, id uint64) string {
	return fmt.Sprintf("%d/%016x.seg", key, id)
}

// orphans removes each segment and check file of a channel directory that the state
// does not name, and cuts each named segment file to its committed length. In the
// root, it keeps what a switch keeps.
func (e *engine) orphans() error {
	snapshot, err := e.state.Snapshot()
	if err != nil {
		return err
	}
	named := make(map[string]state.Segment)
	for _, ch := range snapshot.Channels {
		for _, seg := range ch.Segments {
			named[segmentName(ch.Key, seg.ID)] = seg
		}
	}
	infos, err := e.fs.List("")
	if err != nil {
		return err
	}
	for _, info := range infos {
		key, err := strconv.ParseUint(info.Name(), 10, 32)
		if err != nil || !info.IsDir() {
			continue
		}
		if err = e.orphansOf(ChannelKey(key), named); err != nil {
			return err
		}
	}
	return checkpoint.Clean(e.fs, e.head)
}

func (e *engine) orphansOf(key ChannelKey, named map[string]state.Segment) error {
	dir := strconv.FormatUint(uint64(key), 10)
	infos, err := e.fs.List(dir)
	if err != nil {
		return err
	}
	removed := false
	for _, info := range infos {
		name := dir + "/" + info.Name()
		if !strings.HasSuffix(name, ".seg") && !strings.HasSuffix(name, ".chk") {
			continue
		}
		seg, ok := named[name]
		if !ok {
			if err = e.fs.Remove(name); err != nil {
				return err
			}
			removed = true
			continue
		}
		if end := e.segmentConfig(seg).
			FileOffset(int64(seg.Length)); info.Size() > end {
			if err = e.cut(name, end); err != nil {
				return err
			}
		}
	}
	if removed {
		return e.fs.SyncDir(dir)
	}
	return nil
}

// cut cuts the file name to size. The bytes past the committed length of a segment are
// never data, so the cut needs no sync.
func (e *engine) cut(name string, size int64) error {
	f, err := e.fs.Open(name, os.O_RDWR)
	if err != nil {
		return err
	}
	return errors.Join(f.Truncate(size), f.Close())
}

// segmentConfig returns the identity and block layout of seg.
func (e *engine) segmentConfig(seg state.Segment) segment.Config {
	return segment.Config{
		DB:       e.db,
		ID:       seg.ID,
		DataType: seg.DataType,
		Clock:    seg.Clock,
		Variable: telem.DataTypeCode(seg.DataType).DataType().IsVariable(),
	}
}
