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

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
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
// by key. Children of another type are omitted. It returns query.ErrNotFound if no
// group has key.
func (s *Service) RetrieveGroupSymbols(
	ctx context.Context,
	key group.Key,
) ([]ontology.ID, error) {
	_, children, err := s.retrieveGroup(ctx, nil, key)
	if err != nil {
		return nil, err
	}
	return symbolIDs(children), nil
}

// DeleteGroup deletes the group identified by key and every symbol in it. It returns
// query.ErrNotFound if no group has key, and an error if the service was opened without
// a group service, which leaves it unable to delete a group.
//
// Symbols are deleted first because the group service refuses to delete a group that
// still holds children. A group holding anything other than symbols therefore fails
// with a validation error, and tx rolls the symbol deletes back with it.
func (s *Service) DeleteGroup(ctx context.Context, tx gorp.Tx, key group.Key) error {
	if s.cfg.Group == nil {
		return errors.New("symbol service was opened without a group service")
	}
	_, children, err := s.retrieveGroup(ctx, tx, key)
	if err != nil {
		return err
	}
	keys, err := KeysFromOntologyIDs(symbolIDs(children))
	if err != nil {
		return err
	}
	if err = s.NewWriter(tx).Delete(ctx, keys...); err != nil {
		return err
	}
	return s.cfg.Group.NewWriter(tx).Delete(ctx, key)
}

func symbolIDs(resources []ontology.Resource) []ontology.ID {
	ids := lo.FilterMap(
		resources,
		func(r ontology.Resource, _ int) (ontology.ID, bool) {
			return r.ID, r.ID.Type == ontology.ResourceTypeSchematicSymbol
		},
	)
	return ids
}
