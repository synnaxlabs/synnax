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

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/crdt"
	"github.com/synnaxlabs/x/errors"
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
	return s.locks.Do(key, func() error {
		var sweep []Action
		if err := s.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return s.table.NewUpdate().Where(gorp.MatchKeys[Key, Arc](key)).
				ChangeErr(func(_ gorp.Context, a Arc) (Arc, error) {
					sweep = nil
					var dead []crdt.ID
					src, isText := a.Base().Source.Variant.(TextSource)
					if isText && s.sweeper.quiet(key) {
						dead = s.sweeper.forgettable(src.Text.Doc)
					}
					name := a.Base().Name
					a, err := Reduce(a, acts...)
					if err != nil {
						return a, err
					}
					if renamed := a.Base().Name; renamed != name {
						if err = s.NewWriter(tx).validateName(
							ctx, OntologyID(key), renamed, a.Type() == TypeModule,
						); err != nil {
							return Arc{}, err
						}
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
		dispatcher.WithDispatchKey(dispatchKey).Notify(ctx, key, acts...)
		if len(sweep) > 0 {
			dispatcher.Notify(ctx, key, sweep...)
		}
		s.taskSync.Trigger(key)
		return nil
	})
}

// resyncTask rewrites the Arc's task configs from its current content, holding each
// task's lock through the commit so that a sync cannot overwrite a task.Dispatch. It
// runs on the task-sync debouncer and after an Arc write commits, so it has no caller
// to return errors to and logs them instead. A cancelled context (the service closed)
// and a deleted Arc or task are not errors.
func (s *Service) resyncTask(ctx context.Context, key Key) {
	taskKeys, err := s.NewWriter(nil).childTaskKeys(ctx, key)
	for _, taskKey := range taskKeys {
		if err != nil {
			break
		}
		err = errors.Skip(s.cfg.Task.Transact(
			ctx,
			[]task.Key{taskKey},
			func(tx gorp.Tx) error {
				var a Arc
				if err := s.table.NewRetrieve().
					Where(gorp.MatchKeys[Key, Arc](key)).
					Entry(&a).
					Exec(ctx, tx); err != nil {
					return err
				}
				return s.NewWriter(tx).syncTask(ctx, a.Base(), taskKey)
			},
		), query.ErrNotFound)
	}
	if err == nil || errors.Is(err, context.Canceled) ||
		errors.Is(err, query.ErrNotFound) {
		return
	}
	s.cfg.L.Error("failed to sync arc task", zap.Error(err), zap.Stringer("key", key))
}

// UpdateTask applies Writer.UpdateTask in its own transaction. It holds the Arc's lock
// and the locks of the Arc's tasks through the commit, so it cannot overwrite a
// task.Dispatch and two binds of one Arc cannot each create a task.
func (s *Service) UpdateTask(
	ctx context.Context,
	key Key,
	update TaskUpdate,
) (*task.Task, error) {
	var tsk *task.Task
	if err := s.locks.Do(key, func() error {
		for {
			locked, err := s.NewWriter(nil).childTaskKeys(ctx, key)
			if err != nil {
				return err
			}
			var unlocked bool
			if err = s.cfg.Task.Transact(ctx, locked, func(tx gorp.Tx) error {
				w := s.NewWriter(tx)
				children, err := w.childTaskKeys(ctx, key)
				if err != nil {
					return err
				}
				// A task create links its task without the Arc's lock.
				if unlocked = !lo.Every(locked, children); unlocked {
					return nil
				}
				tsk, err = w.updateTask(ctx, key, children, update)
				return err
			}); err != nil || !unlocked {
				return err
			}
		}
	}); err != nil {
		return nil, err
	}
	return tsk, nil
}
