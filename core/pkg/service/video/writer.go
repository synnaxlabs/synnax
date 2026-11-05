// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package video

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/project"
)

// Writer creates and deletes video views in the transaction given to
// Service.NewWriter.
type Writer struct {
	tx         gorp.Tx
	otgWriter  ontology.Writer
	table      *gorp.Table[Key, Video]
	dispatcher actions.Dispatcher[Key, Action]
}

// WithDispatchKey returns a copy of the writer that puts dispatchKey on the create
// broadcasts it sends, so the client that sent the create recognizes its own echo.
func (w Writer) WithDispatchKey(dispatchKey string) Writer {
	w.dispatcher = w.dispatcher.WithDispatchKey(dispatchKey)
	return w
}

// Create creates v under the project with projectKey, or under no project when
// projectKey is uuid.Nil(). A view with no key gets a new one. A view with the key of
// an existing view replaces it.
func (w Writer) Create(ctx context.Context, projectKey project.Key, v *Video) error {
	var (
		exists bool
		err    error
	)
	if v.Key == uuid.Nil() {
		v.Key = uuid.New()
	} else if exists, err = w.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, Video](v.Key)).
		Exists(ctx, w.tx); err != nil {
		return err
	}
	if err = v.Validate(); err != nil {
		return err
	}
	if err = w.table.NewCreate().Entry(v).Exec(ctx, w.tx); err != nil {
		return err
	}
	if !exists {
		otgID := v.OntologyID()
		if err = w.otgWriter.DefineResources(ctx, otgID); err != nil {
			return err
		}
		if projectKey != uuid.Nil() {
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
	w.dispatcher.NotifyOnCommit(
		ctx,
		w.tx,
		v.Key,
		NewCreateAction(CreatePayload{Video: *v}),
	)
	return nil
}

// CreateMany creates each of videos under the project with projectKey.
func (w Writer) CreateMany(
	ctx context.Context,
	projectKey project.Key,
	videos *[]Video,
) error {
	for i := range *videos {
		if err := w.Create(ctx, projectKey, &(*videos)[i]); err != nil {
			return err
		}
	}
	return nil
}

// Delete deletes the video views with the given keys.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	if err := w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Video](keys...)).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.otgWriter.DeleteResources(ctx, OntologyIDs(keys)...)
}
