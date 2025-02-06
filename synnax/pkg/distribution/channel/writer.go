// Copyright 2025 Synnax Labs, Inc.
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
	"strings"

	"github.com/synnaxlabs/x/gorp"
)

type Writer interface {
	Create(ctx context.Context, c *Channel, opts ...CreateOption) error
	CreateMany(ctx context.Context, channels *[]Channel, opts ...CreateOption) error
	Delete(ctx context.Context, key Key, allowInternal bool) error
	DeleteMany(ctx context.Context, keys []Key, allowInternal bool) error
	DeleteByName(ctx context.Context, name string, allowInternal bool) error
	DeleteManyByNames(ctx context.Context, names []string, allowInternal bool) error
	Rename(ctx context.Context, key Key, newName string, allowInternal bool) error
	RenameMany(ctx context.Context, keys []Key, newNames []string, allowInternal bool) error
}

type writer struct {
	proxy *leaseProxy
	tx    gorp.Tx
}

var _ Writer = writer{}

func (w writer) Create(ctx context.Context, c *Channel, opts ...CreateOption) error {
	channels := []Channel{*c}
	err := w.CreateMany(ctx, &channels, opts...)
	*c = channels[0]
	return err
}

type createOptions struct {
	retrieveIfNameExists bool
}

type CreateOption func(*createOptions)

func RetrieveIfNameExists(v bool) CreateOption {
	return func(o *createOptions) {
		o.retrieveIfNameExists = v
	}
}

func (w writer) CreateMany(ctx context.Context, channels *[]Channel, opts ...CreateOption) error {
	var o createOptions
	for _, opt := range opts {
		opt(&o)
	}
	return w.proxy.create(ctx, w.tx, applyManyAdjustments(channels), o.retrieveIfNameExists)
}

func (w writer) Delete(ctx context.Context, key Key, allowInternal bool) error {
	return w.DeleteMany(ctx, []Key{key}, allowInternal)
}

func (w writer) DeleteMany(ctx context.Context, keys []Key, allowInternal bool) error {
	return w.proxy.delete(ctx, w.tx, keys, allowInternal)
}

func (w writer) DeleteByName(ctx context.Context, name string, allowInternal bool) error {
	return w.DeleteManyByNames(ctx, []string{name}, allowInternal)
}

func (w writer) DeleteManyByNames(ctx context.Context, names []string, allowInternal bool) error {
	return w.proxy.deleteByName(ctx, w.tx, names, allowInternal)
}

func (w writer) Rename(
	ctx context.Context,
	key Key,
	newName string,
	allowInternal bool,
) error {
	return w.RenameMany(ctx, []Key{key}, []string{newName}, allowInternal)
}

func (w writer) RenameMany(
	ctx context.Context,
	keys []Key,
	newNames []string,
	allowInternal bool,
) error {
	return w.proxy.rename(ctx, w.tx, keys, newNames, allowInternal)
}

func applyAdjustments(c Channel) Channel {
	c.Name = strings.TrimSpace(c.Name)
	return c
}

func applyManyAdjustments(channels *[]Channel) *[]Channel {
	for i := range *channels {
		(*channels)[i] = applyAdjustments((*channels)[i])
	}
	return channels
}
