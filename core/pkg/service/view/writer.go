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
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// Writer is used to create and update views within the DB.
type Writer struct {
	tx        gorp.Tx
	otgWriter ontology.Writer
	otg       *ontology.Ontology
	group     group.Group
	table     *gorp.Table[Key, View]
}

// Create creates or updates a view within the DB. If the view already has a key and an
// existing view already exists with that key, the existing view will be updated.
func (w Writer) Create(ctx context.Context, view *View) error {
	if err := w.validate(*view); err != nil {
		return err
	}
	if view.GorpKey() == uuid.Nil() {
		*view = view.WithKey(uuid.New())
	}
	if err := w.table.NewCreate().
		Entry(view).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	otgID := view.OntologyID()
	if err := w.otgWriter.DefineResources(ctx, otgID); err != nil {
		return err
	}
	return w.otgWriter.DefineRelationships(
		ctx,
		w.group.OntologyID(),
		ontology.RelationshipTypeParentOf,
		otgID,
	)
}

// CreateMany creates or updates multiple views within the DB. If any of the views
// already exist, they will be updated.
func (w Writer) CreateMany(ctx context.Context, views *[]View) error {
	for i := range *views {
		if err := w.Create(ctx, &(*views)[i]); err != nil {
			return err
		}
	}
	return nil
}

// Rename renames the view with the given key, keeping every other field. It returns
// query.ErrNotFound if no view with the key exists and a validation error if name is
// empty.
func (w Writer) Rename(ctx context.Context, key Key, name string) error {
	return w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, View](key)).
		ChangeErr(func(_ gorp.Context, v View) (View, error) {
			v = v.WithName(name)
			if err := w.validate(v); err != nil {
				return View{}, err
			}
			return v, nil
		}).
		Exec(ctx, w.tx)
}

// Delete deletes the views with the given keys. Delete is idempotent.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	if err := w.table.NewDelete().Where(gorp.MatchKeys[Key, View](keys...)).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.otgWriter.DeleteResources(ctx, OntologyIDs(keys)...)
}

func (w Writer) validate(v View) error {
	vld := validate.New("view.view")
	vld.NotEmptyString("name", v.Base().Name)
	vld.NotEmptyString("type", v.Type())
	if err := vld.Error(); err != nil {
		return err
	}
	var err error
	switch variant := v.Variant.(type) {
	case RangeView:
		if variant.Where != nil {
			_, err = ranger.MatchWhere(*variant.Where)
		}
	case StatusView:
		if variant.Where != nil {
			_, err = status.MatchWhere[any](*variant.Where)
		}
	default:
		return errors.Newf("view %s has the unknown variant %T", v.Base().Key, variant)
	}
	return validate.PathedError(err, "where")
}
