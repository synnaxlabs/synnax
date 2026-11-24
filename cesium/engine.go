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
	"cmp"
	"context"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

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
	// compact rewrites the segments that pass the candidate test of the state.
	compact *compactor
	// limit is the budget of the background reads, which each owner of them shares.
	limit *limiter
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
	stored   set.Set[ChannelKey]
	shutdown sync.Once
	closeErr error
	pins     pins
	// partSize is the part size of the checkpointer.
	partSize int
	// cache is the part cache of the checkpointer and the readers.
	cache *checkpoint.Cache
	// parts holds the parts that the last two heads name, which the range of a Channel
	// may name. Only the state owner stores it, in the step that publishes the ranges.
	parts atomic.Pointer[partSets]
	// replaying is true while recovery replays the logs into the state.
	replaying bool
	// opened holds the parts of the head that open loaded, whose records name segment
	// files that the orphan pass keeps. It is nil after that pass.
	opened *checkpoint.Index
}

// partSets are the parts of the last two heads.
type partSets struct{ live, older headParts }

// headParts are the parts that the head of generation names, in rising order of
// identity.
type headParts struct {
	generation uint64
	parts      []checkpoint.Part
}

// openEngine recovers the database of the root of o, or makes a new one in an empty
// root. It refuses a root that holds the old layout with errors.ErrUnsupported.
func openEngine(o *options) (*engine, error) {
	r := &recovery{fs: o.fs}
	snapshot, err := r.loadHead()
	if err != nil {
		return nil, err
	}
	e := &engine{
		ins:      o.Instrumentation,
		fs:       o.fs,
		exec:     o.executor,
		ready:    make(chan struct{}),
		failed:   make(chan struct{}),
		unstored: telem.NewAlignmentCounter(telem.LeadingDomain),
		partSize: o.partSize,
	}
	e.pins.readers, e.pins.dead = make(map[uint64]int), make(map[uint64]state.Dead)
	e.pins.parts, e.pins.keys = make(map[uint64]int), make(map[ChannelKey]int)
	if snapshot == nil {
		if snapshot, err = e.create(o, r); err != nil {
			return nil, err
		}
	}
	e.db, e.head, e.opened = r.db, r.head.Generation, r.parts
	e.stored = set.New[ChannelKey]()
	for _, ch := range snapshot.Channels {
		e.stored.Add(ch.Key)
	}
	if e.state, err = state.Open(state.Config{
		Versions:   set.New[uint16](segmentVersion),
		Version:    segmentVersion,
		Threshold:  o.gcCfg.Threshold,
		Append:     e.append,
		Executor:   e.exec,
		OnHead:     e.onHead,
		Generation: r.head.Generation,
	}, snapshot); err != nil {
		return nil, err
	}
	// Replay preloads the part records that its records reach.
	e.cache = checkpoint.NewCache(e.fs, r.db, partCacheSize)
	e.parts.Store(&partSets{
		live: headParts{generation: r.head.Generation, parts: r.head.Parts},
	})
	if err = e.recover(r); err != nil {
		return nil, err
	}
	if e.budget, err = segment.NewBudget(handleLimit, syncWidth, e.exec); err != nil {
		return nil, err
	}
	e.limit = newLimiter(o.clock, o.gcCfg.Rate)
	e.compact = newCompactor(e, o.clock, e.limit)
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
	e.replaying = true
	err = r.replay(e.replay)
	e.replaying = false
	if err != nil {
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
		LogSize:  log.FileSize,
		Prepared: e.log.Prepared,
		Executor: e.exec,
		Parts:    r.parts,
		Live:     r.head.Generation,
		PartSize: e.partSize,
		Cache:    e.cache,
		Parted:   e.parted,
		Pinned:   e.pinned,
	})
	return err
}

// parted hands what a new head holds to the state, which makes its parts readable for
// the Channels that it publishes with their ranges.
func (e *engine) parted(
	generation uint64,
	parts []checkpoint.Part,
	tags []state.Tag,
	ranges []state.Range,
) {
	swap := func() {
		e.parts.Store(&partSets{
			live:  headParts{generation: generation, parts: parts},
			older: e.parts.Load().live,
		})
	}
	if err := e.state.Parted(generation, tags, ranges, swap); err != nil {
		e.ins.L.Error("state refused what a head holds", zap.Error(err))
	}
}

// partsOf returns the parts that p, the parts of key, names, from the head that p
// names.
func (e *engine) partsOf(key ChannelKey, p *state.Parts) ([]checkpoint.Part, error) {
	if p == nil {
		return nil, nil
	}
	sets := e.parts.Load()
	var head headParts
	switch p.Generation {
	case sets.live.generation:
		head = sets.live
	case sets.older.generation:
		head = sets.older
	default:
		return nil, errors.Newf(
			"key %d names the parts of head %d, which the engine no longer holds",
			key,
			p.Generation,
		)
	}
	lo, found := slices.BinarySearchFunc(head.parts, p.Oldest, compareParts)
	hi, foundNewest := slices.BinarySearchFunc(head.parts, p.Newest, compareParts)
	if !found || !foundNewest {
		return nil, errors.Newf(
			"key %d names parts %d to %d, which head %d does not name",
			key,
			p.Oldest,
			p.Newest,
			p.Generation,
		)
	}
	return head.parts[lo : hi+1], nil
}

func compareParts(p checkpoint.Part, id uint64) int { return cmp.Compare(p.ID, id) }

func (e *engine) append(entry *log.Entry) { e.log.Append(entry) }

