// Copyright 2026 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/service/label"
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
	label     *label.Service
	table     *gorp.Table[Key, Range]
}

// Create creates or updates the given range. If r.Parent is non-nil, the range is
// parented to that range; otherwise its parent relationship is left unchanged on update
// and left undefined on create. If r.Labels is non-empty, a LabeledBy relationship is
// defined from the range to each label; existing relationships are preserved.
// If the range does not already have a key, a new key will be assigned. If the range
// already exists and r.Parent is non-nil, the existing parent relationship will be
// replaced.
func (w Writer) Create(ctx context.Context, r *Range) error {
	if r.Key == uuid.Nil {
		r.Key = uuid.New()
	}
	if err := w.validate(*r); err != nil {
		return err
	}
	exists, err := w.table.
		NewRetrieve().
		Where(gorp.MatchKeys[Key, Range](r.Key)).
		Exists(ctx, w.tx)
	if err != nil && !errors.Is(err, query.ErrNotFound) {
		return err
	}
	if err = w.table.NewCreate().Entry(r).Exec(ctx, w.tx); err != nil {
		return err
	}
	otgID := OntologyID(r.Key)
	if err = w.otgWriter.DefineResource(ctx, otgID); err != nil {
		return err
	}
	if len(r.Labels) > 0 {
		labelKeys := lo.Map(r.Labels, func(l label.Label, _ int) label.Key { return l.Key })
		if err = w.label.NewWriter(w.tx).Label(ctx, otgID, labelKeys); err != nil {
			return err
		}
	}
	if r.Parent == nil {
		return nil
	}
	parent := r.Parent.OntologyID()
	if exists {
		if relAlreadyExists, err := w.otgWriter.HasRelationship(
			ctx,
			parent,
			ontology.RelationshipTypeParentOf,
			otgID,
		); relAlreadyExists || err != nil {
			return err
		}
		if err = w.otgWriter.DeleteIncomingRelationshipsOfType(
			ctx,
			otgID,
			ontology.RelationshipTypeParentOf,
		); err != nil {
			return err
		}
	}
	return w.otgWriter.DefineRelationship(
		ctx,
		parent,
		ontology.RelationshipTypeParentOf,
		otgID,
	)
}

// CreateMany creates multiple ranges within the DB. If any of the ranges already exist,
// they will be updated. Each range's Parent field, if non-nil, is used to set its
// parent relationship.
func (w Writer) CreateMany(ctx context.Context, ranges *[]Range) error {
	for i := range *ranges {
		if err := w.Create(ctx, &(*ranges)[i]); err != nil {
			return err
		}
	}
	return nil
}

// Rename renames the range with the given key.
func (w Writer) Rename(ctx context.Context, key Key, name string) error {
	return w.table.
		NewUpdate().
		Where(gorp.MatchKeys[Key, Range](key)).
		Change(func(_ gorp.Context, r Range) Range { r.Name = name; return r }).
		Exec(ctx, w.tx)
}

// Delete deletes the range with the given key. Delete will also delete all children of
// the range. Delete is idempotent.
func (w Writer) Delete(ctx context.Context, key Key) error {
	// Query the ontology to find all children of the range and delete them as well
	var children []ontology.Resource
	if err := w.
		otgWriter.
		NewRetrieve().
		WhereIDs(OntologyID(key)).
		TraverseTo(ontology.ChildrenTraverser).
		Entries(&children).
		ExcludeFieldData(true).
		// The check for query.ErrNotFound is necessary because the child may have
		// already been deleted, and delete is idempotent.
		Exec(ctx, w.tx); err != nil && !errors.Is(err, query.ErrNotFound) {
		return err
	}
	keys := lo.FilterMap(children, func(r ontology.Resource, _ int) (string, bool) {
		// Don't delete anything that's not a child range
		if r.ID.Type != ontology.ResourceTypeRange {
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
	if err := w.table.
		NewDelete().
		Where(gorp.MatchKeys[Key, Range](key)).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.otgWriter.DeleteResource(ctx, OntologyID(key))
}

func (w Writer) validate(r Range) error {
	v := validate.New("ranger.range")
	validate.NotNil(v, "key", r.Key)
	validate.NotEmptyString(v, "name", r.Name)
	validate.NonZero(v, "time_range.start", r.TimeRange.Start)
	validate.NonZero(v, "time_range.end", r.TimeRange.End)
	v.Ternary(
		"time_range",
		r.TimeRange.Start.After(r.TimeRange.End),
		"time_range.start cannot be after time_range.end",
	)
	return v.Error()
}
