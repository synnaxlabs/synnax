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
	"github.com/synnaxlabs/synnax/pkg/service/arc/directory"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// Move moves the Arcs and directories with ids from directory from to directory to,
// which becomes the only parent of each. An entry not in from is a validation error.
func (w Writer) Move(
	ctx context.Context,
	from, to directory.Key,
	ids ...ontology.ID,
) error {
	source := directory.OntologyID(from)
	parent := directory.OntologyID(to)
	for _, id := range ids {
		switch {
		case id == directory.OntologyID(w.root):
			return errRoot("moved")
		case id == parent:
			return errors.Wrap(
				validate.ErrValidation, "a directory cannot move into itself",
			)
		case id.Type != ontology.ResourceTypeArc &&
			id.Type != ontology.ResourceTypeArcDirectory:
			return errors.Wrapf(
				validate.ErrValidation, "a %s cannot go in an Arc directory", id.Type,
			)
		}
	}
	var entries []ontology.Resource
	if err := w.otg.NewRetrieve().
		WhereIDs(ids...).
		Entries(&entries).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	if dups := lo.FindDuplicatesBy(entries, func(r ontology.Resource) string {
		return r.Name
	}); len(dups) > 0 {
		return errNameUsed(dups[0].Name)
	}
	for _, entry := range entries {
		current, err := w.parentOf(ctx, entry.ID)
		if err != nil {
			return err
		}
		if current != source {
			return errors.Wrapf(
				validate.ErrValidation, "%q is not in the source directory", entry.Name,
			)
		}
		identifier := true
		if entry.ID.Type == ontology.ResourceTypeArc {
			if identifier, err = w.isModule(ctx, entry.ID); err != nil {
				return err
			}
		}
		if err = w.validateEntry(
			ctx, parent, entry.ID, entry.Name, identifier,
		); err != nil {
			return err
		}
	}
	for _, id := range ids {
		if err := w.otgWriter.DeleteIncomingRelationshipsOfType(
			ctx, id, ontology.RelationshipTypeParentOf,
		); err != nil {
			return err
		}
	}
	return w.otgWriter.DefineRelationships(
		ctx, parent, ontology.RelationshipTypeParentOf, ids...,
	)
}

// validateName checks name for the entry id in the directory that holds it. An entry
// in no directory is not checked.
func (w Writer) validateName(
	ctx context.Context,
	id ontology.ID,
	name string,
	identifier bool,
) error {
	parent, err := w.parentOf(ctx, id)
	if err != nil || parent.IsZero() {
		return err
	}
	return w.validateEntry(ctx, parent, id, name, identifier)
}

// parentOf returns the directory that holds id, or a zero ID when no directory does.
func (w Writer) parentOf(ctx context.Context, id ontology.ID) (ontology.ID, error) {
	var parents []ontology.Resource
	if err := w.otg.NewRetrieve().
		WhereIDs(id).
		TraverseTo(ontology.ParentsTraverser).
		WhereTypes(ontology.ResourceTypeArcDirectory).
		ExcludeFieldData(true).
		Entries(&parents).
		Exec(ctx, w.tx); err != nil || len(parents) == 0 {
		return ontology.ID{}, err
	}
	return parents[0].ID, nil
}

// validateEntry checks name for the entry id placed in the directory parent. The name
// of an identifier entry must be an identifier.
func (w Writer) validateEntry(
	ctx context.Context,
	parent, id ontology.ID,
	name string,
	identifier bool,
) error {
	if identifier && !validate.IsIdentifier(name) {
		return errors.Wrapf(
			validate.ErrValidation,
			"name %q must contain only letters, digits, and underscores",
			name,
		)
	}
	if parent == directory.OntologyID(w.root) && w.reserved.Contains(name) {
		return errors.Wrapf(
			validate.ErrValidation,
			"%q is a standard module name and cannot go directly under Arcs",
			name,
		)
	}
	var siblings []ontology.Resource
	if err := w.otg.NewRetrieve().
		WhereIDs(parent).
		TraverseTo(ontology.ChildrenTraverser).
		Entries(&siblings).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	for _, sibling := range siblings {
		if sibling.ID != id && sibling.Name == name {
			return errNameUsed(name)
		}
	}
	return nil
}

func (w Writer) isModule(ctx context.Context, id ontology.ID) (bool, error) {
	key, err := uuid.Parse(id.Key)
	if err != nil {
		return false, err
	}
	var a Arc
	if err = w.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, Arc](key)).
		Entry(&a).
		Exec(ctx, w.tx); err != nil {
		return false, err
	}
	return a.Type() == TypeModule, nil
}

func errNameUsed(name string) error {
	return errors.Wrapf(
		validate.ErrValidation, "name %q is already used in this directory", name,
	)
}

func errRoot(verb string) error {
	return errors.Wrapf(
		validate.ErrValidation, "the root Arcs directory cannot be %s", verb,
	)
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
