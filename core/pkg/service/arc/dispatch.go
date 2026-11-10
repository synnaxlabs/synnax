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
	"github.com/synnaxlabs/x/crdt"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"go.uber.org/zap"
)

// Apply applies a sequence of actions to the Arc with the given key and attaches them
// to the transaction of w, so its commit carries them to the change channels with the
// dispatch key of w. It returns acts, without the sweep below.
//
// When the Arc's text has gone quiet, Apply also reclaims the space held by tombstoned
// characters: it forgets the characters that were already dead before this apply and
// attaches that sweep as a second batch with an empty dispatch key, so every editor
// (including the originator, which skips its own echo) applies it.
//
// When the transaction of w commits, the Arc's task config gets a debounced rewrite, so
// a burst of edits produces one rewrite instead of one per edit. A text edit moves the
// Arc's last edit time only when the transaction commits.
//
// Apply returns a validation error when a rename or a move leaves a name that the
// directory of the Arc cannot hold.
func (w Writer) Apply(ctx context.Context, key Key, acts ...Action) ([]Action, error) {
	var sweep []Action
	if err := w.table.NewUpdate().Where(gorp.MatchKeys[Key, Arc](key)).
		ChangeErr(func(_ gorp.Context, a Arc) (Arc, error) {
			sweep = nil
			var dead []crdt.ID
			src, isText := a.Base().Source.Variant.(TextSource)
			if isText && w.sweeper.quiet(key) {
				dead = w.sweeper.forgettable(src.Text.Doc)
			}
			prev := a.Base()
			a, err := Reduce(a, acts...)
			if err != nil {
				return a, err
			}
			if next := a.Base(); next.Directory != nil && (next.Name != prev.Name ||
				*next.Directory != lo.FromPtr(prev.Directory)) {
				if err = w.validateEntry(
					ctx,
					*next.Directory,
					ResourceID(key),
					next.Name,
					a.Type() == TypeModule,
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
			return a, nil
		}).Exec(ctx, w.tx); err != nil {
		return nil, err
	}
	// The edit time moves before the lock releases, so a later transaction cannot sweep
	// the characters that this one deletes.
	if containsTextEdit(acts) {
		gorp.BeforeCommit(w.tx, func(context.Context) error {
			w.sweeper.recordEdit(key)
			return nil
		})
	}
	w.dispatcher.Attach(w.tx, key, acts...)
	if len(sweep) > 0 {
		w.dispatcher.WithDispatchKey("").Attach(w.tx, key, sweep...)
	}
	w.resyncs.Add(w.tx, key)
	return acts, nil
}

// followCommit schedules a task resync for each Arc that c dispatched to, then drops
// the edit time and pending resync of each Arc that c deleted. It runs only for a
// commit, so a denied or failed write changes neither.
func (s *Service) followCommit(_ context.Context, c gorp.Commit) {
	for _, key := range s.resyncs.Of(c) {
		s.taskSync.Trigger(key)
	}
	for _, key := range s.deletes.Of(c) {
		s.sweeper.forget(key)
		s.taskSync.Forget(key)
	}
}

// resyncTask rewrites the Arc's task config from its current content. It runs on the
// task-sync debouncer, so it has no caller to return errors to and logs them instead. A
// cancelled context (the service closed) and a deleted Arc are not errors.
func (s *Service) resyncTask(ctx context.Context, key Key) {
	err := s.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		var a Arc
		if err := s.table.NewRetrieve().
			Where(gorp.MatchKeys[Key, Arc](key)).
			Entry(&a).
			Exec(ctx, tx); err != nil {
			return err
		}
		return s.NewWriter(tx).syncTask(ctx, a.Base())
	})
	if err == nil || errors.Is(err, context.Canceled) ||
		errors.Is(err, query.ErrNotFound) {
		return
	}
	s.cfg.L.Error("failed to sync arc task", zap.Error(err), zap.Stringer("key", key))
}
