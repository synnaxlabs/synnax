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
	"fmt"
	"regexp"
	"strconv"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/types"
	"github.com/synnaxlabs/x/validate"
)

const calculatedIndexNameSuffix = "_time"

// Writer creates, deletes, and renames channels within a single transaction. It infers
// DataTypes for calculated channels before persisting, orchestrating key/storage
// allocation through the distribution-layer allocator and writing channel metadata to
// the service table. Obtain one from Service.NewWriter. A Writer is not safe for
// concurrent use.
type Writer struct {
	svc      *Service
	tx       gorp.Tx
	otg      ontology.Writer
	analyzer *CalculationAnalyzer
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

// CreateMany creates multiple channels, inferring and assigning DataTypes for any
// calculated channels in the batch by analyzing their expressions. Channels within the
// batch may reference each other by name. A calculated channel whose expression fails
// to analyze (invalid syntax or unresolved dependencies) aborts the entire call with
// the analysis error, so callers get fail-fast validation.
func (w Writer) CreateMany(
	ctx context.Context, channels *[]Channel, opts ...CreateOption,
) error {
	if len(*channels) == 0 {
		return nil
	}
	var o createOptions
	for _, opt := range opts {
		opt(&o)
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
	return w.create(ctx, channels, o)
}

// ChangeDataType persists dataType to the already-existing channel with the given key,
// without analyzing its expression. It is the calculation graph's write path for the
// calculated-channel DataTypes it derives via a cross-channel fixpoint: the graph is
// their source of truth, so re-deriving them here — as CreateMany would — is both
// redundant and, for interdependent channels analyzed in one pass against pre-update
// storage, incorrect. It returns validate.Error if the target channel is not
// calculated, since a non-calculated channel's DataType is fixed by its persistent
// storage.
func (w Writer) ChangeDataType(
	ctx context.Context, key Key, dataType telem.DataType,
) error {
	return w.svc.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Channel](key)).
		ChangeErr(func(_ gorp.Context, c Channel) (Channel, error) {
			if !c.IsCalculated() {
				return Channel{}, errors.Wrapf(
					validate.ErrValidation,
					"cannot change the data type of non-calculated channel %q",
					c.Name,
				)
			}
			c.DataType = dataType
			return c, nil
		}).
		Exec(ctx, w.tx)
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
	return w.delete(ctx, keys, allowInternal)
}

// DeleteManyByNames deletes the channels matching the given names. Unless allowInternal
// is true, deleting any internal channel returns an error and no channels are deleted.
func (w Writer) DeleteManyByNames(
	ctx context.Context, names []string, allowInternal bool,
) error {
	var res []Channel
	if err := w.svc.newRetrieve().
		Where(MatchNames(names...)).
		Entries(&res).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.delete(ctx, KeysFromChannels(res), allowInternal)
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
	return w.rename(ctx, keys, newNames, allowInternal)
}

