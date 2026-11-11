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
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// Move moves the Arcs and directories with ids from directory from to directory to.
// An entry not in from is a validation error.
func (w Writer) Move(
	ctx context.Context,
	from, to directory.Key,
	ids ...resource.ID,
) error {
	var arcKeys, dirKeys []uuid.UUID
	for _, id := range ids {
		switch {
		case id == directory.ResourceID(w.root):
			return errRoot("moved")
		case id == directory.ResourceID(to):
			return errors.Wrap(
				validate.ErrValidation, "a directory cannot move into itself",
			)
		case id.Type != resource.TypeArc &&
			id.Type != resource.TypeArcDirectory:
			return errors.Wrapf(
				validate.ErrValidation, "a %s cannot go in an Arc directory", id.Type,
			)
		}
		key, err := resource.ParseUUIDKey(id)
		if err != nil {
			return err
		}
		if id.Type == resource.TypeArc {
			arcKeys = append(arcKeys, key)
		} else {
			dirKeys = append(dirKeys, key)
		}
	}
	entries, err := w.retrieveEntries(ctx, arcKeys, dirKeys)
	if err != nil {
		return err
	}
	if dups := lo.FindDuplicatesBy(entries, func(e entry) string {
		return e.name
	}); len(dups) > 0 {
		return errNameUsed(dups[0].name)
	}
	for _, e := range entries {
		if e.parent == nil || *e.parent != from {
			return errors.Wrapf(
				validate.ErrValidation, "%q is not in the source directory", e.name,
			)
		}
	}
	for _, key := range dirKeys {
		if err = w.ApplyDirectory(ctx, key, directory.NewSetAction(
			directory.SetPayload{Path: []string{"parent"}, Value: to},
		)); err != nil {
			return err
		}
	}
	set := NewSetDirectoryAction(SetDirectoryPayload{Directory: to})
	for _, key := range arcKeys {
		if err = w.Apply(ctx, key, set); err != nil {
			return err
		}
	}
	return nil
}

// entry is an Arc or a directory as the directory that holds it sees it.
type entry struct {
	id     resource.ID
	name   string
	parent *directory.Key
}

func (w Writer) retrieveEntries(
	ctx context.Context,
	arcKeys, dirKeys []uuid.UUID,
) ([]entry, error) {
	var (
		arcs []Arc
		dirs []directory.Directory
	)
	if len(arcKeys) > 0 {
		if err := w.table.NewRetrieve().
			Where(gorp.MatchKeys[Key, Arc](arcKeys...)).
			Entries(&arcs).
			Exec(ctx, w.tx); err != nil {
			return nil, err
		}
	}
	if len(dirKeys) > 0 {
		if err := w.directories.NewRetrieve().
			Where(directory.MatchKeys(dirKeys...)).
			Entries(&dirs).
			Exec(ctx, w.tx); err != nil {
			return nil, err
		}
	}
	entries := make([]entry, 0, len(arcs)+len(dirs))
	for _, a := range arcs {
		entries = append(entries, arcEntry(a))
	}
	for _, d := range dirs {
		entries = append(entries, directoryEntry(d))
	}
	return entries, nil
}

func arcEntry(a Arc) entry {
	b := a.Base()
	return entry{id: a.ResourceID(), name: b.Name, parent: b.Directory}
}

func directoryEntry(d directory.Directory) entry {
	return entry{id: d.ResourceID(), name: d.Name, parent: d.Parent}
}

// validateEntry checks name for the entry id placed in the directory parent. The name
// of an identifier entry must be an identifier.
func (w Writer) validateEntry(
	ctx context.Context,
	parent directory.Key,
	id resource.ID,
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
	if parent == w.root && w.reserved.Contains(name) {
		return errors.Wrapf(
			validate.ErrValidation,
			"%q is a standard module name and cannot go directly under Arcs",
			name,
		)
	}
	var (
		arcs []Arc
		dirs []directory.Directory
	)
	if err := w.table.NewRetrieve().
		Where(gorp.Match(func(_ gorp.Context, a *Arc) (bool, error) {
			return inDirectories(a, parent), nil
		})).
		Entries(&arcs).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	if err := w.directories.NewRetrieve().
		Where(directory.MatchParents(parent)).
		Entries(&dirs).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	for _, a := range arcs {
		if a.ResourceID() != id && a.Base().Name == name {
			return errNameUsed(name)
		}
	}
	for _, d := range dirs {
		if d.ResourceID() != id && d.Name == name {
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
