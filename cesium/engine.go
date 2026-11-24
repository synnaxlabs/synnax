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
	"path"
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
	// check gives the version 0 segments their check files, and toCheck holds the keys
	// that the orphan pass hands it.
	check   *checker
	toCheck set.Set[ChannelKey]
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
	// loaded is true when open loaded a head. With the identity check, the caller then
	// stored the identity of the head, and the orphan pass removes the leftovers of the
	// old layout.
	loaded   bool
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
// root. It converts a root of the old layout that holds no head that passes.
func openEngine(o *options) (*engine, error) {
	r := &recovery{fs: o.fs}
	old, snapshot, err := r.loadHead(o)
	if err != nil {
		return nil, err
	}
	e, err := newEngine(o, snapshot != nil)
	if err != nil {
		return nil, err
	}
	created := snapshot == nil
	generation := r.head.Generation
	if created {
		if snapshot, err = e.create(o, r, old); err != nil {
			return nil, err
		}
		generation = 1
	}
	// The state loads a created snapshot before head 1 commits it, so a conversion that
	// breaks a rule leaves the old layout.
	if e.state, err = openState(o, e, generation, snapshot); err != nil {
		return nil, err
	}
	var head1 *checkpoint.Index
	if created {
		if head1, err = commit(o, r, snapshot); err != nil {
			return nil, err
		}
	}
	e.db, e.head, e.opened = r.db, r.head.Generation, r.parts
	e.stored = set.New[ChannelKey]()
	for _, ch := range snapshot.Channels {
		e.stored.Add(ch.Key)
	}
	// Replay preloads the part records that its records reach.
	e.cache = checkpoint.NewCache(e.fs, r.db, partCacheSize)
	e.parts.Store(&partSets{
		live: headParts{generation: r.head.Generation, parts: r.head.Parts},
	})
	if head1 != nil {
		// The state holds every record of the created snapshot. It tags the ones that
		// the parts of head 1 hold, as for the parts of a switch.
		parts, tags, ranges := head1.Parted()
		e.parted(1, parts, tags, ranges)
	}
	if err = e.recover(r); err != nil {
		return nil, err
	}
	e.limit = newLimiter(o.clock, o.gcCfg.Rate)
	e.compact = newCompactor(e, o.clock, e.limit)
	e.check = newChecker(e, o.clock, e.limit, o.gcCfg.TryInterval)
	return e, nil
}

// openState opens the state of e from snapshot, the head of generation.
func openState(
	o *options,
	e *engine,
	generation uint64,
	snapshot *state.Snapshot,
) (*state.State, error) {
	return state.Open(state.Config{
		Versions:           set.New[uint16](0, segmentVersion),
		Version:            segmentVersion,
		Threshold:          o.gcCfg.Threshold,
		Append:             e.append,
		Version0Rewritable: o.version0Rewritable,
		Executor:           e.exec,
		OnHead:             e.onHead,
		Generation:         generation,
	}, snapshot)
}

// newEngine returns an engine with no state, which loaded marks as opened from a head.
func newEngine(o *options, loaded bool) (*engine, error) {
	e := &engine{
		ins:      o.Instrumentation,
		fs:       o.fs,
		exec:     o.executor,
		ready:    make(chan struct{}),
		failed:   make(chan struct{}),
		unstored: telem.NewAlignmentCounter(telem.LeadingDomain),
		partSize: o.partSize,
		loaded:   loaded,
	}
	e.pins.readers, e.pins.dead = make(map[uint64]int), make(map[uint64]state.Dead)
	e.pins.parts, e.pins.keys = make(map[uint64]int), make(map[ChannelKey]int)
	var err error
	e.budget, err = segment.NewBudget(handleLimit, syncWidth, e.exec)
	return e, err
}

// create starts a new database in a root with no head that passes: it removes the head
// and log files that a cut-short create left and draws the identity. In a root of the
// old layout, it returns the converted state and makes the file of log 2. Otherwise it
// returns an empty state.
func (e *engine) create(o *options, r *recovery, old bool) (*state.Snapshot, error) {
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
	r.db = db
	if !old {
		return &state.Snapshot{}, nil
	}
	snapshot, err := e.convert(o, db)
	if err != nil {
		return nil, err
	}
	return snapshot, log.Prepare(o.fs, 2, log.FileSize)
}