// create orchestrates channel creation: it validates names, preprocesses calculated
// channels, resolves existing channels, allocates keys and storage through the
// distribution allocator, writes the rich channel records to the table, and registers
// ontology resources.
func (w Writer) create(ctx context.Context, _channels *[]Channel, opts createOptions) error {
	channels := *_channels
	for i := range channels {
		if err := channels[i].Validate(); err != nil {
			return validate.PathedError(err, strconv.Itoa(i))
		}
	}
	if *w.svc.cfg.ValidateNames {
		skipExisting := opts.retrieveIfNameExists || opts.overwriteIfNameExistsAndDifferentProperties
		if err := w.validateChannelNames(ctx, KeysFromChannels(channels), Names(channels), skipExisting); err != nil {
			return err
		}
	}
	for i, ch := range channels {
		if ch.Leaseholder == 0 {
			channels[i].Leaseholder = w.svc.cfg.HostResolver.HostKey()
		}
		if ch.IsCalculated() {
			if ch.LocalIndex != 0 && ch.LocalKey == 0 {
				return validate.PathedError(
					errors.Wrap(validate.ErrValidation, "calculated channels cannot specify an index manually"),
					"local_index",
				)
			}
			channels[i].Leaseholder = node.KeyFree
			channels[i].Virtual = true
		} else if ch.LocalKey != 0 {
			channels[i].LocalKey = 0
		}
	}

	// Update the stored records of channels passed in with a non-zero key, applying
	// name and (for calculated channels) expression/operations/index/data-type changes.
	// When RetrieveIfNameExists is set the in-memory channel is reset to the stored
	// record so the caller does not mistake an update for a no-op create.
	keys := KeysFromChannels(channels)
	existingKeys := lo.Filter(keys, func(k Key, _ int) bool {
		return k.LocalKey() != 0
	})
	if len(existingKeys) != 0 {
		if err := w.svc.table.NewUpdate().
			Where(gorp.MatchKeys[Key, Channel](existingKeys...)).
			ChangeErr(func(_ gorp.Context, c Channel) (Channel, error) {
				idx := lo.IndexOf(keys, c.Key())
				ic := channels[idx]
				if opts.retrieveIfNameExists {
					channels[idx] = c
					return c, nil
				}
				c.Name = ic.Name
				if c.IsCalculated() && ic.IsCalculated() {
					c.Expression = ic.Expression
					c.Operations = ic.Operations
					c.LocalIndex = ic.LocalIndex
					c.DataType = ic.DataType
				}
				return c, nil
			}).
			Exec(ctx, w.tx); err != nil && !errors.Is(err, query.ErrNotFound) {
			return err
		}
	}

	// Auto-create index channels for calculated channels that do not yet have one (both
	// brand-new calculated channels and existing ones missing an index). Synthesis
	// happens before the overwrite and retrieve-existing steps so the auto-created
	// index names flow through the same name-collision machinery as caller-provided
	// channels.
	indexChannels := make([]Channel, 0, len(channels))
	calcNeedingIndex := make([]int, 0)
	batchPositionByName := make(map[string]int, len(channels))
	for i, ch := range channels {
		batchPositionByName[ch.Name] = i
	}
	for i, ch := range channels {
		if !ch.IsCalculated() || ch.LocalIndex != 0 {
			continue
		}
		indexName := ch.Name + calculatedIndexNameSuffix
		if bi, inBatch := batchPositionByName[indexName]; inBatch {
			// An in-batch channel can only serve as the index if it satisfies the same
			// predicate as an adopted existing index below. Adopting a host-leased or
			// persisted index would graft its LocalKey into the calculated channel's
			// free keyspace, producing an index key that resolves to the wrong channel.
			b := channels[bi]
			if !b.IsIndex || !b.Virtual || b.Leaseholder != node.KeyFree ||
				b.DataType != telem.TimeStampT {
				return errors.Wrapf(
					validate.ErrValidation,
					"channel %q in the same request cannot serve as the index for calculated channel %q",
					indexName,
					ch.Name,
				)
			}
			calcNeedingIndex = append(calcNeedingIndex, i)
			continue
		}
		indexChannels = append(indexChannels, Channel{
			Name:        indexName,
			DataType:    telem.TimeStampT,
			IsIndex:     true,
			Virtual:     true,
			Leaseholder: node.KeyFree,
			Internal:    ch.Internal,
		})
		calcNeedingIndex = append(calcNeedingIndex, i)
	}
	// Resolve auto-created index names against existing channels: a compatible index
	// (e.g. one left behind by a deleted calculated channel) is adopted instead of
	// creating a duplicate name, while an incompatible channel is a validation error —
	// unless the overwrite option is set, in which case the overwrite step below
	// deletes and replaces it.
	if len(indexChannels) != 0 {
		indexNames := Names(indexChannels)
		var existing []Channel
		if err := w.svc.newRetrieve().
			Where(MatchNames(indexNames...)).
			Entries(&existing).
			Exec(ctx, w.tx); err != nil && !errors.Is(err, query.ErrNotFound) {
			return err
		}
		for _, ex := range existing {
			idx := lo.IndexOf(indexNames, ex.Name)
			if idx < 0 {
				continue
			}
			if ex.IsIndex && ex.Virtual && ex.Leaseholder == node.KeyFree &&
				ex.DataType == telem.TimeStampT {
				indexChannels[idx] = ex
				continue
			}
			if opts.overwriteIfNameExistsAndDifferentProperties {
				continue
			}
			return errors.Wrapf(
				validate.ErrValidation,
				"channel %q already exists and cannot serve as the index for a calculated channel",
				ex.Name,
			)
		}
	}
	channels = append(channels, indexChannels...)

	if opts.overwriteIfNameExistsAndDifferentProperties {
		// Delete existing channels of the same name whose properties differ from the
		// ones being created, so the create can replace them. Channels that match an
		// existing record (ignoring allocated keys) are reset to that record instead.
		names := Names(channels)
		if len(names) != 0 {
			var existing []Channel
			if err := w.svc.newRetrieve().
				Where(MatchNames(names...)).
				Entries(&existing).
				Exec(ctx, w.tx); err != nil {
				return errors.Skip(err, query.ErrNotFound)
			}
			keysToDelete := make(Keys, 0, len(existing))
			for _, ex := range existing {
				ch, i, found := lo.FindIndexOf(channels, func(ch Channel) bool {
					return ch.Name == ex.Name && ch.Key() != ex.Key()
				})
				if !found {
					continue
				}
				if ch.Equals(ex, "LocalKey", "LocalIndex", "Leaseholder") {
					channels[i] = ex
					continue
				}
				keysToDelete = append(keysToDelete, ex.Key())
			}
			if len(keysToDelete) != 0 {
				if err := w.delete(ctx, keysToDelete, true); err != nil {
					return err
				}
			}
		}
	}

	for _, ch := range channels {
		if len(ch.Name) == 0 {
			return validate.PathedError(validate.ErrRequired, "name")
		}
	}

	// Resolve channels that already exist by name (when RetrieveIfNameExists is set,
	// reusing their stored records), then allocate local keys and storage for the
	// remaining new channels through the distribution allocator.
	if opts.retrieveIfNameExists {
		names := Names(channels)
		var existing []Channel
		if err := w.svc.newRetrieve().
			Where(MatchNames(names...)).
			Entries(&existing).
			Exec(ctx, w.tx); err != nil {
			return errors.Skip(err, query.ErrNotFound)
		}
		for _, e := range existing {
			idx := lo.IndexOf(names, e.Name)
			if idx < 0 {
				continue
			}
			channels[idx] = e
		}
	}
	newIndices := make([]int, 0, len(channels))
	minimal := make([]channel.Channel, 0, len(channels))
	for i, ch := range channels {
		if ch.LocalKey == 0 {
			newIndices = append(newIndices, i)
			minimal = append(minimal, ch.Distribution())
		}
	}
	if err := w.allocateAndWrite(ctx, channels, newIndices, minimal); err != nil {
		return err
	}

	// Persist updated LocalIndex links for existing calculated channels (those already
	// in the table) whose index channel was just created.
	for _, idx := range calcNeedingIndex {
		ch := channels[idx]
		if ch.LocalKey == 0 || ch.LocalIndex == 0 {
			continue
		}
		exists, err := w.svc.newRetrieve().Where(MatchKeys(ch.Key())).Exists(ctx, w.tx)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}
		if err := w.svc.table.NewUpdate().
			Where(gorp.MatchKeys[Key, Channel](ch.Key())).
			Change(func(_ gorp.Context, c Channel) Channel {
				c.LocalIndex = ch.LocalIndex
				return c
			}).
			Exec(ctx, w.tx); err != nil {
			return err
		}
	}

	*_channels = channels

	// Register ontology resources and the group parent relationship for the created
	// channels.
	externalIDs := lo.FilterMap(channels, func(ch Channel, _ int) (ontology.ID, bool) {
		return OntologyID(ch.Key()), !ch.Internal
	})
	if err := w.otg.DefineResources(ctx, externalIDs...); err != nil {
		return err
	}
	if opts.createWithoutGroupRelationship {
		return nil
	}
	return w.otg.DefineRelationships(
		ctx,
		group.OntologyID(w.svc.group.Key),
		ontology.RelationshipTypeParentOf,
		externalIDs...,
	)
}

