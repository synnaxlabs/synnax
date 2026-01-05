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

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/gorp"
)

// Writer wraps a transaction to create, update, and delete labels.
type Writer struct {
	tx  gorp.Tx
	otg ontology.Writer
}

// Create creates a new label, assigning it a unique key if one is not provided. If
// a label with the same key already exists, it will be overwritten.
func (w Writer) Create(
	ctx context.Context,
	l *Label,
) (err error) {
	if l.Key == uuid.Nil {
		l.Key = uuid.New()
	}
	if err = gorp.NewCreate[uuid.UUID, Label]().Entry(l).Exec(ctx, w.tx); err != nil {
		return
	}
	return w.otg.DefineResource(ctx, OntologyID(l.Key))
}

// CreateMany creates multiple labels in a single transaction. If any of the labels
// exist, they will be overwritten.
func (w Writer) CreateMany(
	ctx context.Context,
	ls *[]Label,
) (err error) {
	for i, l := range *ls {
		if err = w.Create(ctx, &l); err != nil {
			return
		}
		(*ls)[i] = l
	}
	return err
}

// Delete removes a label from the database and ontology. Delete is idempotent, and will
// not return an error if the label does not exist.
func (w Writer) Delete(
	ctx context.Context,
	k uuid.UUID,
) (err error) {
	if err = gorp.NewDelete[uuid.UUID, Label]().WhereKeys(k).Exec(ctx, w.tx); err != nil {
		return
	}
	return w.otg.DeleteResource(ctx, OntologyID(k))
}

// DeleteMany removes multiple labels from the database and ontology.
func (w Writer) DeleteMany(
	ctx context.Context,
	ks []uuid.UUID,
) (err error) {
	for _, k := range ks {
		if err = w.Delete(ctx, k); err != nil {
			return
		}
	}
	return err
}

// Label assigns a set of labels to the target resource. If the target resource already
// has labels, Label will add the new labels to the existing set.
func (w Writer) Label(
	ctx context.Context,
	target ontology.ID,
	labels []uuid.UUID,
) error {
	for _, label := range labels {
		if err := w.otg.DefineRelationship(ctx, target, LabeledBy, OntologyID(label)); err != nil {
			return err
		}
	}
	return nil
}

// Clear removes all labels from the target resource.
func (w Writer) Clear(
	ctx context.Context,
	target ontology.ID,
) error {
	return w.otg.DeleteOutgoingRelationshipsOfType(ctx, target, LabeledBy)
}

// RemoveLabel removes a set of labels from the target resource. RemoveLabel is idempotent,
// and will not return an error if the target resource does not have the specified labels.
func (w Writer) RemoveLabel(
	ctx context.Context,
	target ontology.ID,
	labels []uuid.UUID,
) error {
	for _, label := range labels {
		if err := w.otg.DeleteRelationship(ctx, target, LabeledBy, OntologyID(label)); err != nil {
			return err
		}
	}
	return nil
}
