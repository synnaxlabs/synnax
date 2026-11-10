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
// dispatch key of w.
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
func (w Writer) Apply(ctx context.Context, key Key, acts ...Action) error {
	var (
		sweep       []Action
		module      bool
		pathChanged bool
	)
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
			module = a.Type() == TypeModule
			next := a.Base()
			pathChanged = next.Name != prev.Name ||
				lo.FromPtr(next.Directory) != lo.FromPtr(prev.Directory)
			if pathChanged && next.Directory != nil {
				if err = w.validateEntry(
					ctx, *next.Directory, ResourceID(key), next.Name, module,
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
		return err
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
	if module && pathChanged {
		return w.resyncImporting(ctx)
	}
	if module && containsTextEdit(acts) {
		w.editedModules.Add(w.tx, key)
	}
	return nil
}

// followCommit schedules a task resync for each Arc that c dispatched to, then drops
// the edit time and pending resync of each Arc that c deleted. It runs only for a
// commit, so a denied or failed write changes neither.
func (s *Service) followCommit(_ context.Context, c gorp.Commit) {
	for _, key := range s.resyncs.Of(c) {
		s.taskSync.Trigger(key)
	}
	for _, key := range s.editedModules.Of(c) {
		s.importerSync.Trigger(key)
	}
	for _, key := range s.deletes.Of(c) {
		s.sweeper.forget(key)
		s.taskSync.Forget(key)
		s.importerSync.Forget(key)
	}
}

// resync rewrites an automation's task config or a module's compile status from its
// content. It runs on the task-sync debouncer.
func (s *Service) resync(ctx context.Context, key Key) {
	s.withArc(ctx, key, "failed to resync arc", Writer.sync)
}

// resyncImporters schedules a resync of every importer of the module with key. It
// runs on the importer-sync debouncer.
func (s *Service) resyncImporters(ctx context.Context, key Key) {
	s.withArc(ctx, key, "failed to resync arc importers", func(
		w Writer,
		ctx context.Context,
		a Arc,
	) error {
		path, err := w.pathOf(ctx, a)
		if err != nil {
			return err
		}
		keys, err := w.importersOf(ctx, path)
		if err != nil {
			return err
		}
		w.resyncs.Add(w.tx, keys...)
		return nil
	})
}

// withArc runs fn on the Arc with key in its own transaction. A debouncer callback has
// no caller to return errors to, so withArc logs them under msg. A cancelled context
// (the service closed) and a deleted Arc are not errors.
func (s *Service) withArc(
	ctx context.Context,
	key Key,
	msg string,
	fn func(Writer, context.Context, Arc) error,
) {
	err := s.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		var a Arc
		if err := s.table.NewRetrieve().
			Where(gorp.MatchKeys[Key, Arc](key)).
			Entry(&a).
			Exec(ctx, tx); err != nil {
			return err
		}
		return fn(s.NewWriter(tx), ctx, a)
	})
	if err == nil || errors.Is(err, context.Canceled) ||
		errors.Is(err, query.ErrNotFound) {
		return
	}
	s.cfg.L.Error(msg, zap.Error(err), zap.Stringer("key", key))
}
