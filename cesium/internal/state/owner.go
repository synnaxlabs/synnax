// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package state

import (
	"context"
	"maps"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/invariants"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// errClosed is the error of a call after the owner stopped.
var errClosed = errors.New("state is closed")

// Config is the configuration of a state.
type Config struct {
	// Versions are the segment format versions that this binary reads.
	Versions set.Set[uint16]
	// Version is the segment format version of new segments, one of Versions.
	Version uint16
	// Append queues an entry for the log and returns at once. The state appends a
	// Reserve with it before it gives out a value above the durable limit.
	Append func(e *log.Entry)
	// Executor runs each step of the owner. It must run every function it gets.
	Executor func(f func())
	// OnHead gets the durable snapshot that the head of the log of generation holds, on
	// a step of the owner. It must return at once.
	OnHead func(generation uint64, s *Snapshot)
	// Generation is the generation of the head that Open loads, from 1.
	Generation uint64
}

// validate returns an error that wraps validate.ErrValidation when a field of c is not
// set.
func (c Config) validate() error {
	v := validate.New("state.Config")
	v.Ternary("versions", !c.Versions.Contains(c.Version), "must hold version")
	v.NotNil("append", c.Append)
	v.NotNil("executor", c.Executor)
	v.NotNil("on_head", c.OnHead)
	v.Ternary("generation", c.Generation == 0, "must be at least 1")
	return v.Error()
}

// request is a call that the owner runs on the core, in queue order. A request that
// comes after the state failed gets the error instead.
type request interface {
	run(s *State)
	fail(err error)
}

// State is the state of a database. Its owner, one goroutine, runs each request in the
// order of its call, and readers load the visible Channel of a key with no lock. A
// State is safe for concurrent use.
type State struct {
	cfg  Config
	core *core
	// keys maps each key to its channel for readers. The owner replaces the map when it
	// adds or drops a key.
	keys atomic.Pointer[map[uint32]*chanState]
	wake chan struct{}
	exit chan struct{}
	mu   struct {
		sync.Mutex
		queue  []request
		err    error
		closed bool
	}
	step    func()
	stepped chan bool
	// replayed is true once replay mode ends.
	replayed bool
	// The fields below belong to the owner.
	nextID, nextAlign uint64
	idBlocks          int
	alignBlocks       int
	// refusedBlocks holds idBlocks and alignBlocks from before the Reserve in flight,
	// which a refusal of it restores.
	refusedBlocks [2]int
	entry         log.Entry
	// parked holds the requests that wait for a durable Reserve.
	parked []request
}

// Open returns the state of the durable state head, in replay mode. It returns an error
// when a value of head breaks a rule, or log.ErrUnknown for a segment format version
// that this binary does not read.
func Open(cfg Config, head *Snapshot) (*State, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	s := &State{
		cfg:     cfg,
		core:    newCore(cfg.Versions),
		wake:    make(chan struct{}, 1),
		exit:    make(chan struct{}),
		stepped: make(chan bool, 1),
	}
	if err := s.core.load(head, cfg.Generation); err != nil {
		return nil, err
	}
	s.core.generation = cfg.Generation
	s.core.count.end()
	s.step = s.run
	s.nextID, s.nextAlign = max(head.Limits.Segments, 1), max(head.Limits.Alignments, 1)
	return s, nil
}

// Replay applies the records of the durable entry of sequence, in replay mode, before
// Run. It returns an error for a record that breaks a rule, or log.ErrUnknown for one
// this binary does not apply.
func (s *State) Replay(sequence uint64, records []log.Record) error {
	defer s.core.count.end()
	return s.core.apply(sequence, records)
}

// EndReplay ends replay mode and publishes the visible state of every key, so readers
// see the replayed state before Run. Replay must not follow it.
func (s *State) EndReplay() {
	s.core.endReplay()
	s.storeKeys()
	s.core.count.end()
	s.replayed = true
}

// Run ends replay mode unless EndReplay did, publishes the visible state of every key,
// and runs one step of the owner on the executor each time requests are queued, until
// Close, until ctx is done, or until a step fails the state. It returns the error of
// ctx, the error that failed the state, or nil at Close. A request that Run leaves
// queued gets the error.
func (s *State) Run(ctx context.Context) error {
	defer close(s.exit)
	if !s.replayed {
		s.EndReplay()
	}
	s.nextID = max(s.nextID, s.core.limits.Segments, 1)
	s.nextAlign = max(s.nextAlign, s.core.limits.Alignments)
	for {
		select {
		case <-ctx.Done():
			s.fail(ctx.Err())
			s.drain()
			return ctx.Err()
		case <-s.wake:
		}
		s.cfg.Executor(s.step)
		closed := <-s.stepped
		if err := s.Err(); err != nil {
			s.drain()
			return err
		}
		if closed {
			return nil
		}
	}
}

// drain fails the queued and the parked requests with the error of the failed state,
// which refuses new ones.
func (s *State) drain() {
	err := s.Err()
	s.mu.Lock()
	queue := s.mu.queue
	s.mu.queue = nil
	s.mu.Unlock()
	for _, r := range queue {
		r.fail(err)
	}
	s.failParked(err)
}

// run is one step of the owner: it runs every queued request.
func (s *State) run() {
	s.mu.Lock()
	queue, closed := s.mu.queue, s.mu.closed
	s.mu.queue = nil
	s.mu.Unlock()
	for _, r := range queue {
		if err := s.Err(); err != nil {
			r.fail(err)
			continue
		}
		r.run(s)
	}
	if err := s.Err(); err != nil {
		s.failParked(err)
	} else if closed {
		s.failParked(errClosed)
	}
	s.core.count.end()
	s.stepped <- closed
}

// enqueue queues r, or returns the error that refuses it.
func (s *State) enqueue(r request) error {
	s.mu.Lock()
	switch {
	case s.mu.err != nil:
		err := s.mu.err
		s.mu.Unlock()
		return err
	case s.mu.closed:
		s.mu.Unlock()
		return errClosed
	}
	s.mu.queue = append(s.mu.queue, r)
	s.mu.Unlock()
	select {
	case s.wake <- struct{}{}:
	default:
	}
	return nil
}

// fail fails the state with err: every later call returns it.
func (s *State) fail(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.mu.err == nil {
		s.mu.err = errors.Wrap(err, "state failed")
	}
}

// Err returns the error that failed the state, or nil.
func (s *State) Err() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mu.err
}

