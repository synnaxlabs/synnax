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
	"slices"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// Retrieve is a builder for querying labels.
type Retrieve struct {
	baseTX   gorp.Tx
	gorp     gorp.Retrieve[Key, Label]
	indexes  indexes
	ontology *ontology.Ontology
}

// MatchFor returns a filter that matches the labels attached to the resource with the
// given ID. A resource that does not exist has no labels.
func MatchFor(id resource.ID) Filter {
	return func(r Retrieve) gorp.Filter[Key, Label] {
		return gorp.MatchKeysFunc[Key, Label](func(ctx gorp.Context) ([]Key, error) {
			start := r.ontology.NewRetrieve().WhereIDs(id).SkipMissingIDs()
			return resolveFor(ctx, ctx.Tx, start)
		})
	}
}

// ForIDs returns the IDs that the for lines of w name, at any depth.
func (w Where) ForIDs() []resource.ID {
	var ids []resource.ID
	if w.For != nil {
		ids = append(ids, *w.For)
	}
	for _, c := range slices.Concat(w.And, w.Or) {
		ids = append(ids, c.ForIDs()...)
	}
	if w.Not != nil {
		ids = append(ids, w.Not.ForIDs()...)
	}
	return ids
}

func resolveFor(
	ctx context.Context,
	tx gorp.Tx,
	resource ontology.Retrieve,
) ([]Key, error) {
	var labels []ontology.Resource
	if err := resource.
		TraverseTo(LabelsOntologyTraverser).
		ExcludeFieldData(true).
		Entries(&labels).
		Exec(ctx, tx); err != nil {
		return nil, err
	}
	return KeysFromResourceIDs(ontology.ResourceIDs(labels))
}

// RetrieveFor retrieves all labels that are associated with the provided resource ID.
// If a tx is provided, RetrieveFor will use it to execute the query. Otherwise, it will
// execute against the underlying gorp.DB. A resource that does not exist has no labels.
// It returns query.ErrNotFound if one of the resource's labels does not exist.
func (s *Service) RetrieveFor(
	ctx context.Context,
	id resource.ID,
	tx gorp.Tx,
) ([]Label, error) {
	tx = gorp.OverrideTx(s.cfg.DB, tx)
	keys, err := resolveFor(ctx, tx, s.cfg.Ontology.NewRetrieve().WhereIDs(id))
	if err != nil {
		return nil, err
	}
	labels := make([]Label, 0, len(keys))
	if err := s.NewRetrieve().
		Where(MatchKeys(keys...)).
		Entries(&labels).
		Exec(ctx, tx); err != nil {
		return nil, err
	}
	return labels, nil
}
