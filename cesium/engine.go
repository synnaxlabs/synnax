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
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium/internal/checkpoint"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/segment"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
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
	ins    alamos.Instrumentation
	fs     xfs.FS
	db     uint64
	exec   func(f func())
	state  *state.State
	log    *log.Log
	ckpt   *checkpoint.Checkpointer
	budget *segment.Budget
	// handles holds the read handles of each segment file that reads opened, by key and
	// segment ID, until the file is removed or the engine closes.
	handles struct {
		sync.RWMutex
		m map[ChannelKey]map[uint64]*segmentHandles
	}
	// head is the generation of the head that recovery used.
	head uint64
	// switched is the generation of a log that recovery made live, whose head the
	// checkpointer writes once it runs, or zero.
	switched uint64
	// ready is closed once the orphan pass is done. A writer waits for it before it
	// creates a segment file, and lockRoot before a call that removes files.
	ready chan struct{}
	// unstored gives out the alignment numbers of unstored writers, from
	// telem.LeadingDomain.
	unstored *telem.AlignmentCounter
	// failed is closed with failure, the first fatal error of a caller, which ends Run.
	failed  chan struct{}
	failure error
	failing sync.Once
	// stored holds the keys that held a segment when recovery ended.
	stored set.Set[ChannelKey]
	// cover holds the dead segments whose files wait until the log covers the record
	// that took them out of the state, so no crash cuts that record while the file is
	// gone. Every entry below covered is covered.
	cover struct {
		sync.Mutex
		covered uint64
		dead    []state.Dead
		// advanced is closed and replaced each time covered rises.
		advanced chan struct{}
	}
	// reaps asks Run for a reap. A reap waits for a step of the state, so it never runs
	// on the executor that runs those steps.
	reaps    chan struct{}
	shutdown sync.Once
	closeErr error
}

// openEngine recovers the database of the root of o, or makes a new one in an empty
// root. It refuses a root that holds the old layout with errors.ErrUnsupported.
func openEngine(o *options) (*engine, error) {
	r := &recovery{fs: o.fs, logSize: o.logFileSize}
	old, err := r.list()
	if err != nil {
		return nil, err
	}
	if old && len(r.heads) == 0 {
		return nil, errors.Wrap(
			errors.ErrUnsupported,
			"the pointer log cannot open a root of the old layout",
		)
	}
	snapshot, err := r.pickHead()
	if err != nil {
		return nil, err
	}
	e := &engine{
		ins:      o.Instrumentation,
		fs:       o.fs,
		exec:     o.executor,
		ready:    make(chan struct{}),
		failed:   make(chan struct{}),
		reaps:    make(chan struct{}, 1),
		unstored: telem.NewAlignmentCounter(telem.LeadingDomain),
	}
	e.cover.advanced = make(chan struct{})
	e.handles.m = make(map[ChannelKey]map[uint64]*segmentHandles)
	if snapshot == nil {
		if snapshot, err = e.create(o, r); err != nil {
			return nil, err
		}
	}
	e.db, e.head = r.db, r.head.Generation
	e.stored = set.New[ChannelKey]()
	for _, ch := range snapshot.Channels {
		e.stored.Add(ch.Key)
	}
	if e.state, err = state.Open(state.Config{
		Versions: set.New[uint16](segmentVersion),
		Version:  segmentVersion,
		Append:   e.append,
		Executor: e.exec,
		OnHead:   e.onHead,
	}, snapshot); err != nil {
		return nil, err
	}
	if err = e.recover(r); err != nil {
		return nil, err
	}
	// Readers see the recovered state before Run.
	e.state.EndReplay()
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
	if err := log.Prepare(o.fs, 1, o.logFileSize); err != nil {
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
	if err = r.replay(e.replay); err != nil {
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
		OnRefused:  e.refused,
		OnCovered:  e.covered,
		Prepare:    e.prepare,
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
		LogSize:  r.logSize,
		Prepared: e.log.Prepared,
		Executor: e.exec,
	})
	return err
}

func (e *engine) append(entry *log.Entry) { e.log.Append(entry) }

// durable hands a durable entry to the state. A refusal fails the state, whose later
// calls return the error, so the hook only logs it.
func (e *engine) durable(sequence uint64, records []log.Record) {
	if err := e.state.Durable(sequence, records); err != nil {
		e.ins.L.Error(
			"state refused a durable log entry",
			zap.Uint64("sequence", sequence),
			zap.Error(err),
		)
	}
}

// refused hands an entry that the log refused to the state. The state fails the
// requests that wait for it, and returns no error of its own.
func (e *engine) refused(records []log.Record, err error) {
	if stateErr := e.state.Refused(records, err); stateErr != nil {
		e.ins.L.Error("state failed a refused log entry", zap.Error(stateErr))
	}
}