// Close stops the owner after the queued requests, and waits for Run to return.
func (s *State) Close() error {
	s.mu.Lock()
	s.mu.closed = true
	s.mu.Unlock()
	select {
	case s.wake <- struct{}{}:
	default:
	}
	<-s.exit
	return s.Err()
}

// storeKeysIfChanged publishes the map of keys when a key joined or left. Only the
// owner calls it.
func (s *State) storeKeysIfChanged() {
	if s.core.keysChanged {
		s.storeKeys()
	}
}

// storeKeys publishes the map of keys for readers. Only the owner calls it.
func (s *State) storeKeys() {
	s.core.count.begin()
	s.core.keysChanged = false
	m := maps.Clone(s.core.channels)
	s.keys.Store(&m)
}

// Channel returns the visible state of key, or nil when the state holds no segment of
// it. It takes no lock, and the Channel never changes.
func (s *State) Channel(key uint32) *Channel {
	m := s.keys.Load()
	if m == nil {
		return nil
	}
	ch, ok := (*m)[key]
	if !ok {
		return nil
	}
	return ch.pub.Load()
}

// Load sets each of chs to the visible state of the key of keys at its index, from one
// step of the owner, or to nil for a key that the state holds no segment of. It loads
// them all again while the owner publishes between its loads. It returns the publish
// count of the loads, which Unchanged takes. chs must be as long as keys.
func (s *State) Load(keys []uint32, chs []*Channel) uint64 {
	for {
		count := s.core.count.n.Load()
		if count%2 == 1 {
			runtime.Gosched()
			continue
		}
		for i, key := range keys {
			chs[i] = s.Channel(key)
		}
		if s.core.count.n.Load() == count {
			return count
		}
	}
}

