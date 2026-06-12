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
	"github.com/synnaxlabs/x/set"
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
// parented to that range via the parent-of relationship; only r.Parent.Key is consulted
// — the parent's other fields are ignored and the parent must already exist (it is not
// validated or written). If r.Parent is nil on update, the existing parent relationship
// is preserved; on create it is left undefined. If r.Labels is non-empty, a LabeledBy
// relationship is defined from the range to each label; existing relationships are
// preserved. If the range does not already have a key, a new key will be assigned. If
// the range already exists and r.Parent is non-nil, the existing parent relationship
// will be replaced.
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
	if err != nil {
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
		labelKeys := lo.Map(r.Labels, func(l label.Label, _ int) label.Key {
			return l.Key
		})
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

// Delete deletes the ranges with the given keys. Delete also recursively removes every
// descendant range (via the parent-of relationship). Delete is idempotent: missing keys
// are silently ignored. The full subtree is collected in a single batched breadth-first
// sweep over the ontology — one query per tree level rather than one per node — and the
// underlying Gorp and ontology deletes are issued as bulk operations.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	if len(keys) == 0 {
		return nil
	}
	toDelete := set.New(keys...)
	frontier := keys
	for len(frontier) > 0 {
		var children []ontology.Resource
		if err := w.otgWriter.
			NewRetrieve().
			WhereIDs(OntologyIDs(frontier)...).
			TraverseTo(ontology.ChildrenTraverser).
			Entries(&children).
			ExcludeFieldData(true).
			// The check for query.ErrNotFound is necessary because the range may
			// already have been deleted, and delete is idempotent.
			Exec(ctx, w.tx); err != nil && !errors.Is(err, query.ErrNotFound) {
			return err
		}
		nextFrontier := make([]Key, 0, len(children))
		for _, c := range children {
			// Don't traverse into anything that's not a child range.
			if c.ID.Type != ontology.ResourceTypeRange {
				continue
			}
			k, err := KeyFromOntologyID(c.ID)
			if err != nil {
				return err
			}
			if toDelete.Contains(k) {
				continue
			}
			toDelete.Add(k)
			nextFrontier = append(nextFrontier, k)
		}
		frontier = nextFrontier
	}
	allKeys := toDelete.Slice()
	if err := w.table.
		NewDelete().
		Where(gorp.MatchKeys[Key, Range](allKeys...)).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.otgWriter.DeleteResource(ctx, OntologyIDs(allKeys)...)
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