// allocateAndWrite enforces the external-channel overflow cap, allocates local keys and
// storage for the new channels (those at newIndices, mirrored in minimal) through the
// distribution allocator, links calculated channels to their index channels, writes the
// newly created channels to the table, and records their keys in the external
// non-virtual set. The service lock is held for the whole sequence so the cap check,
// allocation, table write, and set update are atomic: a create rejected by the cap
// allocates no keys or storage, and concurrent creates cannot interleave past the cap.
func (w Writer) allocateAndWrite(
	ctx context.Context,
	channels []Channel,
	newIndices []int,
	minimal []channel.Channel,
) error {
	externalNewCount := 0
	for _, i := range newIndices {
		if !channels[i].Internal && !channels[i].Virtual {
			externalNewCount++
		}
	}
	w.svc.mu.Lock()
	defer w.svc.mu.Unlock()
	count := w.svc.mu.externalNonVirtualSet.Size()
	if err := w.svc.cfg.IntOverflowCheck(types.Uint20(int(count) + externalNewCount)); err != nil {
		return err
	}
	toCreate := make([]Channel, 0, len(newIndices))
	if len(minimal) != 0 {
		allocated, err := w.svc.cfg.Channel.Create(ctx, minimal)
		if err != nil {
			return err
		}
		for j, i := range newIndices {
			channels[i].LocalKey = allocated[j].LocalKey
			channels[i].LocalIndex = allocated[j].LocalIndex
			toCreate = append(toCreate, channels[i])
		}
	}

	// Link each calculated channel to its index channel by setting the calculated
	// channel's LocalIndex to the index channel's assigned LocalKey. Applied to both
	// the newly created channels and the full set (which includes existing ones).
	indexKeyByName := make(map[string]LocalKey, len(channels))
	for _, ch := range channels {
		if ch.IsIndex {
			indexKeyByName[ch.Name] = ch.LocalKey
		}
	}
	linkCalculatedIndexes := func(chs []Channel) {
		for i := range chs {
			if !chs[i].IsCalculated() || chs[i].LocalIndex != 0 {
				continue
			}
			if k, ok := indexKeyByName[chs[i].Name+calculatedIndexNameSuffix]; ok {
				chs[i].LocalIndex = k
			}
		}
	}
	linkCalculatedIndexes(toCreate)
	linkCalculatedIndexes(channels)

	externalCreatedKeys := make(Keys, 0, len(toCreate))
	for _, ch := range toCreate {
		if !ch.Internal && !ch.Virtual {
			externalCreatedKeys = append(externalCreatedKeys, ch.Key())
		}
	}
	if err := w.svc.table.NewCreate().Entries(&toCreate).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.svc.mu.externalNonVirtualSet.Insert(externalCreatedKeys...)
	return nil
}