// Unchanged reports whether the owner published nothing since the loads of Load that
// returned count.
func (s *State) Unchanged(count uint64) bool { return s.core.count.n.Load() == count }

type durableRequest struct {
	sequence uint64
	records  []log.Record
}

func (*durableRequest) fail(error) {}

func (r *durableRequest) run(s *State) {
	if err := s.core.apply(r.sequence, r.records); err != nil {
		s.fail(err)
		return
	}
	if invariants.Enabled {
		s.checkCommits(r.records)
	}
	if len(s.parked) > 0 {
		s.unpark()
	}
}

// Tag names the part that holds a segment record, as the part holds it.
type Tag struct {
	Part    uint64
	Segment Segment
}

type partedRequest struct {
	generation uint64
	tags       []Tag
	ranges     []Range
	swap       func()
}

// fail drops the request: the state failed, and later calls return its error.
func (*partedRequest) fail(error) {}

func (r *partedRequest) run(s *State) {
	// The swap changes the parts that a load pins, so it opens the count of publishes.
	s.core.count.begin()
	r.swap()
	changed := set.New[uint32]()
	for _, t := range r.tags {
		seg, ok := s.core.segments[t.Segment.ID]
		if !ok || !sameRecord(seg.vis, t.Segment) {
			continue
		}
		seg.vis.Part = t.Part
		changed.Add(seg.key)
	}
	folded := func(k Kill) bool { return k.Generation < r.generation }
	s.core.droppedKills = slices.DeleteFunc(s.core.droppedKills, folded)
	ranges := make(map[uint32]Range, len(r.ranges))
	for _, rg := range r.ranges {
		ranges[rg.Key] = rg
	}
	// A key that a Drop took out after the snapshot of the head is a new key, and the
	// range of the head belongs to the dropped one.
	for _, k := range s.core.droppedKills {
		delete(ranges, k.Key)
	}
	for key, ch := range s.core.channels {
		rg := ranges[key]
		// A range names parts of the new head, even when its bounds stay.
		repart := ch.oldest != 0 || rg.Oldest != 0
		if repart {
			ch.oldest, ch.newest, ch.generation = rg.Oldest, rg.Newest, r.generation
		}
		if slices.ContainsFunc(ch.kills, folded) {
			// A published Channel holds the old slice, so the kills left go to a new
			// one.
			ch.kills = slices.DeleteFunc(slices.Clone(ch.kills), folded)
			repart = true
		}
		if repart {
			ch.repart()
			changed.Add(key)
		}
	}
	for key := range changed {
		s.core.channels[key].publish()
	}
}

// sameRecord reports whether a and b are the same sealed segment record.
func sameRecord(a, b Segment) bool {
	return a.Sealed && b.Sealed && a.Length == b.Length && a.TailCRC == b.TailCRC &&
		slices.Equal(a.Pointers, b.Pointers)
}

// Parted queues what the head of generation holds once it commits: the tags of the
// records that its switch wrote to parts, and the part range of each key that parts
// hold. The owner first calls swap, which makes the parts of the head readable. swap
// runs on a step of the owner, so it must return at once and not wait on the state.
// The owner then tags each sealed segment whose record is still the one that its part
// holds, gives each key its range in the head or none, and drops the kills from before
// generation, which the head holds as dead marks. Parted does not wait for the owner.
func (s *State) Parted(
	generation uint64,
	tags []Tag,
	ranges []Range,
	swap func(),
) error {
	return s.enqueue(&partedRequest{
		generation: generation,
		tags:       tags,
		ranges:     ranges,
		swap:       swap,
	})
}

// Drops returns the count of the Drop records that the state applied.
func (s *State) Drops() uint64 { return s.core.drops.Load() }

// Durable queues the records of the durable entry of sequence, which the log owner
// hands over in log order after the entry's frame is synced. The owner applies them to
// the durable view with the rules of Replay. A refusal is a defect of the database: it
// fails the state.
func (s *State) Durable(sequence uint64, records []log.Record) error {
	return s.enqueue(&durableRequest{sequence: sequence, records: records})
}

