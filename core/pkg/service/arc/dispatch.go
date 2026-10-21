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

	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/crdt"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
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
		var (
			sweep  []Action
			module bool
		)
		if err := s.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return s.table.NewUpdate().Where(gorp.MatchKeys[Key, Arc](key)).
				ChangeErr(func(_ gorp.Context, a Arc) (Arc, error) {
					sweep = nil
					module = a.Type == TypeModule
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
		if module {
			return s.resyncAll(ctx, nil)
		}
		s.taskSync.Trigger(key)
		return nil
	})
}

// resync rewrites the Arc's task config from its current content, or compiles a
// module, which has no task. It runs on the task-sync debouncer, so it has no caller
// to return errors to and logs them instead. A cancelled context (the service closed)
// and a deleted Arc are not errors.
func (s *Service) resync(ctx context.Context, key Key) {
	err := s.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		var a Arc
		if err := s.table.NewRetrieve().
			Where(gorp.MatchKeys[Key, Arc](key)).
			Entry(&a).
			Exec(ctx, tx); err != nil {
			return err
		}
		if a.Type == TypeModule {
			return s.setCompileStatus(ctx, tx, a)
		}
		return s.NewWriter(tx).syncTask(ctx, a)
	})
	if err == nil || errors.Is(err, context.Canceled) ||
		errors.Is(err, query.ErrNotFound) {
		return
	}
	s.cfg.L.Error("failed to sync arc task", zap.Error(err), zap.Stringer("key", key))
}

// resyncAll schedules a resync of every Arc, so importers follow a module write.
func (s *Service) resyncAll(ctx context.Context, tx gorp.Tx) error {
	var arcs []Arc
	if err := s.table.NewRetrieve().
		Entries(&arcs).
		Exec(ctx, gorp.OverrideTx(s.cfg.DB, tx)); err != nil {
		return err
	}
	for _, a := range arcs {
		s.taskSync.Trigger(a.Key)
	}
	return nil
}

// setCompileStatus compiles the module and records the result as its status: an
// error carrying the diagnostics, or success.
func (s *Service) setCompileStatus(ctx context.Context, tx gorp.Tx, a Arc) error {
	stat := Status{
		Key:     a.OntologyID().String(),
		Name:    a.Name,
		Variant: status.VariantSuccess,
		Message: "Compiled",
		Time:    telem.Now(),
	}
	if _, err := s.compile(ctx, a); err != nil {
		stat.Variant = status.VariantError
		stat.Message = err.Error()
	}
	return s.cfg.Status.NewWriter(tx).Set(ctx, &stat)
}