// validNamePattern matches valid channel names: a leading letter or underscore followed
// by letters, digits, and underscores. Because every stored channel name is accepted by
// this pattern, MatchNames also uses it to decide whether an input is a literal
// exact-match target (routable through the name index) or a regex pattern.
var validNamePattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// validateChannelNames rejects a create/rename request whose proposed names are empty,
// contain characters outside validNamePattern, duplicate each other, or collide with an
// existing channel under a different key.
func (w Writer) validateChannelNames(
	ctx context.Context,
	keys Keys,
	names []string,
	skipExisting bool,
) error {
	for i, name := range names {
		if name == "" {
			return validate.PathedError(
				errors.Wrap(validate.ErrValidation, "name cannot be empty"),
				fmt.Sprintf("[%d]", i), "name",
			)
		}
		if !validNamePattern.MatchString(name) {
			return validate.PathedError(
				errors.Wrapf(
					validate.ErrValidation,
					"channel name '%s' contains invalid characters. Only letters, digits, and underscores are allowed, and it cannot start with a digit",
					name,
				),
				fmt.Sprintf("[%d]", i), "name",
			)
		}
	}
	namesSeen := make(set.Set[string], len(names))
	for i, name := range names {
		if namesSeen.Contains(name) {
			return validate.PathedError(
				errors.Wrapf(
					validate.ErrValidation,
					"duplicate channel name '%s' in request",
					name,
				),
				fmt.Sprintf("[%d].name", i),
			)
		}
		namesSeen.Add(name)
	}
	if skipExisting {
		return nil
	}
	var conflictingChannels []Channel
	if err := w.svc.newRetrieve().
		Where(MatchNames(names...)).
		Entries(&conflictingChannels).
		Exec(ctx, w.tx); err != nil {
		return errors.Skip(err, query.ErrNotFound)
	}
	nameConflicts := make(map[string]int, len(conflictingChannels))
	for i, ch := range conflictingChannels {
		nameConflicts[ch.Name] = i
	}
	for i, name := range names {
		conflictingIdx, conflict := nameConflicts[name]
		if !conflict {
			continue
		}
		existingCh := conflictingChannels[conflictingIdx]
		if existingCh.Key() == keys[i] {
			continue
		}
		return validate.PathedError(
			errors.Wrapf(
				validate.ErrValidation, "channel with name '%s' already exists", name,
			),
			fmt.Sprintf("[%d].name", i),
		)
	}
	return nil
}

