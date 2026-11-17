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
	"github.com/synnaxlabs/synnax/pkg/service/channel"
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
		w.pathChanges.Add(w.tx, struct{}{})
		return nil
	}
	if module && containsTextEdit(acts) {
		w.editedModules.Add(w.tx, key)
	}
	return nil
}

// followCommit schedules the resyncs that c asked for, then drops the edit time and
// pending resync of each Arc that c deleted. It runs only for a commit, so a denied or
// failed write changes neither.
func (s *Service) followCommit(_ context.Context, c gorp.Commit) {
	for _, key := range s.resyncs.Of(c) {
		s.taskSync.Trigger(key)
	}
	for _, key := range s.editedModules.Of(c) {
		s.importerSync.Trigger(key)
	}
	if len(s.pathChanges.Of(c)) > 0 {
		s.pathSync.Trigger(struct{}{})
	}
	for _, key := range s.deletes.Of(c) {
		s.sweeper.forget(key)
		s.taskSync.Forget(key)
		s.importerSync.Forget(key)
	}
}

// resync rewrites an automation's task config or a module's compile status from its
// content. It runs on the task-sync debouncer and compiles before its transaction
// opens. An automation with no task has nothing to rewrite.
func (s *Service) resync(ctx context.Context, key Key) {
	s.withArc(ctx, key, "failed to resync arc", func(a Arc) error {
		if a.Type() == TypeAutomation {
			tasks, err := retrieveTaskKeys(
				ctx, s.cfg.Task, s.cfg.DB, s.cfg.MatchTasks(key),
			)
			if err != nil || len(tasks) == 0 {
				return err
			}
		}
		b, err := s.Build(ctx, key)
		if errors.Is(err, channel.ErrLookupsChanged) {
			s.taskSync.Trigger(key)
			return nil
		}
		if err != nil {
			return err
		}
		return s.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
			same, err := b.lookups.Unchanged(ctx, tx)
			if err != nil {
				return err
			}
			// A write committed after the compile. A channel change schedules no resync
			// of its own, so skip the write and resync again.
			if !same {
				s.taskSync.Trigger(key)
				return nil
			}
			if err = s.table.NewRetrieve().
				Where(gorp.MatchKeys[Key, Arc](key)).
				Entry(&a).
				Exec(ctx, tx); err != nil {
				return err
			}
			return s.NewWriter(tx).sync(ctx, a, b)
		})
	})
}

// resyncImporters schedules a resync of every importer of the module with key. It runs
// on the importer-sync debouncer and reads committed state with no lock.
func (s *Service) resyncImporters(ctx context.Context, key Key) {
	s.withArc(ctx, key, "failed to resync arc importers", func(a Arc) error {
		path, err := s.pathOf(ctx, s.cfg.DB, a)
		if err != nil {
			return err
		}
		keys, err := s.importersOf(ctx, s.cfg.DB, path)
		for _, k := range keys {
			s.taskSync.Trigger(k)
		}
		return err
	})
}

// withArc runs fn on the Arc with key, read with no lock. A debouncer callback has no
// caller to return errors to, so withArc logs them under msg.
func (s *Service) withArc(
	ctx context.Context,
	key Key,
	msg string,
	fn func(Arc) error,
) {
	var a Arc
	err := s.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, Arc](key)).
		Entry(&a).
		Exec(ctx, s.cfg.DB)
	if err == nil {
		err = fn(a)
	}
	if failed(err) {
		s.cfg.L.Error(msg, zap.Error(err), zap.Stringer("key", key))
	}
}

// failed reports whether a debouncer callback should log err. A cancelled context (the
// service closed) and a deleted Arc are not failures.
func failed(err error) bool {
	return err != nil && !errors.Is(err, context.Canceled) &&
		!errors.Is(err, query.ErrNotFound)
}
