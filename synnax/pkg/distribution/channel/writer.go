// Copyright 2023 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/x/gorp"
	"strings"
)

type Writer interface {
	Create(ctx context.Context, c *Channel) error
	CreateIfNameDoesntExist(ctx context.Context, c *Channel) error
	CreateManyIfNamesDontExist(ctx context.Context, channels *[]Channel) error
	CreateMany(ctx context.Context, channels *[]Channel) error
	Delete(ctx context.Context, key Key) error
	DeleteMany(ctx context.Context, keys []Key) error
	DeleteByName(ctx context.Context, name string) error
	DeleteManyByNames(ctx context.Context, names []string) error
}

type writer struct {
	proxy *leaseProxy
	tx    gorp.Tx
}

var _ Writer = writer{}

func (w writer) Create(ctx context.Context, c *Channel) error {
	channels := []Channel{*c}
	err := w.CreateMany(ctx, &channels)
	*c = channels[0]
	return err
}

func (w writer) CreateIfNameDoesntExist(ctx context.Context, c *Channel) error {
	channels := []Channel{*c}
	err := w.CreateManyIfNamesDontExist(ctx, &channels)
	*c = channels[0]
	return err
}

func (w writer) CreateManyIfNamesDontExist(ctx context.Context, channels *[]Channel) error {
	return w.proxy.create(ctx, w.tx, applyManyAdjustments(channels), true)
}

func (w writer) CreateMany(ctx context.Context, channels *[]Channel) error {
	return w.proxy.create(ctx, w.tx, applyManyAdjustments(channels), false)
}

func (w writer) Delete(ctx context.Context, key Key) error {
	return w.DeleteMany(ctx, []Key{key})
}

func (w writer) DeleteMany(ctx context.Context, keys []Key) error {
	return w.proxy.delete(ctx, w.tx, keys)
}

func (w writer) DeleteByName(ctx context.Context, name string) error {
	return w.DeleteManyByNames(ctx, []string{name})
}

func (w writer) DeleteManyByNames(ctx context.Context, names []string) error {
	return w.proxy.deleteByName(ctx, w.tx, names)
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