func (w Writer) delete(ctx context.Context, keys Keys, allowInternal bool) error {
	if err := w.svc.table.NewDelete().
		Where(gorp.MatchKeys[Key, Channel](keys...)).
		Guard(func(_ gorp.Context, c Channel) error {
			if c.Internal && !allowInternal {
				return errors.Wrapf(
					validate.ErrValidation,
					"can't delete internal channel %q",
					c.Name,
				)
			}
			return nil
		}).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	if err := w.otg.DeleteResources(ctx, OntologyIDsFromKeys(keys)...); err != nil {
		return err
	}
	// Storage deletion goes last, as it is the only operation that can fail without an
	// atomic guarantee.
	if err := w.svc.cfg.Channel.Delete(ctx, keys); err != nil {
		return err
	}
	w.svc.mu.Lock()
	w.svc.mu.externalNonVirtualSet.Remove(keys...)
	w.svc.mu.Unlock()
	return nil
}

func (w Writer) rename(
	ctx context.Context,
	keys Keys,
	names []string,
	allowInternal bool,
) error {
	if len(keys) != len(names) {
		return errors.Wrap(
			validate.ErrValidation,
			"keys and names must be the same length",
		)
	}
	if *w.svc.cfg.ValidateNames {
		if err := w.validateChannelNames(ctx, keys, names, false); err != nil {
			return err
		}
	}
	renameMap := make(map[Key]string, len(keys))
	for i, key := range keys {
		renameMap[key] = names[i]
	}
	if err := w.svc.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Channel](keys...)).
		ChangeErr(func(_ gorp.Context, c Channel) (Channel, error) {
			if c.Internal && !allowInternal {
				return Channel{},
					errors.Wrapf(
						validate.ErrValidation,
						"cannot rename internal channel %v",
						c,
					)
			}
			c.Name = renameMap[c.Key()]
			return c, nil
		}).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.svc.cfg.Channel.Rename(ctx, renameMap)
}