// commit writes head 1 of snapshot, and the parts of its sealed segments, over the file
// of log 1, whose header the log writes at its start. It returns the index of the
// parts.
func commit(
	o *options,
	r *recovery,
	snapshot *state.Snapshot,
) (*checkpoint.Index, error) {
	if err := log.Prepare(o.fs, 1, log.FileSize); err != nil {
		return nil, err
	}
	x, err := checkpoint.Create(o.fs, r.db, 1, snapshot, o.partSize)
	if err != nil {
		return nil, err
	}
	parts, _, _ := x.Parted()
	r.parts, r.head = x, checkpoint.Header{DB: r.db, Generation: 1, Parts: parts}
	r.heads, r.logs = []uint64{1}, []uint64{1}
	return x, nil
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
		case *log.Tombstone, *log.Seal, *log.Restart, *log.Check:
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

// run runs the owners until ctx is done, then closes them in order: the check pass, the
// compactor, the log, so that its queued entries and a clean mark reach the file, then
// the checkpointer, then the state. It runs the orphan pass first.
func (e *engine) run(ctx context.Context) error {
	owners, cancel := signal.Isolated()
	defer cancel()
	owners.Go(e.state.Run, signal.CancelOnFail())
	owners.Go(e.log.Run, signal.CancelOnFail())
	owners.Go(e.ckpt.Run, signal.CancelOnFail())
	owners.Go(e.compact.run, signal.CancelOnFail())
	owners.Go(e.check.run, signal.CancelOnFail())
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
		e.check.start(e.toCheck)
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
			pre, err = e.commitPreloads(pre, r, created)
		case *log.Tombstone:
			start, end := telem.TimeStamp(r.Start), telem.TimeStamp(r.End)
			pre, err = e.addPreload(pre, r.Key, start, end-1, true)
		case *log.Drop:
			e.stored.Remove(r.Key)
			pre, err = e.addPreload(
				pre,
				r.Key,
				telem.TimeStampMin,
				telem.TimeStampMax,
				true,
			)
		case *log.Check:
			// The record names no time range, so the preload holds every part record
			// of the key.
			pre, err = e.addPreload(
				pre,
				r.Key,
				telem.TimeStampMin,
				telem.TimeStampMax,
				false,
			)
		}
		if err != nil {
			return err
		}
	}
	slices.SortFunc(
		pre,
		func(a, b state.Preload) int { return cmp.Compare(a.Key, b.Key) },
	)
	return e.state.Replay(sequence, records, pre...)
}

// commitPreloads adds to pre the preload of the key of each slot of c over the range of
// c. created maps the segments that the entry of c creates to their keys.
func (e *engine) commitPreloads(
	pre []state.Preload,
	c *log.Commit,
	created map[uint64]ChannelKey,
) ([]state.Preload, error) {
	first, last := telem.TimeStamp(c.First), telem.TimeStamp(c.Last)
	for _, slot := range c.Slots {
		key, ok := e.state.KeyOf(slot.Segment)
		if !ok {
			key, ok = created[slot.Segment]
		}
		if !ok {
			continue
		}
		var err error
		if pre, err = e.addPreload(pre, key, first, last, false); err != nil {
			return nil, err
		}
	}
	return pre, nil
}

// close closes the owners once.
func (e *engine) close() error {
	e.shutdown.Do(func() {
		e.check.close()
		e.compact.close()
		e.closeErr = errors.Join(e.log.Close(), e.ckpt.Close(), e.state.Close())
	})
	return e.closeErr
}

// partCacheSize is the budget of the part cache.
const partCacheSize = 64 << 20

// segmentName returns the file name of segment id of key, or the name of old file n of
// key for a version 0 segment, whose n is not zero.
func segmentName(key ChannelKey, id uint64, n uint16) string {
	if n != 0 {
		return fmt.Sprintf("%d/%d.domain", key, n)
	}
	return fmt.Sprintf("%d/%016x.seg", key, id)
}

