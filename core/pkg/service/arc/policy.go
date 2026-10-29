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
// an Arc with no other parent, a group name is an identifier, a name is unique in its
// directory, and a standard module name is reserved directly under Arcs. A group is
// checked with every entry below it. Outside Arcs an Arc, a group under Arcs, or a
// group holding an Arc is rejected.
func (s *Service) ValidateParent(
	ctx context.Context,
	tx gorp.Tx,
	parent ontology.ID,
	children ...ontology.ID,
) error {
	directory, err := s.isDirectory(ctx, tx, parent)
	if err != nil {
		return err
	}
	if !directory {
		for _, child := range children {
			var inside bool
			if inside, err = s.isDirectory(ctx, tx, child); err != nil {
				return err
			}
			if inside {
				return errors.Wrap(
					validate.ErrValidation,
					"a group under Arcs cannot have a parent outside Arcs",
				)
			}
			if err = s.validateOutside(ctx, tx, child); err != nil {
				return err
			}
		}
		return nil
	}
	var entries []ontology.Resource
	if err = s.cfg.Ontology.NewRetrieve().
		WhereIDs(children...).
		Entries(&entries).
		Exec(ctx, tx); err != nil {
		return err
	}
	if dups := lo.FindDuplicatesBy(entries, func(r ontology.Resource) string {
		return r.Name
	}); len(dups) > 0 {
		return errNameUsed(dups[0].Name)
	}
	for _, entry := range entries {
		if err = s.validateEntry(ctx, tx, parent, entry.ID, entry.Name); err != nil {
			return err
		}
		if err = s.validateSoleParent(ctx, tx, parent, entry); err != nil {
			return err
		}
		if entry.ID.Type == ontology.ResourceTypeGroup {
			if err = s.validateSubtree(ctx, tx, entry.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

// ValidateDetach implements ontology.Policy. An Arc, or a group holding one, keeps a
// directory parent.
func (s *Service) ValidateDetach(
	ctx context.Context,
	tx gorp.Tx,
	child ontology.ID,
	parents ...ontology.ID,
) error {
	for _, parent := range parents {
		directory, err := s.isDirectory(ctx, tx, parent)
		if err != nil {
			return err
		}
		if directory {
			return nil
		}
	}
	return s.validateOutside(ctx, tx, child)
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
	if id.Type != ontology.ResourceTypeGroup && id.Type != ontology.ResourceTypeArc {
		return errors.Wrapf(
			validate.ErrValidation,
			"a %s cannot go under Arcs",
			id.Type,
		)
	}
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
			return errNameUsed(name)
		}
	}
	return nil
}

func errNameUsed(name string) error {
	return errors.Wrapf(
		validate.ErrValidation, "name %q is already used in this directory", name,
	)
}

// validateSoleParent rejects entry when it has a parent other than parent.
func (s *Service) validateSoleParent(
	ctx context.Context,
	tx gorp.Tx,
	parent ontology.ID,
	entry ontology.Resource,
) error {
	parents, err := s.parents(ctx, tx, entry.ID)
	if err != nil {
		return err
	}
	if lo.SomeBy(parents, func(p ontology.ID) bool { return p != parent }) {
		return errors.Wrapf(
			validate.ErrValidation, "%q already has another parent", entry.Name,
		)
	}
	return nil
}

// validateSubtree checks every entry below the directory g.
func (s *Service) validateSubtree(
	ctx context.Context,
	tx gorp.Tx,
	g ontology.ID,
) error {
	return s.walk(
		ctx, tx, g, false,
		func(parent ontology.ID, child ontology.Resource) error {
			err := s.validateEntry(ctx, tx, parent, child.ID, child.Name)
			if err != nil {
				return err
			}
			return s.validateSoleParent(ctx, tx, parent, child)
		},
	)
}

// validateOutside rejects id placed outside Arcs when it is an Arc or a group holding
// one.
func (s *Service) validateOutside(
	ctx context.Context,
	tx gorp.Tx,
	id ontology.ID,
) error {
	switch id.Type {
	case ontology.ResourceTypeArc:
		return errors.Wrap(validate.ErrValidation, "an Arc can only go under Arcs")
	case ontology.ResourceTypeGroup:
	default:
		return nil
	}
	return s.walk(
		ctx, tx, id, true,
		func(_ ontology.ID, child ontology.Resource) error {
			if child.ID.Type == ontology.ResourceTypeArc {
				return errors.Wrap(
					validate.ErrValidation,
					"a group holding an Arc can only go under Arcs",
				)
			}
			return nil
		},
	)
}

// walk calls visit on every entry below g, descending through groups. Field data is
// skipped when fieldDataExcluded is true, for a visit that reads only types.
func (s *Service) walk(
	ctx context.Context,
	tx gorp.Tx,
	g ontology.ID,
	fieldDataExcluded bool,
	visit func(parent ontology.ID, child ontology.Resource) error,
) error {
	visited := set.New(g)
	frontier := []ontology.ID{g}
	for len(frontier) > 0 {
		current := frontier[0]
		frontier = frontier[1:]
		var children []ontology.Resource
		if err := s.cfg.Ontology.NewRetrieve().
			WhereIDs(current).
			TraverseTo(ontology.ChildrenTraverser).
			ExcludeFieldData(fieldDataExcluded).
			Entries(&children).
			Exec(ctx, tx); err != nil {
			return err
		}
		for _, child := range children {
			if err := visit(current, child); err != nil {
				return err
			}
			if child.ID.Type == ontology.ResourceTypeGroup &&
				!visited.Contains(child.ID) {
				visited.Add(child.ID)
				frontier = append(frontier, child.ID)
			}
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
