// Copyright 2025 Synnax Labs, Inc.
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

// Create creates or updates a view within the DB. If the View already has a key and
// an existing View already exists with that key, the existing view will be updated.
func (w Writer) Create(
	ctx context.Context,
	s *View,
) error {
	return w.CreateWithParent(ctx, s, ontology.ID{})
}

// CreateWithParent creates or updates a view as a child of the ontology.Resource with the given
// ID. If the view already exists and a parent is provided, the existing parent relationship
// will be deleted and a new parent relationship will be created. If the view already exists
// and no parent is provided, the existing parent relationship will be preserved. If an empty
// parent is provided, the view will be created under the top level "Views" group.
func (w Writer) CreateWithParent(
	ctx context.Context,
	s *View,
	parent ontology.ID,
) (err error) {
	hasParent := !parent.IsZero()
	if !hasParent {
		parent = w.group.OntologyID()
	}
	if err = w.validate(*s); err != nil {
		return
	}
	exists := false
	if s.Key == uuid.Nil {
		s.Key = uuid.New()
	} else {
		exists, err = gorp.NewRetrieve[uuid.UUID, View]().WhereKeys(s.Key).Exists(ctx, w.tx)
		if err != nil {
			return
		}
	}
	if err = gorp.NewCreate[uuid.UUID, View]().Entry(s).Exec(ctx, w.tx); err != nil {
		return
	}
	otgID := OntologyID(s.Key)
	if err = w.otgWriter.DefineResource(ctx, otgID); err != nil {
		return
	}
	// View already exists and parent provided = delete incoming relationships and define new parent
	// View already exists and no parent provided = do nothing
	// View does not exist = define parent
	if exists && hasParent {
		if hasRel, err := w.otgWriter.HasRelationship(ctx, parent, ontology.ParentOf, otgID); hasRel || err != nil {
			return err
		}
		if err = w.otgWriter.DeleteIncomingRelationshipsOfType(ctx, otgID, ontology.ParentOf); err != nil {
			return
		}
		if err = w.otgWriter.DefineRelationship(ctx, parent, ontology.ParentOf, otgID); err != nil {
			return
		}
	} else if !exists {
		if err = w.otgWriter.DefineRelationship(ctx, parent, ontology.ParentOf, otgID); err != nil {
			return
		}
	}
	return
}

// CreateMany creates or updates multiple views within the DB. If any of the views already
// exist, they will be updated.
func (w Writer) CreateMany(
	ctx context.Context,
	ss *[]View,
) (err error) {
	for i, s := range *ss {
		if err = w.Create(ctx, &s); err != nil {
			return
		}
		(*ss)[i] = s
	}
	return err
}

// CreateManyWithParent creates or updates multiple views within the DB as child views of
// the ontology.Resource with the given ID. If any of the views already exist, they will be
// updated. If the view already exists and a parent is provided, the existing parent relationship
// will be deleted and a new parent relationship will be created. If the view already exists and
// no parent is provided, the existing parent relationship will be preserved. If an empty parent is
// provided, the view will be created under the top level "Views" group.
func (w Writer) CreateManyWithParent(
	ctx context.Context,
	ss *[]View,
	parent ontology.ID,
) (err error) {
	if ss == nil {
		return
	}
	for i, s := range *ss {
		if err = w.CreateWithParent(ctx, &s, parent); err != nil {
			return
		}
		(*ss)[i] = s
	}
	return err
}

// Delete deletes the view with the given key. Delete is idempotent.
func (w Writer) Delete(ctx context.Context, key uuid.UUID) error {
	if err := gorp.NewDelete[uuid.UUID, View]().WhereKeys(key).Exec(ctx, w.tx); err != nil && !errors.Is(err, query.NotFound) {
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

func (w Writer) validate(s View) error {
	v := validate.New("view.View")
	validate.NotEmptyString(v, "Name", s.Name)
	return v.Error()
}
