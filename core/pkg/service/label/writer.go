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
	tx    gorp.Tx
	otg   ontology.Writer
	table *gorp.Table[Key, Label]
}

// Create creates a new label, assigning it a unique key if one is not provided. If
// a label with the same key already exists, it will be overwritten.
func (w Writer) Create(ctx context.Context, l *Label) error {
	if l.Key == uuid.Nil {
		l.Key = uuid.New()
	}
	if err := l.Validate(); err != nil {
		return err
	}
	if err := w.table.NewCreate().Entry(l).Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.otg.DefineResources(ctx, OntologyID(l.Key))
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
	if err := w.table.NewDelete().Where(gorp.MatchKeys[Key, Label](keys...)).Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.otg.DeleteResources(ctx, OntologyIDs(keys)...)
}

// Label assigns a set of labels to the target resource. If the target resource already
// has labels, Label will add the new labels to the existing set.
func (w Writer) Label(ctx context.Context, target ontology.ID, labels []Key) error {
	for _, label := range labels {
		if err := w.otg.DefineRelationships(ctx, target, OntologyRelationshipTypeLabeledBy, OntologyID(label)); err != nil {
			return err
		}
	}
	return nil
}

// Clear removes all labels from the target resource.
func (w Writer) Clear(ctx context.Context, target ontology.ID) error {
	return w.otg.DeleteOutgoingRelationshipsOfType(ctx, target, OntologyRelationshipTypeLabeledBy)
}

// RemoveLabel removes a set of labels from the target resource. RemoveLabel is
// idempotent, and will not return an error if the target resource does not have the
// specified labels.
func (w Writer) RemoveLabel(
	ctx context.Context,
	target ontology.ID,
	labels []Key,
) error {
	for _, label := range labels {
		if err := w.otg.DeleteRelationship(ctx, target, OntologyRelationshipTypeLabeledBy, OntologyID(label)); err != nil {
			return err
		}
	}
	return nil
}
