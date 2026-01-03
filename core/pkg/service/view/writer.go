// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package view

import (
	"context"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// Writer is used to create and update views within the DB.
type Writer struct {
	tx        gorp.Tx
	otgWriter ontology.Writer
	otg       *ontology.Ontology
	group     group.Group
}

// Create creates or updates a view within the DB. If the view already has a key and an
// existing view already exists with that key, the existing view will be updated.
func (w Writer) Create(ctx context.Context, view *View) error {
	if err := w.validate(*view); err != nil {
		return err
	}
	if view.Key == uuid.Nil {
		view.Key = uuid.New()
	}
	if err := gorp.
		NewCreate[uuid.UUID, View]().
		Entry(view).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	otgID := OntologyID(view.Key)
	if err := w.otgWriter.DefineResource(ctx, otgID); err != nil {
		return err
	}
	return w.otgWriter.DefineRelationship(
		ctx,
		w.group.OntologyID(),
		ontology.ParentOf,
		otgID,
	)
}

// CreateMany creates or updates multiple views within the DB. If any of the views
// already exist, they will be updated.
func (w Writer) CreateMany(ctx context.Context, views *[]View) error {
	for i, view := range *views {
		if err := w.Create(ctx, &view); err != nil {
			return err
		}
		(*views)[i] = view
	}
	return nil
}

// Delete deletes the view with the given key. Delete is idempotent.
func (w Writer) Delete(ctx context.Context, key uuid.UUID) error {
	if err := gorp.
		NewDelete[uuid.UUID, View]().
		WhereKeys(key).
		Exec(ctx, w.tx); err != nil && !errors.Is(err, query.NotFound) {
		return err
	}
	return w.otgWriter.DeleteResource(ctx, OntologyID(key))
}

// DeleteMany deletes multiple views with the given keys. DeleteMany is idempotent.
func (w Writer) DeleteMany(ctx context.Context, keys ...uuid.UUID) error {
	for _, key := range keys {
		if err := w.Delete(ctx, key); err != nil {
			return err
		}
	}
	return nil
}

func (w Writer) validate(v View) error {
	vld := validate.New("view.view")
	validate.NotEmptyString(vld, "name", v.Name)
	validate.NotEmptyString(vld, "type", v.Type)
	return vld.Error()
}
