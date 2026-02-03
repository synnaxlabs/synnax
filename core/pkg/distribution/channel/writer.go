// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/x/gorp"
)

type Writer struct {
	svc *Service
	tx  gorp.Tx
}

func (w Writer) Create(ctx context.Context, c *Channel, opts ...CreateOption) error {
	channels := []Channel{*c}
	if err := w.CreateMany(ctx, &channels, opts...); err != nil {
		return err
	}
	*c = channels[0]
	return nil
}

type CreateOptions struct {
	RetrieveIfNameExists                        bool
	OverwriteIfNameExistsAndDifferentProperties bool
	CreateWithoutGroupRelationship              bool
}

type CreateOption func(*CreateOptions)

func RetrieveIfNameExists() CreateOption {
	return func(o *CreateOptions) { o.RetrieveIfNameExists = true }
}

func OverwriteIfNameExistsAndDifferentProperties() CreateOption {
	return func(o *CreateOptions) {
		o.OverwriteIfNameExistsAndDifferentProperties = true
	}
}

func CreateWithoutGroupRelationship() CreateOption {
	return func(o *CreateOptions) {
		o.CreateWithoutGroupRelationship = true
	}
}

func (w Writer) CreateMany(ctx context.Context, channels *[]Channel, opts ...CreateOption) error {
	var o CreateOptions
	for _, opt := range opts {
		opt(&o)
	}
	return w.svc.proxy.create(ctx, w.tx, channels, o)
}

func (w Writer) Delete(ctx context.Context, key Key, allowInternal bool) error {
	return w.DeleteMany(ctx, []Key{key}, allowInternal)
}

func (w Writer) DeleteMany(ctx context.Context, keys []Key, allowInternal bool) error {
	return w.svc.proxy.delete(ctx, w.tx, keys, allowInternal)
}

func (w Writer) DeleteByName(ctx context.Context, name string, allowInternal bool) error {
	return w.DeleteManyByNames(ctx, []string{name}, allowInternal)
}

func (w Writer) DeleteManyByNames(ctx context.Context, names []string, allowInternal bool) error {
	return w.svc.proxy.deleteByName(ctx, w.tx, names, allowInternal)
}

func (w Writer) MapRename(ctx context.Context, names map[string]string, allowInternal bool) error {
	oldNames := lo.Keys(names)
	oldChannels := make([]Channel, 0, len(oldNames))
	if err := w.svc.NewRetrieve().WhereNames(oldNames...).Entries(&oldChannels).Exec(ctx, w.tx); err != nil {
		return err
	}
	newNames := make([]string, 0, len(oldChannels))
	for _, oldChannel := range oldChannels {
		newName := names[oldChannel.Name]
		newNames = append(newNames, newName)
	}
	return w.RenameMany(ctx, KeysFromChannels(oldChannels), newNames, allowInternal)
}

func (w Writer) Rename(
	ctx context.Context,
	key Key,
	newName string,
	allowInternal bool,
) error {
	return w.RenameMany(ctx, []Key{key}, []string{newName}, allowInternal)
}

func (w Writer) RenameMany(
	ctx context.Context,
	keys []Key,
	newNames []string,
	allowInternal bool,
) error {
	return w.svc.proxy.rename(ctx, w.tx, keys, newNames, allowInternal)
}
