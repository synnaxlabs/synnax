// Copyright 2023 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/group"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/validate"
)

type Writer struct {
	tx    gorp.Tx
	otg   ontology.Writer
	group group.Group
}

func (w Writer) Create(
	ctx context.Context,
	r *Range,
) error {
	return w.CreateWithParent(ctx, r, ontology.ID{})
}

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
	if err = w.otg.DefineResource(ctx, otgID); err != nil {
		return
	}
	// Range already exists and parent provided  = delete incoming relationships and define new parent
	// Range already exists and no parent provided = do nothing
	// Range does not exist = define parent
	if exists && hasParent {
		if err = w.otg.DeleteIncomingRelationshipsOfType(ctx, otgID, ontology.ParentOf); err != nil {
			return
		}
		if err = w.otg.DefineRelationship(ctx, parent, ontology.ParentOf, otgID); err != nil {
			return
		}
	} else if !exists {
		if err = w.otg.DefineRelationship(ctx, parent, ontology.ParentOf, otgID); err != nil {
			return
		}
	}
	r.tx = w.tx
	return
}

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

func (w Writer) CreateManyWithParent(
	ctx context.Context,
	rs *[]Range,
	parent ontology.ID,
) (err error) {
	for i, r := range *rs {
		if err = w.CreateWithParent(ctx, &r, parent); err != nil {
			return
		}
		(*rs)[i] = r
	}
	return err
}

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

func (w Writer) Delete(ctx context.Context, key uuid.UUID) error {
	if err := gorp.NewDelete[uuid.UUID, Range]().WhereKeys(key).Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.otg.DeleteResource(ctx, OntologyID(key))
}

func (w Writer) validate(r Range) error {
	v := validate.New("ranger.Range")
	validate.NotNil(v, "Task", r.Key)
	validate.NotEmptyString(v, "Name", r.Name)
	validate.NonZero(v, "TimeRange.Start", r.TimeRange.Start)
	validate.NonZero(v, "TimeRange.end", r.TimeRange.End)
	return v.Error()
}
