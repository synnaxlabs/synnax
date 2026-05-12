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

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/validate"
)

// ErrMultipleMatches signals duplicate-name ambiguity. Reserved for a future
// strict-mode by-name upsert (e.g., ranges) that prefers erroring over the
// lenient write-to-first-match path UpsertByName takes today.
var ErrMultipleMatches = errors.New("multiple matches by name")

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
	exists, err := gorp.NewRetrieve[string, status.Status[D]]().Where(gorp.MatchKeys[string, status.Status[D]](s.Key)).Exists(ctx, w.tx)
	if err != nil {
		return err
	}
	if err = gorp.NewCreate[string, status.Status[D]]().Entry(s).Exec(ctx, w.tx); err != nil {
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
		if hasRel, err := w.otgWriter.HasRelationship(ctx, parent, ontology.RelationshipTypeParentOf, otgID); hasRel || err != nil {
			return err
		}
		if err = w.otgWriter.DeleteIncomingRelationshipsOfType(ctx, otgID, ontology.RelationshipTypeParentOf); err != nil {
			return err
		}
		if err = w.otgWriter.DefineRelationship(ctx, parent, ontology.RelationshipTypeParentOf, otgID); err != nil {
			return err
		}
	} else if !exists {
		if err = w.otgWriter.DefineRelationship(ctx, parent, ontology.RelationshipTypeParentOf, otgID); err != nil {
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

// Update applies the change function to the status with the given key and persists
// the result. Returns query.ErrNotFound if no status exists for the supplied key.
func (w Writer[D]) Update(
	ctx context.Context,
	key string,
	change func(*Status[D]) error,
) error {
	var s Status[D]
	if err := gorp.NewRetrieve[string, status.Status[D]]().
		Where(gorp.MatchKeys[string, status.Status[D]](key)).
		Entry(&s).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	if err := change(&s); err != nil {
		return err
	}
	return w.Set(ctx, &s)
}

// retrieveByName returns all statuses sharing the given name. Used by UpsertByName
// and DeleteByName to share the predicate-scan retrieve.
func (w Writer[D]) retrieveByName(ctx context.Context, name string) ([]Status[D], error) {
	var matches []Status[D]
	err := gorp.NewRetrieve[string, status.Status[D]]().
		Where(gorp.Match(func(_ gorp.Context, s *status.Status[D]) (bool, error) {
			return s.Name == name, nil
		})).
		Entries(&matches).
		Exec(ctx, w.tx)
	return matches, errors.Skip(err, query.ErrNotFound)
}

// UpsertByName updates the status with this name, or creates one if none exists.
// When multiple rows share the name, writes to the first by key order and returns
// multipleMatches=true so callers can surface the ambiguity.
func (w Writer[D]) UpsertByName(
	ctx context.Context,
	name string,
	change func(*Status[D]) error,
) (key string, multipleMatches bool, err error) {
	matches, err := w.retrieveByName(ctx, name)
	if err != nil {
		return "", false, err
	}
	var s Status[D]
	if len(matches) >= 1 {
		s = matches[0]
		multipleMatches = len(matches) > 1
	} else {
		s = Status[D]{Key: uuid.NewString(), Name: name, Variant: status.VariantInfo}
	}
	if err = change(&s); err != nil {
		return "", false, err
	}
	return s.Key, multipleMatches, w.Set(ctx, &s)
}

// DeleteByName deletes all statuses with the given name and returns the count
// of rows deleted so callers can distinguish not-found, single, and multi-match.
func (w Writer[D]) DeleteByName(ctx context.Context, name string) (count int, err error) {
	matches, err := w.retrieveByName(ctx, name)
	if err != nil {
		return 0, err
	}
	for _, m := range matches {
		if err = w.Delete(ctx, m.Key); err != nil {
			return 0, err
		}
	}
	return len(matches), nil
}

// Delete deletes the status with the given key. Delete is idempotent.
func (w Writer[D]) Delete(ctx context.Context, key string) error {
	if err := gorp.NewDelete[string, status.Status[D]]().
		Where(gorp.MatchKeys[string, status.Status[D]](key)).
		Exec(ctx, w.tx); err != nil && !errors.Is(err, query.ErrNotFound) {
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
