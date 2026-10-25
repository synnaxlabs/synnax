// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package label

import (
	"context"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
)

// Writer wraps a transaction to create, update, and delete labels.
type Writer struct {
	tx         gorp.Tx
	otg        ontology.Writer
	table      *gorp.Table[Key, Label]
	dispatcher actions.Dispatcher[Key, Action]
	// dispatchKey is stamped onto create broadcasts so the originator skips its echo.
	dispatchKey string
}

// WithDispatchKey returns a copy of the writer that stamps dispatchKey onto the create
// broadcasts it emits, so the client that sent the create recognizes its own echo.
func (w Writer) WithDispatchKey(dispatchKey string) Writer {
	w.dispatchKey = dispatchKey
	return w
}

// notify broadcasts the actions on the label with the given key after the transaction
// commits.
func (w Writer) notify(ctx context.Context, key Key, acts ...Action) {
	gorp.OnCommit(w.tx, func() { w.dispatcher.Notify(ctx, key, w.dispatchKey, acts) })
}

// Create creates a new label, assigning it a unique key if one is not provided. If
// a label with the same key already exists, it will be overwritten. After the
// transaction commits, Create broadcasts a create action that carries the stored label.
func (w Writer) Create(ctx context.Context, l *Label) error {
	if l.Key == uuid.Nil() {
		l.Key = uuid.New()
	}
	if err := l.Validate(); err != nil {
		return err
	}
	if err := w.table.NewCreate().Entry(l).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.notify(ctx, l.Key, NewCreateAction(CreatePayload{Label: *l}))
	return w.otg.DefineResources(ctx, l.OntologyID())
}

// apply applies the actions to the label with the given key and stores the result.
func (w Writer) apply(ctx context.Context, key Key, acts []Action) error {
	var prev Label
	if err := w.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, Label](key)).
		Entry(&prev).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	next, err := Reduce(prev, acts...)
	if err != nil {
		return err
	}
	if err = next.Validate(); err != nil {
		return err
	}
	return w.table.NewCreate().Entry(&next).Exec(ctx, w.tx)
}

// CreateMany creates multiple labels in a single transaction. If any of the labels
// exist, they will be overwritten.
func (w Writer) CreateMany(ctx context.Context, labels *[]Label) error {
	for i := range *labels {
		if err := w.Create(ctx, &(*labels)[i]); err != nil {
			return err
		}
	}
	return nil
}

// Delete removes a label from the database and ontology. Delete is idempotent, and will
// not return an error if the label does not exist.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	if err := w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Label](keys...)).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.otg.DeleteResources(ctx, OntologyIDs(keys)...)
}

// Label assigns a set of labels to the target resource. If the target resource already
// has labels, Label will add the new labels to the existing set.
func (w Writer) Label(ctx context.Context, target ontology.ID, labels []Key) error {
	return w.otg.DefineRelationships(
		ctx, target, OntologyRelationshipTypeLabeledBy, OntologyIDs(labels)...,
	)
}

// Replace makes the given labels the complete set of labels on the target resource.
// Labels the target already has are left untouched.
func (w Writer) Replace(ctx context.Context, target ontology.ID, labels []Key) error {
	return w.otg.ReplaceOutgoingRelationshipsOfType(
		ctx, target, OntologyRelationshipTypeLabeledBy, OntologyIDs(labels)...,
	)
}

// RemoveLabel removes a set of labels from the target resource. RemoveLabel is
// idempotent, and will not return an error if the target resource does not have the
// specified labels.
func (w Writer) RemoveLabel(
	ctx context.Context,
	target ontology.ID,
	labels []Key,
) error {
	return w.otg.DeleteRelationships(
		ctx,
		lo.Map(labels, func(key Key, _ int) ontology.Relationship {
			return ontology.Relationship{
				From: target,
				Type: OntologyRelationshipTypeLabeledBy,
				To:   OntologyID(key),
			}
		})...,
	)
}
