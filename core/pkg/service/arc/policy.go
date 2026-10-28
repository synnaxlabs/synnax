// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"context"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

var _ ontology.Policy = (*Service)(nil)

// ValidateParent implements ontology.Policy. Below the Arcs group a child is a group or
// an Arc, a group name is an identifier, a name is unique in its directory, and a
// standard module name is reserved directly under Arcs. An Arc is rejected anywhere
// else.
func (s *Service) ValidateParent(
	ctx context.Context,
	tx gorp.Tx,
	parent, child ontology.ID,
) error {
	directory, err := s.isDirectory(ctx, tx, parent)
	if err != nil {
		return err
	}
	if !directory {
		if child.Type == ontology.ResourceTypeArc {
			return errors.Wrap(validate.ErrValidation, "an Arc can only go under Arcs")
		}
		return nil
	}
	if child.Type != ontology.ResourceTypeGroup &&
		child.Type != ontology.ResourceTypeArc {
		return errors.Wrapf(
			validate.ErrValidation, "a %s cannot go under Arcs", child.Type,
		)
	}
	var res ontology.Resource
	if err = s.cfg.Ontology.NewRetrieve().
		WhereIDs(child).
		Entry(&res).
		Exec(ctx, tx); err != nil {
		return err
	}
	return s.validateEntry(ctx, tx, parent, child, res.Name)
}

// ValidateName implements ontology.Policy.
func (s *Service) ValidateName(
	ctx context.Context,
	tx gorp.Tx,
	id ontology.ID,
	name string,
) error {
	if id.Type != ontology.ResourceTypeGroup && id.Type != ontology.ResourceTypeArc {
		return nil
	}
	parents, err := s.parents(ctx, tx, id)
	if err != nil {
		return err
	}
	for _, parent := range parents {
		directory, err := s.isDirectory(ctx, tx, parent)
		if err != nil {
			return err
		}
		if !directory {
			continue
		}
		if err = s.validateEntry(ctx, tx, parent, id, name); err != nil {
			return err
		}
	}
	return nil
}

// validateEntry checks name for the entry id placed in the directory parent.
func (s *Service) validateEntry(
	ctx context.Context,
	tx gorp.Tx,
	parent, id ontology.ID,
	name string,
) error {
	identifier := id.Type == ontology.ResourceTypeGroup
	if id.Type == ontology.ResourceTypeArc {
		module, err := s.isModule(ctx, tx, id)
		if err != nil {
			return err
		}
		identifier = module
	}
	if identifier && !validate.IsIdentifier(name) {
		return errors.Wrapf(
			validate.ErrValidation,
			"name %q must contain only letters, digits, and underscores",
			name,
		)
	}
	if parent == s.group.OntologyID() && s.reserved.Contains(name) {
		return errors.Wrapf(
			validate.ErrValidation,
			"%q is a standard module name and cannot go directly under Arcs",
			name,
		)
	}
	var siblings []ontology.Resource
	if err := s.cfg.Ontology.NewRetrieve().
		WhereIDs(parent).
		TraverseTo(ontology.ChildrenTraverser).
		Entries(&siblings).
		Exec(ctx, tx); err != nil {
		return err
	}
	for _, sibling := range siblings {
		if sibling.ID != id && sibling.Name == name {
			return errors.Wrapf(
				validate.ErrValidation,
				"name %q is already used in this directory",
				name,
			)
		}
	}
	return nil
}

func (s *Service) isModule(
	ctx context.Context,
	tx gorp.Tx,
	id ontology.ID,
) (bool, error) {
	key, err := uuid.Parse(id.Key)
	if err != nil {
		return false, err
	}
	var a Arc
	if err = s.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, Arc](key)).
		Entry(&a).
		Exec(ctx, tx); err != nil {
		return false, err
	}
	return a.Type == TypeModule, nil
}

// isDirectory reports whether id is the Arcs group or a group below it.
func (s *Service) isDirectory(
	ctx context.Context,
	tx gorp.Tx,
	id ontology.ID,
) (bool, error) {
	root := s.group.OntologyID()
	visited := set.New(id)
	frontier := []ontology.ID{id}
	for len(frontier) > 0 {
		current := frontier[0]
		frontier = frontier[1:]
		if current == root {
			return true, nil
		}
		if current.Type != ontology.ResourceTypeGroup {
			continue
		}
		parents, err := s.parents(ctx, tx, current)
		if err != nil {
			return false, err
		}
		for _, parent := range parents {
			if !visited.Contains(parent) {
				visited.Add(parent)
				frontier = append(frontier, parent)
			}
		}
	}
	return false, nil
}

func (s *Service) parents(
	ctx context.Context,
	tx gorp.Tx,
	id ontology.ID,
) ([]ontology.ID, error) {
	var parents []ontology.Resource
	if err := s.cfg.Ontology.NewRetrieve().
		WhereIDs(id).
		TraverseTo(ontology.ParentsTraverser).
		ExcludeFieldData(true).
		Entries(&parents).
		Exec(ctx, tx); err != nil {
		return nil, err
	}
	return lo.Map(parents, func(r ontology.Resource, _ int) ontology.ID {
		return r.ID
	}), nil
}

// reservedNames returns the names of the modules every Arc can import without a user
// module.
func reservedNames() set.Set[string] {
	names := set.New[string]()
	for _, sym := range standardSymbols() {
		if sym.Kind == symbol.KindModule && !sym.Internal {
			names.Add(sym.Name)
		}
	}
	return names
}
