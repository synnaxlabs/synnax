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
	"sync"
	"sync/atomic"

	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// ErrClosed is the error of a call after the owner stopped.
var ErrClosed = errors.New("state is closed")

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
}

// Validate returns an error that wraps validate.ErrValidation when a field of c is not
// set.
func (c Config) Validate() error {
	v := validate.New("state.Config")
	v.Ternary("versions", !c.Versions.Contains(c.Version), "must hold version")
	v.NotNil("append", c.Append)
	v.NotNil("executor", c.Executor)
	return v.Error()
}

// request is a call that the owner runs on the core, in queue order.
type request interface{ run(s *State) }

// State is the state of a database. Its owner, one goroutine, runs each request in the
// order of its call, and readers load the visible Channel of a key with no lock. A
// State is safe for concurrent use.
type State struct {
	cfg  Config
	core *core
	// keys maps each key to its channel for readers. The owner replaces the map when
	// it adds or drops a key.
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
	// The fields below belong to the owner.
	nextID, nextAlign uint64
	idBlocks          int
	alignBlocks       int
	entry             log.Entry
}

// Open returns the state of the durable state head, in replay mode. It returns an
// error that wraps ErrRefused when a value of head breaks a rule, or log.ErrUnknown
// for a segment format version that this binary does not read.
func Open(cfg Config, head *Snapshot) (*State, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	s := &State{
		cfg:     cfg,
		core:    newCore(cfg.Versions),
		wake:    make(chan struct{}, 1),
		exit:    make(chan struct{}),
		stepped: make(chan bool, 1),
	}
	if err := s.core.load(head); err != nil {
		return nil, err
	}
	s.step = s.run
	s.nextID, s.nextAlign = head.Limits.Segments, max(head.Limits.Alignments, 1)
	return s, nil
}

// Replay applies the records of the durable entry of sequence, in replay mode, before
// Run. It returns an error that wraps ErrRefused for a record that breaks a rule, or
// log.ErrUnknown for one this binary does not apply.
func (s *State) Replay(sequence uint64, records []log.Record) error {
	return s.core.apply(sequence, records)
}

// Run ends replay mode, publishes the visible state of every key, and runs one step of
// the owner on the executor each time requests are queued, until Close or until ctx
// is done. It returns the error of ctx, or nil at Close.
func (s *State) Run(ctx context.Context) error {
	defer close(s.exit)
	s.core.endReplay()
	s.storeKeys()
	s.nextID = max(s.nextID, s.core.limits.Segments)
	s.nextAlign = max(s.nextAlign, s.core.limits.Alignments)
	for {
		select {
		case <-ctx.Done():
			s.fail(ctx.Err())
			return ctx.Err()
		case <-s.wake:
		}
		s.cfg.Executor(s.step)
		if <-s.stepped {
			return nil
		}
	}
}

// run is one step of the owner: it runs every queued request.
func (s *State) run() {
	s.mu.Lock()
	queue, closed := s.mu.queue, s.mu.closed
	s.mu.queue = nil
	s.mu.Unlock()
	for _, r := range queue {
		r.run(s)
	}
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
		return ErrClosed
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

type durableRequest struct {
	sequence uint64
	records  []log.Record
}

func (r *durableRequest) run(s *State) {
	if s.Err() != nil {
		return
	}
	if err := s.core.apply(r.sequence, r.records); err != nil {
		s.fail(err)
	}
}

// Durable queues the records of the durable entry of sequence, which the log owner
// hands over in log order after the entry's frame is synced. The owner applies them to
// the durable view with the rules of Replay. A refusal is a defect of the database: it
// fails the state.
func (s *State) Durable(sequence uint64, records []log.Record) error {
	return s.enqueue(&durableRequest{sequence: sequence, records: records})
}

type snapshotRequest struct {
	snap *Snapshot
	done chan error
}

func (r *snapshotRequest) run(s *State) {
	r.snap = s.core.snapshot()
	r.done <- nil
}

// wait returns the error that done receives, or ErrClosed when the owner stops first.
func wait(s *State, done chan error) error {
	select {
	case err := <-done:
		return err
	case <-s.exit:
		select {
		case err := <-done:
			return err
		default:
			return errors.Join(ErrClosed, s.Err())
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
