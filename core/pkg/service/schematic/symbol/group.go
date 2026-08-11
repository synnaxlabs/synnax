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
	"github.com/synnaxlabs/x/gorp"
	"go.uber.org/zap"
)

// retrieveGroup returns the group identified by key together with its children. It
// returns query.ErrNotFound if no group has key.
func (s *Service) retrieveGroup(
	ctx context.Context,
	tx gorp.Tx,
	key group.Key,
) (root ontology.Resource, children []ontology.Resource, err error) {
	if err = s.cfg.Ontology.NewRetrieve().
		WhereIDs(group.OntologyID(key)).
		Entry(&root).
		TraverseTo(ontology.ChildrenTraverser).
		Entries(&children).
		Exec(ctx, tx); err != nil {
		return ontology.Resource{}, nil, err
	}
	return root, children, nil
}

// RetrieveGroupSymbols returns the ontology ID of every symbol in the group identified
// by key, reading through tx. Children that are not schematic symbols are skipped and
// logged as a warning. It returns query.ErrNotFound if no group has key.
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
	symbols, skipped := partitionSymbols(children)
	s.warnSkipped(root.Name, skipped)
	return lo.Map(symbols, func(r ontology.Resource, _ int) ontology.ID {
		return r.ID
	}), nil
}

// DeleteGroup deletes the group identified by key together with the symbols that ids
// names. Children that are not schematic symbols survive: they move to the permanent
// symbol group before the group is deleted. Pass the ids RetrieveGroupSymbols returned
// under tx, so the symbols removed and the symbols the caller enforced access on come
// from one snapshot.
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
	if err = s.rescueStrays(ctx, tx, key); err != nil {
		return err
	}
	return s.cfg.Group.NewWriter(tx).Delete(ctx, key)
}

// rescueStrays moves any children the delete skipped to the permanent symbol group, so
// the children survive the delete and the group service's no-children rule passes.
func (s *Service) rescueStrays(ctx context.Context, tx gorp.Tx, key group.Key) error {
	groupID := group.OntologyID(key)
	var strays []ontology.Resource
	if err := s.cfg.Ontology.NewRetrieve().
		WhereIDs(groupID).
		TraverseTo(ontology.ChildrenTraverser).
		ExcludeFieldData(true).
		Entries(&strays).
		Exec(ctx, tx); err != nil {
		return err
	}
	if len(strays) == 0 {
		return nil
	}
	ids := lo.Map(strays, func(r ontology.Resource, _ int) ontology.ID { return r.ID })
	w := s.cfg.Ontology.NewWriter(tx)
	rels := lo.Map(ids, func(id ontology.ID, _ int) ontology.Relationship {
		return ontology.Relationship{
			From: groupID,
			Type: ontology.RelationshipTypeParentOf,
			To:   id,
		}
	})
	if err := w.DeleteRelationships(ctx, rels...); err != nil {
		return err
	}
	return w.DefineRelationships(
		ctx, s.group.OntologyID(), ontology.RelationshipTypeParentOf, ids...,
	)
}

// partitionSymbols splits children into schematic symbols and the IDs of children of
// any other type.
func partitionSymbols(
	children []ontology.Resource,
) (symbols []ontology.Resource, skipped []ontology.ID) {
	for _, child := range children {
		if child.ID.Type == ontology.ResourceTypeSchematicSymbol {
			symbols = append(symbols, child)
		} else {
			skipped = append(skipped, child.ID)
		}
	}
	return symbols, skipped
}

// warnSkipped logs the children a group operation ignored because they are not
// schematic symbols.
func (s *Service) warnSkipped(groupName string, skipped []ontology.ID) {
	if len(skipped) == 0 {
		return
	}
	s.cfg.L.Warn(
		"skipping children that are not schematic symbols",
		zap.String("group", groupName),
		zap.Stringers("children", skipped),
	)
}