// durable hands a durable entry to the state, and signals the compactor for a record
// that may make a candidate. A refusal fails the state, whose later calls return the
// error, so the hook only logs it.
func (e *engine) durable(sequence uint64, records []log.Record) {
	if err := e.state.Durable(sequence, records); err != nil {
		e.ins.L.Error(
			"state refused a durable log entry",
			zap.Uint64("sequence", sequence),
			zap.Error(err),
		)
	}
	for _, r := range records {
		switch r.(type) {
		case *log.Tombstone, *log.Seal, *log.Restart:
			e.compact.signal()
			return
		}
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

// reap unlinks the file of each segment that left the state, once no reader pins it.
func (e *engine) reap() error {
	dead, err := e.state.TakeDead()
	if err != nil {
		return err
	}
	return e.release(dead)
}

// run runs the owners until ctx is done, then closes them in order: the compactor, the
// log, so that its queued entries and a clean mark reach the file, then the
// checkpointer, then the state. It runs the orphan pass first.
func (e *engine) run(ctx context.Context) error {
	owners, cancel := signal.Isolated()
	defer cancel()
	owners.Go(e.state.Run, signal.CancelOnFail())
	owners.Go(e.log.Run, signal.CancelOnFail())
	owners.Go(e.ckpt.Run, signal.CancelOnFail())
	owners.Go(e.compact.run, signal.CancelOnFail())
	if e.switched != 0 {
		e.switchedTo(e.switched)
	}
	snapshot, err := e.state.Snapshot()
	if err == nil {
		orphaned := make(chan error, 1)
		e.exec(func() { orphaned <- e.orphans(snapshot) })
		err = <-orphaned
		e.opened = nil
	}
	close(e.ready)
	if err == nil {
		e.compact.start()
	}
	if err == nil {
		select {
		case <-ctx.Done():
		case <-owners.Done():
		case <-e.failed:
			err = e.failure
		}
	}
	return errors.Join(err, e.close(), errors.Skip(owners.Wait(), context.Canceled))
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
	var (
		pre     []state.Preload
		err     error
		created map[uint64]ChannelKey
	)
	for _, r := range records {
		switch r := r.(type) {
		case *log.Create:
			e.stored.Add(r.Key)
			if created == nil {
				created = make(map[uint64]ChannelKey)
			}
			created[r.Segment] = r.Key
		case *log.Commit:
			first, last := telem.TimeStamp(r.First), telem.TimeStamp(r.Last)
			for _, slot := range r.Slots {
				key, ok := e.state.KeyOf(slot.Segment)
				if !ok {
					key, ok = created[slot.Segment]
				}
				if ok {
					if pre, err = e.addPreload(
						pre,
						key,
						first,
						last,
						false,
					); err != nil {
						return err
					}
				}
			}
		case *log.Tombstone:
			start, end := telem.TimeStamp(r.Start), telem.TimeStamp(r.End)
			if pre, err = e.addPreload(pre, r.Key, start, end-1, true); err != nil {
				return err
			}
		case *log.Drop:
			e.stored.Remove(r.Key)
			pre, err = e.addPreload(
				pre,
				r.Key,
				telem.TimeStampMin,
				telem.TimeStampMax,
				true,
			)
			if err != nil {
				return err
			}
		}
	}
	slices.SortFunc(
		pre,
		func(a, b state.Preload) int { return cmp.Compare(a.Key, b.Key) },
	)
	return e.state.Replay(sequence, records, pre...)
}

// close closes the owners once.
func (e *engine) close() error {
	e.shutdown.Do(func() {
		e.compact.close()
		e.closeErr = errors.Join(e.log.Close(), e.ckpt.Close(), e.state.Close())
	})
	return e.closeErr
}

// partCacheSize is the budget of the part cache.
const partCacheSize = 64 << 20

// segmentName returns the file name of segment id of key.
func segmentName(key ChannelKey, id uint64) string {
	return fmt.Sprintf("%d/%016x.seg", key, id)
}

// orphans removes each segment and check file of a channel directory that snapshot does
// not name, and cuts each named segment file to its committed length. A segment file
// that a live record in the parts of the opened head names stays, unless snapshot lacks
// its key, which a Drop then took. The checkpointer cleans the root.
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
		_, stored := slices.BinarySearchFunc(
			snapshot.Channels,
			ChannelKey(key),
			func(ch *state.Channel, key ChannelKey) int { return cmp.Compare(ch.Key, key) },
		)
		if err = e.orphansOf(ChannelKey(key), named, stored); err != nil {
			return err
		}
	}
	return nil
}

func (e *engine) orphansOf(
	key ChannelKey,
	named map[string]state.Segment,
	stored bool,
) error {
	dir := strconv.FormatUint(uint64(key), 10)
	infos, err := e.fs.List(dir)
	if err != nil {
		return err
	}
	removed := false
	var parted map[uint64]state.Segment
	for _, info := range infos {
		name := dir + "/" + info.Name()
		if !strings.HasSuffix(name, ".seg") && !strings.HasSuffix(name, ".chk") {
			continue
		}
		seg, ok := named[name]
		if !ok && stored && e.opened != nil && strings.HasSuffix(name, ".seg") {
			if id, err := strconv.ParseUint(
				strings.TrimSuffix(info.Name(), ".seg"),
				16,
				64,
			); err == nil {
				if parted == nil {
					parted = e.opened.Live(key)
				}
				seg, ok = parted[id]
			}
		}
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
