// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log

import (
	"context"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/workspace"
	"github.com/synnaxlabs/x/gorp"
)

// Writer is used to create, update, and delete logs within Synnax. The writer executes
// all operations within the transaction provided to the Service.NewWriter method. If no
// transaction is provided, the writer will execute operations directly on the database.
type Writer struct {
	tx         gorp.Tx
	otgWriter  ontology.Writer
	otg        *ontology.Ontology
	table      *gorp.Table[Key, Log]
	dispatcher actions.Dispatcher[Key, Action]
}

// Create creates the given log within the workspace provided. If the log does not have
// a key, a new key will be generated. If ws is uuid.Nil, the log is created without a
// workspace ParentOf relationship; this is used by the import path, which does not yet
// wire workspace relationships.
func (w Writer) Create(ctx context.Context, ws workspace.Key, l *Log) error {
	var (
		exists bool
		err    error
	)
	if l.Key == uuid.Nil {
		l.Key = uuid.New()
	} else {
		if exists, err = w.
			table.
			NewRetrieve().
			Where(gorp.MatchKeys[Key, Log](l.Key)).Exists(ctx, w.tx); err != nil {
			return err
		}
	}
	if err = w.table.NewCreate().Entry(l).Exec(ctx, w.tx); err != nil {
		return err
	}
	if exists {
		return nil
	}
	otgID := OntologyID(l.Key)
	if err = w.otgWriter.DefineResource(ctx, otgID); err != nil {
		return err
	}
	if ws == uuid.Nil {
		return nil
	}
	return w.otgWriter.DefineRelationship(
		ctx,
		workspace.OntologyID(ws),
		ontology.RelationshipTypeParentOf,
		otgID,
	)
}

// SetData replaces the body of the log with the given key with the provided value. Key
// and Name are preserved from the existing entry; every other field on data overwrites
// the stored entry verbatim.
func (w Writer) SetData(ctx context.Context, key Key, data Log) error {
	return w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Log](key)).
		Change(func(_ gorp.Context, l Log) Log {
			data.Key = l.Key
			data.Name = l.Name
			return data
		}).Exec(ctx, w.tx)
}

// Dispatch applies a sequence of actions atomically to the log with the given key.
// After a successful update the actions are notified to the service-level observer so
// subscribers (cluster signals) can broadcast them. dispatchKey is a client-generated
// identifier carried verbatim onto the broadcast so the originating client can match
// its own echo against the set of outstanding local replays and skip a redundant
// reduce when no foreign action interleaved.
func (w Writer) Dispatch(
	ctx context.Context,
	key Key,
	dispatchKey string,
	actions []Action,
) error {
	if err := w.table.NewUpdate().Where(gorp.MatchKeys[Key, Log](key)).
		ChangeErr(func(_ gorp.Context, l Log) (Log, error) {
			return Reduce(l, actions...)
		}).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Notify(ctx, key, dispatchKey, actions)
	return nil
}

// Delete deletes the logs with the given keys.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	if err := w.
		table.
		NewDelete().
		Where(gorp.MatchKeys[Key, Log](keys...)).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	for _, key := range keys {
		if err := w.otgWriter.DeleteResource(ctx, OntologyID(key)); err != nil {
			return err
		}
	}
	return nil
}