// openSegment returns the handle of the file of segment seg of key.
func (e *engine) openSegment(key ChannelKey, seg state.Segment) *segment.Handle {
	return e.budget.Open(e.segmentFS(seg.File), segmentName(key, seg.ID, seg.File))
}

// segmentFS returns the file system that opens the file of a segment whose old file is
// n: e.fs, or for a version 0 segment, oldFS of e.fs.
func (e *engine) segmentFS(n uint16) xfs.FS {
	if n == 0 {
		return e.fs
	}
	return oldFS{FS: e.fs}
}

// oldFS opens an old file N.domain that is missing from the file that an old
// compaction renamed it to, N.domain_temp. The first open that loads a head renames
// that file to N.domain at any time, so an open tries N.domain, then N.domain_temp,
// then N.domain once more, and one of them finds a file that exists under either name.
type oldFS struct{ xfs.FS }

func (f oldFS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if !errors.Is(err, os.ErrNotExist) {
		return file, err
	}
	if file, err = f.FS.Open(name+tempSuffix, flag); !errors.Is(err, os.ErrNotExist) {
		return file, err
	}
	return f.FS.Open(name, flag)
}

// removeSegment removes the file of segment id of key, whose old file is n. For a
// version 0 segment, it also removes its check file, and it tries the names of the old
// file in the order of oldFS, removing N.domain_temp only when N.domain is missing.
func (e *engine) removeSegment(key ChannelKey, id uint64, n uint16) error {
	name := segmentName(key, id, n)
	if n == 0 {
		return e.fs.Remove(name)
	}
	if err := e.removeOld(name); err != nil {
		return err
	}
	return e.fs.Remove(checkName(key, id))
}

// removeOld removes the old file name, or the file that an old compaction renamed it
// to when name is missing. The first open that loads a head can rename that file to
// name at any time, so it tries name once more.
func (e *engine) removeOld(name string) error {
	ok, err := e.fs.Exists(name)
	if err != nil {
		return err
	}
	if ok {
		return e.fs.Remove(name)
	}
	if err = e.fs.Remove(name + tempSuffix); err != nil {
		return err
	}
	return e.fs.Remove(name)
}

// The leftovers of the old layout in a channel directory. An old compaction of file N
// copies its live bytes to N.domain_gc, renames N.domain to N.domain_temp, and renames
// the copy to N.domain, with no sync, so either file can hold the only bytes of N that
// the old pointers match. An old truncate copies the bytes that it cuts to a file with
// the suffix .truncated.tmp, which it renames once whole.
const (
	tempSuffix         = "_temp"
	gcSuffix           = "_gc"
	truncatedTmpSuffix = ".truncated.tmp"
)