func (e *engine) prepare(next uint64) { e.ckpt.Prepare(next) }

// switchedTo hands a switch of the log to the state, which takes the snapshot of the
// head there. The state fails itself on an error, and its later calls return it.
func (e *engine) switchedTo(generation uint64) {
	if err := e.state.Switch(generation); err != nil {
		e.ins.L.Error(
			"state failed the switch to a log",
			zap.Uint64("generation", generation),
			zap.Error(err),
		)
	}
}

func (e *engine) onHead(
	generation uint64,
	s *state.Snapshot,
) {
	e.ckpt.Head(generation, s)
}

// reap unlinks the file of each segment that left the state once the log covers the
// record that took it out.
func (e *engine) reap() error {
	dead, err := e.state.TakeDead()
	if err != nil {
		return err
	}
	return e.unlink(e.uncover(0, dead))
}

// covered unlinks, off the step of the log, the file of each dead segment whose record
// the log now covers, and asks Run to take the segments that left the state since the
// last reap.
func (e *engine) covered(sequence uint64) {
	if free := e.uncover(sequence, nil); len(free) > 0 {
		e.exec(func() {
			if err := e.unlink(free); err != nil {
				e.ins.L.Error(
					"failed to unlink the file of a dead segment",
					zap.Error(err),
				)
			}
		})
	}
	if e.state.HasDead() {
		select {
		case e.reaps <- struct{}{}:
		default:
		}
	}
}

// uncover raises the covered sequence to covered, adds dead to the segments that wait,
// and returns those whose record the log covers.
func (e *engine) uncover(covered uint64, dead []state.Dead) []state.Dead {
	e.cover.Lock()
	defer e.cover.Unlock()
	if covered > e.cover.covered {
		e.cover.covered = covered
		close(e.cover.advanced)
		e.cover.advanced = make(chan struct{})
	}
	e.cover.dead = append(e.cover.dead, dead...)
	var free []state.Dead
	waiting := e.cover.dead[:0]
	for _, d := range e.cover.dead {
		if d.Sequence < e.cover.covered {
			free = append(free, d)
		} else {
			waiting = append(waiting, d)
		}
	}
	e.cover.dead = waiting
	return free
}

