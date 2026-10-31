// Copyright 2026 Synnax Labs, Inc.
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
	"encoding/json"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// clearWindow is how long a cleared problem stays the same problem: a status that goes
// bad again within it keeps the problem's since.
const clearWindow = 30 * telem.Second

// Writer is used to create and update statuses within the DB. Every write touches both
// the status entry and its ontology resource, and the two land together only when the
// Writer holds a transaction spanning them. Each set computes the Since and Cleared
// fields of a status from the stored status and ignores the values the caller sends.
type Writer struct {
	tx        gorp.Tx
	table     *gorp.Table[Key, Status[any]]
	otgWriter ontology.Writer
	otg       *ontology.Ontology
	group     group.Group
	now       func() telem.TimeStamp
}

// Set creates or updates a status within the DB. If the Status already has a key and
// an existing Status already exists with that key, the existing status will be updated.
func (w Writer) Set[D any](ctx context.Context, s *Status[D]) error {
	return w.SetWithParent(ctx, s, ontology.ID{})
}

// SetWithParent creates or updates a status as a child of the ontology.Resource with
// the given ID. If the status already exists and a parent is provided, the existing
// parent relationship will be deleted and a new parent relationship will be created. If
// the status already exists and no parent is provided, the existing parent relationship
// will be preserved. If an empty parent is provided, the status will be created under
// the top level "Statuses" group.
func (w Writer) SetWithParent[D any](
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
	// Raw details skip decoding the stored details, which the write replaces.
	var prev Status[json.RawMessage]
	err := gorp.NewRetrieve[Key, Status[json.RawMessage]]().
		Where(gorp.MatchKeys[Key, Status[json.RawMessage]](s.Key)).
		Entry(&prev).
		Exec(ctx, w.tx)
	exists := err == nil
	if err != nil && !errors.Is(err, query.ErrNotFound) {
		return err
	}
	markProblem(s, prev, w.now())
	if err = w.table.NewCreate().Entry(new(s.ToAny())).Exec(ctx, w.tx); err != nil {
		return err
	}
	otgID := s.OntologyID()
	if err = w.otgWriter.DefineResources(ctx, otgID); err != nil {
		return err
	}
	// Status already exists and parent provided = delete incoming relationships and
	// define new parent Status already exists and no parent provided = do nothing
	// Status does not exist = define parent
	if exists && hasParent {
		if hasRel, err := w.otg.RelationshipExists(ctx, w.tx, ontology.Relationship{
			From: parent,
			Type: ontology.RelationshipTypeParentOf,
			To:   otgID,
		}); hasRel || err != nil {
			return err
		}
		if err = w.otgWriter.DeleteIncomingRelationshipsOfType(
			ctx,
			otgID,
			ontology.RelationshipTypeParentOf,
		); err != nil {
			return err
		}
		if err = w.otgWriter.DefineRelationships(
			ctx, parent, ontology.RelationshipTypeParentOf, otgID,
		); err != nil {
			return err
		}
	} else if !exists {
		if err = w.otgWriter.DefineRelationships(
			ctx, parent, ontology.RelationshipTypeParentOf, otgID,
		); err != nil {
			return err
		}
	}
	return nil
}

// SetMany creates or updates multiple statuses within the DB. If any of the statuses
// already exist, they will be updated.
func (w Writer) SetMany[D any](ctx context.Context, statuses *[]Status[D]) error {
	return w.SetManyWithParent(ctx, statuses, ontology.ID{})
}

// SetManyWithParent creates or updates multiple statuses within the DB as child
// statuses of the ontology.Resource with the given ID. If any of the statuses already
// exist, they will be updated. If the status already exists and a parent is provided,
// the existing parent relationship will be deleted and a new parent relationship will
// be created. If the status already exists and no parent is provided, the existing
// parent relationship will be preserved. If an empty parent is provided, the status
// will be created under the top level "Statuses" group.
func (w Writer) SetManyWithParent[D any](
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

// Rename renames the status with the given key, keeping every other field. It returns
// query.ErrNotFound if no status with the key exists and a validation error if name is
// empty.
func (w Writer) Rename(ctx context.Context, key Key, name string) error {
	v := validate.New("status.status")
	v.NotEmptyString("name", name)
	if err := v.Error(); err != nil {
		return err
	}
	// Raw details round-trip byte for byte. Decoding them as any would round numbers
	// past float64 precision.
	var s Status[json.RawMessage]
	if err := gorp.NewRetrieve[Key, Status[json.RawMessage]]().
		Where(gorp.MatchKeys[Key, Status[json.RawMessage]](key)).
		Entry(&s).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	s.Name = name
	return w.table.NewCreate().Entry(new(s.ToAny())).Exec(ctx, w.tx)
}

// Delete deletes the statuses with the given keys. Delete is idempotent.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	if err := w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Status[any]](keys...)).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.otgWriter.DeleteResources(ctx, OntologyIDs(keys)...)
}

func (w Writer) validate[D any](s Status[D]) error {
	v := validate.New("status.status")
	v.NotEmptyString("key", s.Key)
	v.Positive("time", s.Time)
	v.NotEmptyString("variant", s.Variant)
	return v.Error()
}

// markProblem sets the Since and Cleared fields of s from prev, the stored status or
// the zero status when none is stored, at the time now. A problem is a period with a
// warning or error variant.
func markProblem[D any](
	s *Status[D],
	prev Status[json.RawMessage],
	now telem.TimeStamp,
) {
	since, cleared, wasProblem := prev.Since, prev.Cleared, isProblem(prev.Variant)
	switch {
	case isProblem(s.Variant):
		open := wasProblem || (cleared != nil && cleared.Span(now) < clearWindow)
		if since == nil || !open {
			since = new(now)
		}
		cleared = nil
	case wasProblem:
		cleared = new(now)
	}
	s.Since, s.Cleared = since, cleared
}

func isProblem(v Variant) bool { return v == VariantWarning || v == VariantError }