// orphans removes each segment file of a channel directory that snapshot does not name,
// and each check file of a segment that has no checks on, and cuts each named segment
// file to its committed length. A segment file that a live record in the parts of the
// opened head names stays, unless snapshot lacks its key, which a Drop then took. On an
// open that loaded a head, it also removes the leftovers of the old layout that hold no
// data, and moves each old file that the conversion read from its compaction file into
// place. It sets toCheck to the stored keys that hold more old files with bytes than
// check files, which the check pass reads. The checkpointer cleans the root.
func (e *engine) orphans(snapshot *state.Snapshot) error {
	named := make(map[string]state.Segment)
	for _, ch := range snapshot.Channels {
		for _, seg := range ch.Segments {
			named[segmentName(ch.Key, seg.ID, seg.File)] = seg
			if seg.Checked {
				named[checkName(ch.Key, seg.ID)] = seg
			}
		}
	}
	infos, err := e.fs.List("")
	if err != nil {
		return err
	}
	removed := false
	e.toCheck = set.New[ChannelKey]()
	for _, info := range infos {
		if e.loaded && info.IsDir() && strings.Contains(info.Name(), deletedDirSuffix) {
			if err = e.fs.Remove(info.Name()); err != nil {
				return err
			}
			removed = true
			continue
		}
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
	if removed {
		return e.fs.SyncDir("")
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
	d := &orphanDir{e: e, key: key, named: named, stored: stored}
	changed := false
	checks := 0
	for _, info := range infos {
		name := dir + "/" + info.Name()
		left, moved, err := e.leftover(d, name, infos)
		if err != nil {
			return err
		}
		changed = changed || moved
		if strings.HasSuffix(name, ".chk") {
			kept, err := e.orphanCheck(d, name, info)
			if err != nil {
				return err
			}
			checks += kept
			changed = changed || kept == 0
			continue
		}
		if left || !strings.HasSuffix(name, ".seg") {
			continue
		}
		seg, ok := d.segment(name, info)
		if !ok {
			if err = e.fs.Remove(name); err != nil {
				return err
			}
			changed = true
			continue
		}
		if end := e.segmentConfig(seg).
			FileOffset(int64(seg.Length)); info.Size() > end {
			if err = e.cut(name, end); err != nil {
				return err
			}
		}
	}
	// Each version 0 segment holds one old file, and each checked one a check file.
	if stored && oldFiles(infos) > checks {
		e.toCheck.Add(key)
	}
	if changed {
		return e.fs.SyncDir(dir)
	}
	return nil
}

// orphanCheck removes name, a check file of d with info, unless it belongs to a version
// 0 segment that the head names with its checks on. It returns 1 when it kept the file,
// and 0 when it removed it.
func (e *engine) orphanCheck(
	d *orphanDir,
	name string,
	info xfs.FileInfo,
) (int, error) {
	if d.checked(name, info) {
		return 1, nil
	}
	return 0, e.fs.Remove(name)
}

// oldFiles returns the number of old data files with bytes among infos, the files of a
// channel directory: each N.domain, and each N.domain_temp when N.domain is missing.
func oldFiles(infos []xfs.FileInfo) int {
	n := 0
	for _, info := range infos {
		base, temp := strings.CutSuffix(info.Name(), tempSuffix)
		num, ok := strings.CutSuffix(base, ".domain")
		if _, err := strconv.ParseUint(num, 10, 16); !ok || err != nil ||
			info.Size() == 0 {
			continue
		}
		if !temp || !slices.ContainsFunc(infos, func(other xfs.FileInfo) bool {
			return other.Name() == base
		}) {
			n++
		}
	}
	return n
}

// orphanDir is the channel directory of key in the orphan pass, with the segments of
// key that the head names.
type orphanDir struct {
	e     *engine
	key   ChannelKey
	named map[string]state.Segment
	// stored is true when the snapshot holds key.
	stored bool
	parted map[uint64]state.Segment
}

// live returns the segments of the key that the parts of the head that open loaded
// hold, or nil when the snapshot lacks the key.
func (d *orphanDir) live() map[uint64]state.Segment {
	if d.parted == nil && d.stored && d.e.opened != nil {
		d.parted = d.e.opened.Live(d.key)
	}
	return d.parted
}

// segment returns the segment whose file is name, a file of the directory with info.
func (d *orphanDir) segment(name string, info xfs.FileInfo) (state.Segment, bool) {
	if seg, ok := d.named[name]; ok {
		return seg, true
	}
	hex, ok := strings.CutSuffix(info.Name(), ".seg")
	if !ok {
		return state.Segment{}, false
	}
	id, err := strconv.ParseUint(hex, 16, 64)
	if err != nil {
		return state.Segment{}, false
	}
	seg, ok := d.live()[id]
	return seg, ok
}

// checked reports whether name, a check file of the directory with info, belongs to a
// version 0 segment that the head names with its checks on.
func (d *orphanDir) checked(name string, info xfs.FileInfo) bool {
	if _, ok := d.named[name]; ok {
		return true
	}
	hex, _ := strings.CutSuffix(info.Name(), ".chk")
	id, err := strconv.ParseUint(hex, 16, 64)
	if err != nil {
		return false
	}
	seg, ok := d.live()[id]
	return ok && seg.Checked
}

// namesOld reports whether the head names a version 0 segment whose old file is old.
func (d *orphanDir) namesOld(old string) bool {
	if _, ok := d.named[old]; ok {
		return true
	}
	n, err := strconv.ParseUint(strings.TrimSuffix(path.Base(old), ".domain"), 10, 16)
	if err != nil {
		return false
	}
	for _, seg := range d.live() {
		if seg.File == uint16(n) {
			return true
		}
	}
	return false
}

// leftover handles name when it is a leftover of the old layout among infos, the files
// of its directory. It reports whether name is one, and whether it changed the
// directory. On an open that loaded a head, it removes a truncate copy.
func (e *engine) leftover(
	d *orphanDir,
	name string,
	infos []xfs.FileInfo,
) (bool, bool, error) {
	switch {
	case strings.HasSuffix(name, truncatedTmpSuffix):
		if !e.loaded {
			return true, false, nil
		}
		return true, true, e.fs.Remove(name)
	case strings.HasSuffix(name, tempSuffix), strings.HasSuffix(name, gcSuffix):
		return true, e.oldCompaction(d, name, infos), nil
	}
	return false, false, nil
}

// oldCompaction handles name, a file of an old compaction among infos, the files of its
// directory d. On the open that converts, it logs the file, which it keeps. On an open
// that loaded a head, it renames N.domain_temp to N.domain when N.domain is missing and
// the head names N, and reports whether it did. It never removes such a file.
func (e *engine) oldCompaction(d *orphanDir, name string, infos []xfs.FileInfo) bool {
	old, temp := strings.CutSuffix(name, tempSuffix)
	missing := temp && !slices.ContainsFunc(infos, func(info xfs.FileInfo) bool {
		return info.Name() == path.Base(old)
	})
	if !e.loaded {
		e.ins.L.Warn(
			"kept a file of an old compaction, which can hold the bytes of an old file",
			zap.String("file", name),
			zap.Bool("read", missing),
		)
		return false
	}
	if !missing || !d.namesOld(old) {
		return false
	}
	// A reader can hold the file open, and Windows then refuses the rename. The next
	// open tries again, and oldFS finds the file under either name meanwhile.
	if err := e.fs.Rename(name, old); err != nil {
		e.ins.L.Warn(
			"failed to move the file that the conversion read for a missing old file "+
				"into place",
			zap.String("file", name),
			zap.Error(err),
		)
		return false
	}
	e.ins.L.Info(
		"moved the file that the conversion read for a missing old file into place",
		zap.String("file", name),
		zap.String("to", old),
	)
	return true
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

// checkName returns the name of the check file of version 0 segment id of key.
func checkName(key ChannelKey, id uint64) string {
	return fmt.Sprintf("%d/%016x.chk", key, id)
}

// readConfig returns the config of seg, a segment of key, for a read: with the checks
// of a version 0 segment that has its check file. It reads a check file that fails its
// checksum once more, then returns an error that wraps place.ErrCorrupt.
func (e *engine) readConfig(key ChannelKey, seg state.Segment) (segment.Config, error) {
	cfg := e.segmentConfig(seg)
	if !seg.Checked {
		return cfg, nil
	}
	var err error
	for range 2 {
		var b []byte
		if b, err = xfs.ReadFile(e.fs, checkName(key, seg.ID)); err != nil {
			return cfg, err
		}
		if cfg.Checks, err = cfg.DecodeChecks(b, int64(seg.Length)); err == nil {
			return cfg, nil
		}
	}
	return cfg, err
}

// segmentConfig returns the identity and block layout of seg.
func (e *engine) segmentConfig(seg state.Segment) segment.Config {
	return segment.Config{
		DB:       e.db,
		ID:       seg.ID,
		DataType: seg.DataType,
		Clock:    seg.Clock,
		Variable: telem.DataTypeCode(seg.DataType).DataType().IsVariable(),
		Bare:     seg.Version == 0,
	}
}

// storedAfter reports whether readers see a sample of key after stamp, in a segment of
// the state or in a record that a part holds.
func (e *engine) storedAfter(key ChannelKey, stamp telem.TimeStamp) (bool, error) {
	if ch := e.state.Channel(key); ch != nil && endsAfter(ch.Segments, stamp) {
		return true, nil
	}
	pre, err := e.records(key, stamp+1, telem.TimeStampMax)
	return err == nil && endsAfter(pre.Segments, stamp), err
}

// endsAfter reports whether a pointer of segs ends after stamp.
func endsAfter(segs []state.Segment, stamp telem.TimeStamp) bool {
	for _, seg := range segs {
		for _, p := range seg.Pointers {
			if p.Last > stamp {
				return true
			}
		}
	}
	return false
}
