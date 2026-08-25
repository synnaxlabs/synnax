// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"context"
	"sync"
	"time"

	"github.com/synnaxlabs/x/crdt"
	"github.com/synnaxlabs/x/debounce"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	"go.uber.org/zap"
)

// Dispatch applies a sequence of actions atomically to the Arc with the given key.
// Dispatches for the same Arc run one at a time, each in its own transaction committed
// before the actions are notified, so two concurrent dispatches can never overwrite
// each other's edits. dispatchKey is a client-generated identifier carried verbatim
// onto the broadcast so the originating client can recognize its own echo.
//
// When the Arc's text has gone quiet, Dispatch also reclaims the space held by
// tombstoned characters: it forgets the characters that were already dead before this
// dispatch and broadcasts that sweep as a separate frame with an empty dispatchKey, so
// every editor (including the originator, which skips its own echo) applies it.
//
// Dispatch also schedules a debounced rewrite of the Arc's task config, so a burst of
// edits produces one rewrite instead of one per edit.
func (s *Service) Dispatch(
	ctx context.Context,
	key Key,
	dispatchKey string,
	acts []Action,
) error {
	dispatcher := s.state.Dispatcher()
	return s.exec.Serialize(key, func() error {
		var sweep []Action
		if err := s.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return s.table.NewUpdate().Where(gorp.MatchKeys[Key, Arc](key)).
				ChangeErr(func(_ gorp.Context, a Arc) (Arc, error) {
					sweep = nil
					var dead []crdt.ID
					if s.sweeper.quiet(key) {
						dead = s.sweeper.forgettable(a.Text.Doc)
					}
					a, err := Reduce(a, acts...)
					if err != nil {
						return a, err
					}
					if len(dead) > 0 {
						sweep = []Action{
							NewForgetCharsAction(ForgetCharsPayload{IDs: dead}),
						}
						if a, err = Reduce(a, sweep...); err != nil {
							return a, err
						}
					}
					if containsTextEdit(acts) {
						s.sweeper.recordEdit(key)
					}
					return a, nil
				}).Exec(ctx, tx)
		}); err != nil {
			return err
		}
		dispatcher.Notify(ctx, key, dispatchKey, acts)
		if len(sweep) > 0 {
			dispatcher.Notify(ctx, key, "", sweep)
		}
		return s.taskSync.trigger(key)
	})
}

// resyncTask rewrites the Arc's task config from its current content. It runs on the
// task-sync debouncer, so it has no caller to return errors to and logs them instead. A
// cancelled context (a newer edit superseded this sync) and a deleted Arc are not
// errors.
func (s *Service) resyncTask(ctx context.Context, key Key) {
	err := s.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		var a Arc
		if err := s.table.NewRetrieve().
			Where(gorp.MatchKeys[Key, Arc](key)).
			Entry(&a).
			Exec(ctx, tx); err != nil {
			return err
		}
		return s.NewWriter(tx).syncTask(ctx, a)
	})
	if err == nil || errors.Is(err, context.Canceled) ||
		errors.Is(err, query.ErrNotFound) {
		return
	}
	s.cfg.L.Error("failed to sync arc task", zap.Error(err), zap.Stringer("key", key))
}

// taskSync debounces per-Arc task config rewrites.
type taskSync struct {
	delay time.Duration
	sync  func(context.Context, Key)
	// mu guards debouncers and closed. wg.Add only runs under mu while closed is false,
	// so close observes every in-flight run.
	mu         sync.Mutex
	closed     bool
	debouncers map[Key]*debounce.Debouncer
	wg         sync.WaitGroup
}

func newTaskSync(delay time.Duration, sync func(context.Context, Key)) *taskSync {
	return &taskSync{
		delay:      delay,
		sync:       sync,
		debouncers: make(map[Key]*debounce.Debouncer),
	}
}

// trigger schedules a sync for the Arc, superseding any pending or in-flight one. A
// no-op once the taskSync is closed.
func (t *taskSync) trigger(key Key) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return nil
	}
	d, ok := t.debouncers[key]
	if !ok {
		var err error
		if d, err = debounce.New(debounce.Config{
			Delay:    t.delay,
			MaxDelay: t.delay * 4,
			Callback: func(ctx context.Context) { t.run(ctx, key) },
		}); err != nil {
			return err
		}
		t.debouncers[key] = d
	}
	d.Trigger()
	return nil
}

func (t *taskSync) run(ctx context.Context, key Key) {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return
	}
	t.wg.Add(1)
	t.mu.Unlock()
	defer t.wg.Done()
	t.sync(ctx, key)
}

// forget drops the Arc's debouncer, discarding any pending sync. It is called when the
// Arc is deleted.
func (t *taskSync) forget(key Key) {
	t.mu.Lock()
	d, ok := t.debouncers[key]
	delete(t.debouncers, key)
	t.mu.Unlock()
	if ok {
		d.Stop()
	}
}

// close discards pending syncs, cancels in-flight ones, and waits for them to return.
func (t *taskSync) close() {
	t.mu.Lock()
	t.closed = true
	ds := make([]*debounce.Debouncer, 0, len(t.debouncers))
	for _, d := range t.debouncers {
		ds = append(ds, d)
	}
	clear(t.debouncers)
	t.mu.Unlock()
	for _, d := range ds {
		d.Stop()
	}
	t.wg.Wait()
}