// Dead is a segment that left the state, whose file the database unlinks.
type Dead struct {
	// Segment is the identity of the segment.
	Segment uint64
	// Key is the key whose samples the segment held.
	Key uint32
	// Sequence is the entry whose record took the segment out of the state, or 0 when
	// no durable record named the segment.
	Sequence uint64
}

type deadRequest struct {
	dead []Dead
	done chan error
}

func (r *deadRequest) fail(err error) { r.done <- err }

func (r *deadRequest) run(s *State) {
	r.dead, s.core.dead = s.core.dead, nil
	s.core.hasDead.Store(false)
	r.done <- nil
}

// HasDead reports whether TakeDead has segments to return. It is safe to call from any
// goroutine.
func (s *State) HasDead() bool { return s.core.hasDead.Load() }

// TakeDead returns the segments that left the state since the last call: by a Drop, or
// by a Tombstone, a Seal, or a Restart that left them sealed with no durable pointer or
// with no durable byte. The database unlinks their files once the log covers the record
// that took them out and no reader pins them.
func (s *State) TakeDead() ([]Dead, error) {
	r := &deadRequest{done: make(chan error, 1)}
	if err := s.enqueue(r); err != nil {
		return nil, err
	}
	if err := wait(s, r.done); err != nil {
		return nil, err
	}
	return r.dead, nil
}

// checkCommits runs checkViews on each key that a Commit of records names.
func (s *State) checkCommits(records []log.Record) {
	for _, r := range records {
		if c, ok := r.(*log.Commit); ok {
			for _, slot := range c.Slots {
				s.core.channels[s.core.segments[slot.Segment].key].checkViews()
			}
		}
	}
}

type snapshotRequest struct {
	snap *Snapshot
	done chan error
}

func (r *snapshotRequest) fail(err error) { r.done <- err }

func (r *snapshotRequest) run(s *State) {
	r.snap = s.core.snapshot()
	r.done <- nil
}

// waited is a request that its caller waits for with a context. Once the caller gives
// up before the owner takes it, the owner neither runs nor fails it, so it leaves no
// claim with no owner and sends no error that a later call would receive.
type waited struct {
	request
	// taken is waiting while the request waits, then taken once the owner takes it,
	// or abandoned once its caller gives up.
	taken atomic.Int32
}

const (
	waiting int32 = iota
	taken
	abandoned
)

func (r *waited) run(s *State) {
	if r.taken.CompareAndSwap(waiting, taken) {
		r.request.run(s)
	}
}

func (r *waited) fail(err error) {
	if r.taken.CompareAndSwap(waiting, taken) {
		r.request.fail(err)
	}
}

// waitFor queues r for the owner and returns the error that done receives. When ctx
// ends
// before the owner takes r, it returns the error of ctx. Once the owner takes r, it
// waits for r to end.
func waitFor(ctx context.Context, s *State, r request, done chan error) error {
	w := &waited{request: r}
	if err := s.enqueue(w); err != nil {
		return err
	}
	select {
	case err := <-done:
		return err
	case <-s.exit:
		return wait(s, done)
	case <-ctx.Done():
		if w.taken.CompareAndSwap(waiting, abandoned) {
			return ctx.Err()
		}
		return wait(s, done)
	}
}

// wait returns the error that done receives, or errClosed when the owner stops first.
func wait(s *State, done chan error) error {
	select {
	case err := <-done:
		return err
	case <-s.exit:
		select {
		case err := <-done:
			return err
		default:
			return errors.Join(errClosed, s.Err())
		}
	}
}

// Snapshot returns the durable state after every entry that Durable queued before the
// call.
func (s *State) Snapshot() (*Snapshot, error) {
	r := &snapshotRequest{done: make(chan error, 1)}
	if err := s.enqueue(r); err != nil {
		return nil, err
	}
	if err := wait(s, r.done); err != nil {
		return nil, err
	}
	return r.snap, nil
}
