// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranger

import (
	"context"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/group"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// Writer is used to create ranges within the DB.
type Writer struct {
	tx        gorp.Tx
	otgWriter ontology.Writer
	otg       *ontology.Ontology
	group     group.Group
}

// Create creates a new range within the DB, assigning it a unique key if it does not
// already have one. If the Range already has a key and an existing Range already
// exists with that key, the existing range will be updated.
func (w Writer) Create(
	ctx context.Context,
	r *Range,
) error {
	return w.CreateWithParent(ctx, r, ontology.ID{})
}

// CreateWithParent creates a new range as a child range of the ontology.Resource with the given
// ID. If the range does not already have a key, a new key will be assigned. If the range
// already exists, it will be updated. If the range already exists and a parent is
// provided, the existing parent relationship will be deleted and a new parent
// relationship will be created. If the range already exists and no parent is provided,
// the existing parent relationship will be preserved. If an empty parent is provided,
// the range will be created under the top level "Ranges" group.
func (w Writer) CreateWithParent(
	ctx context.Context,
	r *Range,
	parent ontology.ID,
) (err error) {
	hasParent := !parent.IsZero()
	if !hasParent {
		parent = w.group.OntologyID()
	}
	if r.Key == uuid.Nil {
		r.Key = uuid.New()
	}
	if err = w.validate(*r); err != nil {
		return
	}
	exists, err := gorp.NewRetrieve[uuid.UUID, Range]().WhereKeys(r.Key).Exists(ctx, w.tx)
	if err = gorp.NewCreate[uuid.UUID, Range]().Entry(r).Exec(ctx, w.tx); err != nil {
		return
	}
	otgID := OntologyID(r.Key)
	if err = w.otgWriter.DefineResource(ctx, otgID); err != nil {
		return
	}
	// Range already exists and parent provided  = delete incoming relationships and define new parent
	// Range already exists and no parent provided = do nothing
	// Range does not exist = define parent
	if exists && hasParent {
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
	r.tx = w.tx
	r.otg = w.otg
	return
}

// CreateMany creates multiple ranges within the DB. If any of the ranges already exist,
// they will be updated.
func (w Writer) CreateMany(
	ctx context.Context,
	rs *[]Range,
) (err error) {
	for i, r := range *rs {
		if err = w.Create(ctx, &r); err != nil {
			return
		}
		(*rs)[i] = r
	}
	return err
}

// CreateManyWithParent creates multiple ranges within the DB as child ranges of the ontology.Resource
// with the given ID. If any of the ranges already exist, they will be updated. If the range
// already exists and a parent is provided, the existing parent relationship will be deleted
// and a new parent relationship will be created. If the range already exists and no parent
// is provided, the existing parent relationship will be preserved. If an empty parent is
// provided, the range will be created under the top level "Ranges" group.
func (w Writer) CreateManyWithParent(
	ctx context.Context,
	rs *[]Range,
	parent ontology.ID,
) (err error) {
	if rs == nil {
		return
	}
	for i, r := range *rs {
		if err = w.CreateWithParent(ctx, &r, parent); err != nil {
			return
		}
		(*rs)[i] = r
	}
	return err
}

// Rename renames the range with the given key.
func (w Writer) Rename(
	ctx context.Context,
	key uuid.UUID,
	name string,
) error {
	return gorp.NewUpdate[uuid.UUID, Range]().WhereKeys(key).Change(func(r Range) Range {
		r.Name = name
		return r
	}).Exec(ctx, w.tx)
}

// Delete deletes the range with the given key. Delete will also delete all children
// of the range. Delete is idempotent.
func (w Writer) Delete(ctx context.Context, key uuid.UUID) error {
	// Query the ontology to find all children of the range and delete them as well
	var children []ontology.Resource
	if err := w.otgWriter.NewRetrieve().
		WhereIDs(OntologyID(key)).
		TraverseTo(ontology.Children).
		Entries(&children).
		ExcludeFieldData(true).
		// The check for query.NotFound is necessary because the child may have already
		// been deleted, and delete is idempotent.
		Exec(ctx, w.tx); err != nil && !errors.Is(err, query.NotFound) {
		return err
	}
	keys := lo.FilterMap(children, func(r ontology.Resource, _ int) (string, bool) {
		// Don't delete anything that's not a child range
		if r.ID.Type != OntologyType {
			return "", false
		}
		return r.ID.Key, true
	})
	for _, k := range keys {
		uK, err := uuid.Parse(k)
		if err != nil {
			return err
		}
		if err = w.Delete(ctx, uK); err != nil {
			return err
		}
	}
	if err := gorp.NewDelete[uuid.UUID, Range]().WhereKeys(key).Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.otgWriter.DeleteResource(ctx, OntologyID(key))
}

func (w Writer) validate(r Range) error {
	v := validate.New("ranger.Range")
	validate.NotNil(v, "Task", r.Key)
	validate.NotEmptyString(v, "Name", r.Name)
	validate.NonZero(v, "TimeRange.Start", r.TimeRange.Start)
	validate.NonZero(v, "TimeRange.end", r.TimeRange.End)
	return v.Error()
}
