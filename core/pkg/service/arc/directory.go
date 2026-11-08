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
	return w.directories.NewWriter(w.tx).Create(ctx, d)
}

// RenameDirectory gives the directory with key the given name.
func (w Writer) RenameDirectory(
	ctx context.Context,
	key directory.Key,
	name string,
) error {
	if key == w.root {
		return errRoot("renamed")
	}
	if err := w.validateName(ctx, directory.ResourceID(key), name, true); err != nil {
		return err
	}
	return w.directories.NewWriter(w.tx).Rename(ctx, key, name)
}

// DeleteDirectory deletes the empty directories with keys.
func (w Writer) DeleteDirectory(ctx context.Context, keys ...directory.Key) error {
	if slices.Contains(keys, w.root) {
		return errRoot("deleted")
	}
	return w.directories.NewWriter(w.tx).Delete(ctx, keys...)
}
