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
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/project"
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

// Create creates the given log within the project provided. If the log does not have a
// key, a new key will be generated. If projectKey is uuid.Nil, the log is created
// without a project ParentOf relationship; this is used by imports that supply no
// parent project.
func (w Writer) Create(ctx context.Context, projectKey project.Key, l *Log) error {
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
	l.ApplyDefaults()
	if err = l.Validate(); err != nil {
		return err
	}
	if err = w.table.NewCreate().Entry(l).Exec(ctx, w.tx); err != nil {
		return err
	}
	if !exists {
		otgID := l.OntologyID()
		if err = w.otgWriter.DefineResources(ctx, otgID); err != nil {
			return err
		}
		if projectKey != uuid.Nil {
			if err = w.otgWriter.DefineRelationships(
				ctx,
				project.OntologyID(projectKey),
				ontology.RelationshipTypeParentOf,
				otgID,
			); err != nil {
				return err
			}
		}
	}
	// Notify last: a create rejected by ontology validation must not be broadcast.
	w.dispatcher.Notify(
		ctx, l.Key, "", []Action{NewCreateAction(CreatePayload{Log: *l})},
	)
	return nil
}

// CreateMany creates the given logs within the project provided. If logs with the
// same key already exist, they will be overwritten.
func (w Writer) CreateMany(ctx context.Context, projectKey project.Key, logs *[]Log) error {
	for i := range *logs {
		if err := w.Create(ctx, projectKey, &(*logs)[i]); err != nil {
			return err
		}
	}
	return nil
}

// Dispatch applies a sequence of actions atomically to the log with the given key.
// After a successful update the actions are notified to the service-level observer so
// subscribers (cluster signals) can broadcast them. dispatchKey is a client-generated
// identifier carried verbatim onto the broadcast so the originating client can match
// its own echo against the set of outstanding local replays and skip a redundant reduce
// when no foreign action interleaved.
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
	return w.otgWriter.DeleteResources(ctx, OntologyIDs(keys)...)
}
