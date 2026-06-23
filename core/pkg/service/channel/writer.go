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

	"github.com/synnaxlabs/x/gorp"
)

// Writer creates, deletes, and renames channels. It infers DataTypes for calculated
// channels before persisting, orchestrating key/storage allocation through the
// distribution-layer allocator and writing channel metadata to the service table.
type Writer struct {
	svc      *Service
	tx       gorp.Tx
	analyzer *Analyzer
}

// NewWriter returns a Writer scoped to the provided transaction (nil writes directly to
// the service DB).
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		svc:      s,
		tx:       s.db.OverrideTx(tx),
		analyzer: NewAnalyzer(s.NewArcSymbolResolver(tx)),
	}
}

type createOptions struct {
	retrieveIfNameExists                        bool
	overwriteIfNameExistsAndDifferentProperties bool
	createWithoutGroupRelationship              bool
}

// CreateOption configures the behavior of a Create or CreateMany call.
type CreateOption func(*createOptions)

// RetrieveIfNameExists returns a CreateOption that, when a channel with the same name
// already exists, populates the provided channel from the existing record instead of
// returning an error.
func RetrieveIfNameExists() CreateOption {
	return func(o *createOptions) { o.retrieveIfNameExists = true }
}

// OverwriteIfNameExistsAndDifferentProperties returns a CreateOption that overwrites an
// existing channel of the same name when its properties differ from the channel being
// created.
func OverwriteIfNameExistsAndDifferentProperties() CreateOption {
	return func(o *createOptions) {
		o.overwriteIfNameExistsAndDifferentProperties = true
	}
}

// CreateWithoutGroupRelationship returns a CreateOption that skips creating the
// ontology relationship between the channel and its group.
func CreateWithoutGroupRelationship() CreateOption {
	return func(o *createOptions) { o.createWithoutGroupRelationship = true }
}

// Create creates a single channel, inferring the DataType if it is calculated.
func (w Writer) Create(ctx context.Context, c *Channel, opts ...CreateOption) error {
	channels := []Channel{*c}
	if err := w.CreateMany(ctx, &channels, opts...); err != nil {
		return err
	}
	*c = channels[0]
	return nil
}

// CreateMany creates multiple channels, inferring DataTypes for any calculated channels
// in the batch. Channels within the batch may reference each other by name.
func (w Writer) CreateMany(
	ctx context.Context, channels *[]Channel, opts ...CreateOption,
) error {
	if len(*channels) == 0 {
		return nil
	}
	for i, ch := range *channels {
		if !ch.IsCalculated() {
			continue
		}
		result, err := w.analyzer.Analyze(ctx, ch)
		if err != nil {
			return err
		}
		(*channels)[i].DataType = result.ChanDataType
	}
	var o createOptions
	for _, opt := range opts {
		opt(&o)
	}
	return w.svc.create(ctx, w.tx, channels, o)
}

// Delete deletes the channel with the given key, along with its storage and ontology
// resources. Unless allowInternal is true, deleting an internal channel returns an
// error.
func (w Writer) Delete(ctx context.Context, key Key, allowInternal bool) error {
	return w.DeleteMany(ctx, []Key{key}, allowInternal)
}

// DeleteMany deletes the channels with the given keys, along with their storage and
// ontology resources. Unless allowInternal is true, deleting any internal channel
// returns an error and no channels are deleted.
func (w Writer) DeleteMany(ctx context.Context, keys []Key, allowInternal bool) error {
	return w.svc.delete(ctx, w.tx, keys, allowInternal)
}

// DeleteManyByNames deletes the channels matching the given names. Unless allowInternal is
// true, deleting any internal channel returns an error and no channels are deleted.
func (w Writer) DeleteManyByNames(
	ctx context.Context, names []string, allowInternal bool,
) error {
	return w.svc.deleteByName(ctx, w.tx, names, allowInternal)
}

// Rename renames the channel with the given key to newName. Unless allowInternal is
// true, renaming an internal channel returns an error.
func (w Writer) Rename(
	ctx context.Context, key Key, newName string, allowInternal bool,
) error {
	return w.RenameMany(ctx, []Key{key}, []string{newName}, allowInternal)
}

// RenameMany renames the channels with the given keys to the corresponding entries in
// newNames, which must be parallel to keys. Unless allowInternal is true, renaming any
// internal channel returns an error.
func (w Writer) RenameMany(
	ctx context.Context, keys []Key, newNames []string, allowInternal bool,
) error {
	return w.svc.rename(ctx, w.tx, keys, newNames, allowInternal)
}
