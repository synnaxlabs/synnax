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

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/crdt"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"go.uber.org/zap"
)

// Apply applies a sequence of actions to the Arc with the given key and attaches them
// to the transaction of w, so its commit carries them to the change channels with the
// dispatch key of w.
//
// When the Arc's text has gone quiet, Apply also reclaims the space held by tombstoned
// characters: it forgets the characters that were already dead before this apply and
// attaches that sweep as a second batch with an empty dispatch key, so every editor
// (including the originator, which skips its own echo) applies it.
//
// Apply also schedules a debounced rewrite of the Arc's task config, so a burst of
// edits produces one rewrite instead of one per edit. The rewrite opens its own
// transaction, so it reads the Arc after the transaction of w commits or rolls back.
func (w Writer) Apply(ctx context.Context, key Key, acts ...Action) error {
	var sweep []Action
	if err := w.table.NewUpdate().Where(gorp.MatchKeys[Key, Arc](key)).
		ChangeErr(func(_ gorp.Context, a Arc) (Arc, error) {
			sweep = nil
			var dead []crdt.ID
			src, isText := a.Base().Source.Variant.(TextSource)
			if isText && w.sweeper.quiet(key) {
				dead = w.sweeper.forgettable(src.Text.Doc)
			}
			name := a.Base().Name
			a, err := Reduce(a, acts...)
			if err != nil {
				return a, err
			}
			if renamed := a.Base().Name; renamed != name {
				if err = w.validateName(
					ctx, ResourceID(key), renamed, a.Type() == TypeModule,
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
				w.sweeper.recordEdit(key)
			}
			return a, nil
		}).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Attach(w.tx, key, acts...)
	if len(sweep) > 0 {
		w.dispatcher.WithDispatchKey("").Attach(w.tx, key, sweep...)
	}
	w.taskSync.Trigger(key)
	return nil
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
