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
	"slices"

	"github.com/synnaxlabs/synnax/pkg/service/arc/directory"
	"github.com/synnaxlabs/x/validate"
)

// CreateDirectory creates d in the directory parent, giving d a key when it has none.
func (w Writer) CreateDirectory(
	ctx context.Context,
	d *directory.Directory,
	parent directory.Key,
) error {
	d.Parent = &parent
	if err := w.validateEntry(
		ctx, parent, directory.ResourceID(d.Key), d.Name, true,
	); err != nil {
		return err
	}
	if err := w.directories.NewWriter(w.tx).Create(ctx, d); err != nil {
		return err
	}
	w.directoryDispatcher.Attach(
		w.tx, d.Key, directory.NewCreateAction(directory.CreatePayload{Directory: *d}),
	)
	return nil
}

// RenameDirectory gives the directory with key the given name.
func (w Writer) RenameDirectory(
	ctx context.Context,
	key directory.Key,
	name string,
) error {
	return w.ApplyDirectory(ctx, key, directory.NewSetAction(
		directory.SetPayload{Path: []string{"name"}, Value: name},
	))
}

// ApplyDirectory applies acts to the directory with key and attaches them to the
// transaction of w. A change to the root, and a name that the directory's parent cannot
// hold, are validation errors.
func (w Writer) ApplyDirectory(
	ctx context.Context,
	key directory.Key,
	acts ...directory.Action,
) error {
	if key == w.root {
		return errRoot("changed")
	}
	if err := w.directories.NewWriter(w.tx).Update(ctx, key, func(
		d directory.Directory,
	) (directory.Directory, error) {
		next, err := directory.Reduce(d, acts...)
		if err != nil {
			return directory.Directory{}, err
		}
		if next.Parent == nil {
			return directory.Directory{}, validate.PathedError(
				validate.ErrRequired, "parent",
			)
		}
		if err = w.validateEntry(
			ctx, *next.Parent, directory.ResourceID(key), next.Name, true,
		); err != nil {
			return directory.Directory{}, err
		}
		return next, nil
	}); err != nil {
		return err
	}
	w.directoryDispatcher.Attach(w.tx, key, acts...)
	w.pathChanges.Add(w.tx, struct{}{})
	return nil
}

// DeleteDirectory deletes the empty directories with keys.
func (w Writer) DeleteDirectory(ctx context.Context, keys ...directory.Key) error {
	if slices.Contains(keys, w.root) {
		return errRoot("deleted")
	}
	if err := w.directories.NewWriter(w.tx).Delete(ctx, keys...); err != nil {
		return err
	}
	w.pathChanges.Add(w.tx, struct{}{})
	return nil
}
