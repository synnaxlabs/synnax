// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package status

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// Writer is used to create and update statuses within the DB.
type Writer[D any] struct {
	tx        gorp.Tx
	otgWriter ontology.Writer
	otg       *ontology.Ontology
	group     group.Group
}

// Set creates or updates a status within the DB. If the Status already has a key and
// an existing Status already exists with that key, the existing status will be updated.
func (w Writer[D]) Set(ctx context.Context, s *Status[D]) error {
	return w.SetWithParent(ctx, s, ontology.ID{})
}

// SetWithParent creates or updates a status as a child of the ontology.Resource with the given
// ID. If the status already exists and a parent is provided, the existing parent relationship
// will be deleted and a new parent relationship will be created. If the status already exists
// and no parent is provided, the existing parent relationship will be preserved. If an empty
// parent is provided, the status will be created under the top level "Statuses" group.
func (w Writer[D]) SetWithParent(
	ctx context.Context,
	s *Status[D],
	parent ontology.ID,
) error {
	hasParent := !parent.IsZero()
	if !hasParent {
		parent = w.group.OntologyID()
	}
	if err := w.validate(*s); err != nil {
		return err
	}
	exists, err := gorp.NewRetrieve[string, Status[D]]().WhereKeys(s.Key).Exists(ctx, w.tx)
	if err != nil {
		return err
	}
	if err = gorp.NewCreate[string, Status[D]]().Entry(s).Exec(ctx, w.tx); err != nil {
		return err
	}
	otgID := OntologyID(s.Key)
	if err = w.otgWriter.DefineResource(ctx, otgID); err != nil {
		return err
	}
	// Status already exists and parent provided = delete incoming relationships and define new parent
	// Status already exists and no parent provided = do nothing
	// Status does not exist = define parent
	if exists && hasParent {
		if hasRel, err := w.otgWriter.HasRelationship(ctx, parent, ontology.ParentOf, otgID); hasRel || err != nil {
			return err
		}
		if err = w.otgWriter.DeleteIncomingRelationshipsOfType(ctx, otgID, ontology.ParentOf); err != nil {
			return err
		}
		if err = w.otgWriter.DefineRelationship(ctx, parent, ontology.ParentOf, otgID); err != nil {
			return err
		}
	} else if !exists {
		if err = w.otgWriter.DefineRelationship(ctx, parent, ontology.ParentOf, otgID); err != nil {
			return err
		}
	}
	return nil
}

// SetMany creates or updates multiple statuses within the DB. If any of the statuses already
// exist, they will be updated.
func (w Writer[D]) SetMany(
	ctx context.Context,
	statuses *[]Status[D],
) error {
	for i, s := range *statuses {
		if err := w.Set(ctx, &s); err != nil {
			return err
		}
		(*statuses)[i] = s
	}
	return nil
}

// SetManyWithParent creates or updates multiple statuses within the DB as child statuses of
// the ontology.Resource with the given ID. If any of the statuses already exist, they will be
// updated. If the status already exists and a parent is provided, the existing parent relationship
// will be deleted and a new parent relationship will be created. If the status already exists and
// no parent is provided, the existing parent relationship will be preserved. If an empty parent is
// provided, the status will be created under the top level "Statuses" group.
func (w Writer[D]) SetManyWithParent(
	ctx context.Context,
	statuses *[]Status[D],
	parent ontology.ID,
) error {
	if statuses == nil {
		return nil
	}
	for i, s := range *statuses {
		if err := w.SetWithParent(ctx, &s, parent); err != nil {
			return err
		}
		(*statuses)[i] = s
	}
	return nil
}

// Delete deletes the status with the given key. Delete is idempotent.
func (w Writer[D]) Delete(ctx context.Context, key string) error {
	if err := gorp.NewDelete[string, Status[D]]().
		WhereKeys(key).
		Exec(ctx, w.tx); err != nil && !errors.Is(err, query.NotFound) {
		return err
	}
	return w.otgWriter.DeleteResource(ctx, OntologyID(key))
}

// DeleteMany deletes multiple statuses with the given keys. DeleteMany is idempotent.
func (w Writer[D]) DeleteMany(ctx context.Context, keys ...string) error {
	for _, key := range keys {
		if err := w.Delete(ctx, key); err != nil {
			return err
		}
	}
	return nil
}

func (w Writer[D]) validate(s Status[D]) error {
	v := validate.New("status.status")
	validate.NotEmptyString(v, "key", s.Key)
	validate.Positive(v, "time", s.Time)
	validate.NotEmptyString(v, "variant", s.Variant)
	return v.Error()
}