// waitCovered returns once the log covers the entry of sequence, or with the error that
// failed the engine, or with the error of ctx.
func (e *engine) waitCovered(ctx context.Context, sequence uint64) error {
	for {
		e.cover.Lock()
		covered, advanced := e.cover.covered, e.cover.advanced
		e.cover.Unlock()
		if sequence < covered {
			return nil
		}
		select {
		case <-advanced:
		case <-e.failed:
			return e.failure
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// unlink closes the read handles of each segment of dead and removes its file.
func (e *engine) unlink(dead []state.Dead) error {
	for _, d := range dead {
		if err := errors.Join(
			e.closeHandle(d.Key, d.Segment),
			e.fs.Remove(segmentName(d.Key, d.Segment)),
		); err != nil {
			return err
		}
	}
	return nil
}

// segmentHandles holds the read handles of one segment file. A read takes one that no
// other read holds, so parallel reads never share a file.
type segmentHandles struct {
	name string
	mu   sync.Mutex
	idle []*segment.Handle
	all  []*segment.Handle
}

// take returns a handle of s that no read holds, and opens a new one through b when
// every handle is in use.
func (s *segmentHandles) take(b *segment.Budget, fs xfs.FS) *segment.Handle {
	s.mu.Lock()
	defer s.mu.Unlock()
	if n := len(s.idle); n > 0 {
		h := s.idle[n-1]
		s.idle = s.idle[:n-1]
		return h
	}
	h := b.Open(fs, s.name)
	s.all = append(s.all, h)
	return h
}

// give returns h, which take returned, for another read.
func (s *segmentHandles) give(h *segment.Handle) {
	s.mu.Lock()
	s.idle = append(s.idle, h)
	s.mu.Unlock()
}

// close closes every handle of s. It waits for the reads that hold them.
func (s *segmentHandles) close() error {
	s.mu.Lock()
	all := s.all
	s.all, s.idle = nil, nil
	s.mu.Unlock()
	var err error
	for _, h := range all {
		err = errors.Join(err, h.Close())
	}
	return err
}

// readHandles returns the read handles of segment id of key.
func (e *engine) readHandles(key ChannelKey, id uint64) *segmentHandles {
	e.handles.RLock()
	hs := e.handles.m[key][id]
	e.handles.RUnlock()
	if hs != nil {
		return hs
	}
	e.handles.Lock()
	defer e.handles.Unlock()
	segs := e.handles.m[key]
	if segs == nil {
		segs = make(map[uint64]*segmentHandles)
		e.handles.m[key] = segs
	}
	if hs = segs[id]; hs == nil {
		hs = &segmentHandles{name: segmentName(key, id)}
		segs[id] = hs
	}
	return hs
}

// closeHandle closes the read handles of segment id of key, if reads opened any. It
// waits for the reads that hold them.
func (e *engine) closeHandle(key ChannelKey, id uint64) error {
	e.handles.Lock()
	hs := e.handles.m[key][id]
	delete(e.handles.m[key], id)
	e.handles.Unlock()
	if hs == nil {
		return nil
	}
	return hs.close()
}

// closeHandles closes the read handles of the segments of keys. It waits for the reads
// that hold them.
func (e *engine) closeHandles(keys ...ChannelKey) error {
	var sets []*segmentHandles
	e.handles.Lock()
	for _, k := range keys {
		for _, hs := range e.handles.m[k] {
			sets = append(sets, hs)
		}
		delete(e.handles.m, k)
	}
	e.handles.Unlock()
	var err error
	for _, hs := range sets {
		err = errors.Join(err, hs.close())
	}
	return err
}

// run runs the owners until ctx is done, then closes them in order: the log, so that
// its queued entries and a clean mark reach the file, then the checkpointer, then the
// state. It runs the orphan pass first.
func (e *engine) run(ctx context.Context) error {
	owners, cancel := signal.Isolated()
	defer cancel()
	owners.Go(e.state.Run, signal.CancelOnFail())
	owners.Go(e.log.Run, signal.CancelOnFail())
	owners.Go(e.ckpt.Run, signal.CancelOnFail())
	if e.switched != 0 {
		e.switchedTo(e.switched)
	}
	snapshot, err := e.state.Snapshot()
	if err == nil {
		orphaned := make(chan error, 1)
		e.exec(func() { orphaned <- e.orphans(snapshot) })
		err = <-orphaned
	}
	close(e.ready)
	if err == nil {
		err = e.serve(ctx, owners)
	}
	return errors.Join(err, e.close(), errors.Skip(owners.Wait(), context.Canceled))
}

// serve reaps each time covered asks, until ctx is done, an owner stops, or a caller
// fails the engine.
func (e *engine) serve(ctx context.Context, owners signal.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-owners.Done():
			return nil
		case <-e.failed:
			return e.failure
		case <-e.reaps:
			if err := e.reap(); err != nil {
				e.ins.L.Error(
					"failed to unlink the file of a dead segment",
					zap.Error(err),
				)
			}
		}
	}
}

// fail ends Run with err, when no fatal error ended it first.
func (e *engine) fail(err error) {
	e.failing.Do(func() {
		e.failure = err
		close(e.failed)
	})
}

// replay applies the records of one recovered entry to the state, and keeps stored
// current.
func (e *engine) replay(sequence uint64, records []log.Record) error {
	for _, r := range records {
		switch r := r.(type) {
		case *log.Create:
			e.stored.Add(r.Key)
		case *log.Drop:
			e.stored.Remove(r.Key)
		}
	}
	return e.state.Replay(sequence, records)
}

// close closes the owners once.
func (e *engine) close() error {
	e.shutdown.Do(func() {
		e.closeErr = errors.Join(
			e.log.Close(),
			e.ckpt.Close(),
			e.state.Close(),
			e.closeAllHandles(),
		)
	})
	return e.closeErr
}

// closeAllHandles closes the read handles of every segment.
func (e *engine) closeAllHandles() error {
	e.handles.RLock()
	keys := slices.Collect(maps.Keys(e.handles.m))
	e.handles.RUnlock()
	return e.closeHandles(keys...)
}

// segmentName returns the file name of segment id of key.
func segmentName(key ChannelKey, id uint64) string {
	return fmt.Sprintf("%d/%016x.seg", key, id)
}

// orphans removes each segment and check file of a channel directory that snapshot does
// not name, and cuts each named segment file to its committed length. In the root, it
// keeps what a switch keeps.
func (e *engine) orphans(snapshot *state.Snapshot) error {
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
			// No read opens a file that the state does not name, so no read handle
			// needs a close first.
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

// storedAfter reports whether readers see a sample of key after stamp.
func (e *engine) storedAfter(key ChannelKey, stamp telem.TimeStamp) bool {
	ch := e.state.Channel(key)
	if ch == nil {
		return false
	}
	for _, seg := range ch.Segments {
		for _, p := range seg.Pointers {
			if p.Last > stamp {
				return true
			}
		}
	}
	return false
}
