// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/validate"
)

// retrieveGroup returns the group identified by key together with its children. It
// returns query.ErrNotFound if no group has key.
func (s *Service) retrieveGroup(
	ctx context.Context,
	tx gorp.Tx,
	key group.Key,
) (root ontology.Resource, children []ontology.Resource, err error) {
	err = s.cfg.Ontology.NewRetrieve().
		WhereIDs(group.OntologyID(key)).
		Entry(&root).
		TraverseTo(ontology.ChildrenTraverser).
		Entries(&children).
		Exec(ctx, tx)
	return root, children, err
}

// RetrieveGroupSymbols returns the ontology ID of every symbol in the group identified
// by key, reading through tx. It returns query.ErrNotFound if no group has key, and a
// validation error if the group holds a child that is not a schematic symbol.
//
// Callers that go on to write must pass the same tx they write under, so the IDs they
// act on and the IDs the write touches come from one snapshot.
func (s *Service) RetrieveGroupSymbols(
	ctx context.Context,
	tx gorp.Tx,
	key group.Key,
) ([]ontology.ID, error) {
	root, children, err := s.retrieveGroup(ctx, tx, key)
	if err != nil {
		return nil, err
	}
	ids, err := symbolIDs(children)
	if err != nil {
		return nil, errors.Wrapf(err, "group %q", root.Name)
	}
	return ids, nil
}

// DeleteGroup deletes the group identified by key together with the symbols that ids
// names. Pass the ids RetrieveGroupSymbols returned under tx, so the symbols removed
// and the symbols the caller enforced access on come from one snapshot.
//
// Symbols are deleted first because the group service refuses to delete a group that
// still holds children.
func (s *Service) DeleteGroup(
	ctx context.Context,
	tx gorp.Tx,
	key group.Key,
	ids []ontology.ID,
) error {
	keys, err := KeysFromOntologyIDs(ids)
	if err != nil {
		return err
	}
	if err = s.NewWriter(tx).Delete(ctx, keys...); err != nil {
		return err
	}
	return s.cfg.Group.NewWriter(tx).Delete(ctx, key)
}

// symbolIDs returns the ontology ID of every child, and a validation error naming the
// first child that is not a schematic symbol.
func symbolIDs(children []ontology.Resource) ([]ontology.ID, error) {
	ids := make([]ontology.ID, 0, len(children))
	for _, child := range children {
		if child.ID.Type != ontology.ResourceTypeSchematicSymbol {
			return nil, errors.Wrapf(
				validate.ErrValidation,
				"child %s is not a schematic symbol",
				child.ID,
			)
		}
		ids = append(ids, child.ID)
	}
	return ids, nil
}
