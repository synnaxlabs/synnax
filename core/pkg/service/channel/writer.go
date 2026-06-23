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
// the service table. Obtain one from Service.NewWriter.
type Writer struct {
	// svc is the owning channel service, providing the metadata table, distribution
	// allocator, ontology integration, and external-channel overflow set.
	svc *Service
	// tx scopes every write the Writer performs; nil writes directly to the service DB.
	tx gorp.Tx
	// analyzer infers DataTypes for calculated channels before they are persisted.
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
	return w.create(ctx, channels, o)
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

// DeleteManyByNames deletes the channels matching the given names. Unless allowInternal is
// true, deleting any internal channel returns an error and no channels are deleted.
func (w Writer) DeleteManyByNames(
	ctx context.Context, names []string, allowInternal bool,
) error {
	return w.deleteByName(ctx, names, allowInternal)
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

// The methods below are convenience shortcuts that run a single operation through a
// fresh, transaction-less Writer. Callers that need to batch writes into a transaction
// should use NewWriter directly.

// Create creates a single channel outside of any transaction. See Writer.Create.
func (s *Service) Create(ctx context.Context, c *Channel, opts ...CreateOption) error {
	return s.NewWriter(nil).Create(ctx, c, opts...)
}

// CreateMany creates multiple channels outside of any transaction. See Writer.CreateMany.
func (s *Service) CreateMany(
	ctx context.Context, channels *[]Channel, opts ...CreateOption,
) error {
	return s.NewWriter(nil).CreateMany(ctx, channels, opts...)
}

// Delete deletes a channel outside of any transaction. See Writer.Delete.
func (s *Service) Delete(ctx context.Context, key Key, allowInternal bool) error {
	return s.NewWriter(nil).Delete(ctx, key, allowInternal)
}

// DeleteMany deletes channels outside of any transaction. See Writer.DeleteMany.
func (s *Service) DeleteMany(ctx context.Context, keys []Key, allowInternal bool) error {
	return s.NewWriter(nil).DeleteMany(ctx, keys, allowInternal)
}

// DeleteManyByNames deletes channels by name outside of any transaction. See
// Writer.DeleteManyByNames.
func (s *Service) DeleteManyByNames(
	ctx context.Context, names []string, allowInternal bool,
) error {
	return s.NewWriter(nil).DeleteManyByNames(ctx, names, allowInternal)
}

// Rename renames a channel outside of any transaction. See Writer.Rename.
func (s *Service) Rename(
	ctx context.Context, key Key, newName string, allowInternal bool,
) error {
	return s.NewWriter(nil).Rename(ctx, key, newName, allowInternal)
}

// RenameMany renames channels outside of any transaction. See Writer.RenameMany.
func (s *Service) RenameMany(
	ctx context.Context, keys []Key, newNames []string, allowInternal bool,
) error {
	return s.NewWriter(nil).RenameMany(ctx, keys, newNames, allowInternal)
}

// create orchestrates channel creation: it validates names, preprocesses calculated
// channels, resolves existing channels, allocates keys and storage through the
// distribution allocator, writes the rich channel records to the table, and registers
// ontology resources.
func (w Writer) create(ctx context.Context, _channels *[]Channel, opts createOptions) error {
	channels := *_channels
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

	// Update existing channels passed in with a key (e.g. a calculated channel whose
	// expression is being changed). Reset to the stored record when RetrieveIfNameExists
	// so the caller does not mistake an update for a no-op create.
	if err := w.updateExisting(ctx, &channels, opts); err != nil {
		return err
	}

	if opts.overwriteIfNameExistsAndDifferentProperties {
		if err := w.deleteOverwritten(ctx, &channels); err != nil {
			return err
		}
	}

	// Auto-create index channels for calculated channels that do not yet have one (both
	// brand-new calculated channels and existing ones missing an index).
	indexChannels := make([]Channel, 0, len(channels))
	calcNeedingIndex := make([]int, 0)
	for i, ch := range channels {
		if ch.IsCalculated() && ch.LocalIndex == 0 {
			indexChannels = append(indexChannels, Channel{
				Name:        ch.Name + calculatedIndexNameSuffix,
				DataType:    telem.TimeStampT,
				IsIndex:     true,
				Virtual:     true,
				Leaseholder: node.KeyFree,
				Internal:    ch.Internal,
			})
			calcNeedingIndex = append(calcNeedingIndex, i)
		}
	}
	channels = append(channels, indexChannels...)

	if err := w.validateFreeVirtual(channels); err != nil {
		return err
	}

	toCreate, err := w.resolveExistingAndAssignKeys(ctx, &channels, opts.retrieveIfNameExists)
	if err != nil {
		return err
	}

	w.linkCalculatedIndexes(toCreate, channels)

	if err := w.writeChannels(ctx, toCreate); err != nil {
		return err
	}

	// Persist updated LocalIndex links for existing calculated channels whose index was
	// just created.
	if err := w.updateCalculatedIndexLinks(ctx, channels, calcNeedingIndex); err != nil {
		return err
	}

	*_channels = channels
	return w.maybeSetResources(ctx, channels, opts)
}

// updateExisting updates the stored records of channels passed in with a non-zero key,
// applying name and (for calculated channels) expression/operations/index/data-type
// changes. When RetrieveIfNameExists is set the in-memory channel is reset to the stored
// record instead.
func (w Writer) updateExisting(ctx context.Context, channels *[]Channel, opts createOptions) error {
	keys := KeysFromChannels(*channels)
	existingKeys := lo.Filter(keys, func(k Key, _ int) bool { return k.LocalKey() != 0 })
	if len(existingKeys) == 0 {
		return nil
	}
	err := w.svc.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Channel](existingKeys...)).
		ChangeErr(func(_ gorp.Context, c Channel) (Channel, error) {
			idx := lo.IndexOf(keys, c.Key())
			ic := (*channels)[idx]
			if opts.retrieveIfNameExists {
				(*channels)[idx] = c
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
		Exec(ctx, w.tx)
	if err != nil && !errors.Is(err, query.ErrNotFound) {
		return err
	}
	return nil
}

// resolveExistingAndAssignKeys resolves channels that already exist by name (when
// retrieveIfNameExists is set, reusing their stored records) and allocates local keys
// plus storage for the remaining new channels through the distribution allocator. It
// returns the set of channels that were newly created.
func (w Writer) resolveExistingAndAssignKeys(
	ctx context.Context,
	channels *[]Channel,
	retrieveIfNameExists bool,
) (toCreate []Channel, err error) {
	if retrieveIfNameExists {
		names := Names(*channels)
		var existing []Channel
		if err = w.svc.newRetrieve().
			Where(MatchNames(names...)).
			Entries(&existing).
			Exec(ctx, w.tx); err != nil {
			return nil, errors.Skip(err, query.ErrNotFound)
		}
		for _, e := range existing {
			idx := lo.IndexOf(names, e.Name)
			if idx < 0 {
				continue
			}
			(*channels)[idx] = e
		}
	}

	newIndices := make([]int, 0, len(*channels))
	minimal := make([]channel.Channel, 0, len(*channels))
	for i, ch := range *channels {
		if ch.LocalKey == 0 {
			newIndices = append(newIndices, i)
			minimal = append(minimal, ch.Distribution())
		}
	}
	if len(minimal) == 0 {
		return toCreate, nil
	}
	allocated, allocErr := w.svc.cfg.Channel.Create(ctx, minimal)
	if allocErr != nil {
		return nil, allocErr
	}
	toCreate = make([]Channel, 0, len(newIndices))
	for j, i := range newIndices {
		(*channels)[i].LocalKey = allocated[j].LocalKey
		(*channels)[i].LocalIndex = allocated[j].LocalIndex
		toCreate = append(toCreate, (*channels)[i])
	}
	return toCreate, nil
}

// linkCalculatedIndexes links each calculated channel to its auto-created index channel
// by setting the calculated channel's LocalIndex to the index channel's assigned
// LocalKey. calcNeedingIndex holds the indices (into channels) of calculated channels
// that had an index auto-created for them.
func (w Writer) linkCalculatedIndexes(toCreate []Channel, channels []Channel) {
	indexKeyByName := make(map[string]LocalKey, len(channels))
	for _, ch := range channels {
		if ch.IsIndex {
			indexKeyByName[ch.Name] = ch.LocalKey
		}
	}
	link := func(set []Channel) {
		for i := range set {
			if !set[i].IsCalculated() || set[i].LocalIndex != 0 {
				continue
			}
			if k, ok := indexKeyByName[set[i].Name+calculatedIndexNameSuffix]; ok {
				set[i].LocalIndex = k
			}
		}
	}
	link(toCreate)
	link(channels)
}

// updateCalculatedIndexLinks persists the LocalIndex link for existing calculated
// channels (those already in the table) whose index channel was just created.
func (w Writer) updateCalculatedIndexLinks(ctx context.Context, channels []Channel, calcNeedingIndex []int) error {
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
	return nil
}

// writeChannels enforces the external-channel overflow cap and writes the newly created
// channels to the table, updating the external/non-virtual key set.
func (w Writer) writeChannels(ctx context.Context, toCreate []Channel) error {
	externalCreatedKeys := make(Keys, 0, len(toCreate))
	for _, ch := range toCreate {
		if !ch.Internal && !ch.Virtual {
			externalCreatedKeys = append(externalCreatedKeys, ch.Key())
		}
	}
	w.svc.external.mu.Lock()
	defer w.svc.external.mu.Unlock()
	count := w.svc.external.set.Size()
	if err := w.svc.cfg.IntOverflowCheck(types.Uint20(int(count) + len(externalCreatedKeys))); err != nil {
		return err
	}
	if err := w.svc.table.NewCreate().Entries(&toCreate).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.svc.external.set.Insert(externalCreatedKeys...)
	return nil
}

// validateChannelNames rejects a create/rename request whose proposed names either
// duplicate each other or collide with an existing channel under a different key.
func (w Writer) validateChannelNames(
	ctx context.Context,
	keys Keys,
	names []string,
	skipExisting bool,
) error {
	for i, name := range names {
		if err := ValidateName(name); err != nil {
			return validate.PathedError(err, fmt.Sprintf("[%d].name", i))
		}
	}
	namesSeen := make(set.Set[string], len(names))
	for i, name := range names {
		if namesSeen.Contains(name) {
			return validate.PathedError(
				errors.Wrapf(validate.ErrValidation, "duplicate channel name '%s' in request", name),
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
			errors.Wrapf(validate.ErrValidation, "channel with name '%s' already exists", name),
			fmt.Sprintf("[%d].name", i),
		)
	}
	return nil
}

func (w Writer) validateFreeVirtual(channels []Channel) error {
	for _, ch := range channels {
		if len(ch.Name) == 0 {
			return validate.PathedError(validate.ErrRequired, "name")
		}
	}
	return nil
}

func (w Writer) deleteOverwritten(ctx context.Context, channels *[]Channel) error {
	names := Names(*channels)
	if len(names) == 0 {
		return nil
	}
	var existing []Channel
	if err := w.svc.newRetrieve().
		Where(MatchNames(names...)).
		Entries(&existing).
		Exec(ctx, w.tx); err != nil {
		return errors.Skip(err, query.ErrNotFound)
	}
	keysToDelete := make(Keys, 0, len(existing))
	for _, ex := range existing {
		ch, i, found := lo.FindIndexOf(*channels, func(ch Channel) bool {
			return ch.Name == ex.Name && ch.Key() != ex.Key()
		})
		if !found {
			continue
		}
		if ch.Equals(ex, "LocalKey", "LocalIndex", "Leaseholder") {
			(*channels)[i] = ex
			continue
		}
		keysToDelete = append(keysToDelete, ex.Key())
	}
	if len(keysToDelete) == 0 {
		return nil
	}
	if err := w.svc.table.NewDelete().
		Where(gorp.MatchKeys[Key, Channel](keysToDelete...)).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.svc.cfg.Channel.Delete(ctx, keysToDelete)
}

func (w Writer) maybeSetResources(
	ctx context.Context,
	channels []Channel,
	opts createOptions,
) error {
	if w.svc.cfg.Ontology == nil || w.svc.cfg.Group == nil {
		return nil
	}
	externalIDs := lo.FilterMap(channels, func(ch Channel, _ int) (ontology.ID, bool) {
		return OntologyID(ch.Key()), !ch.Internal
	})
	ow := w.svc.cfg.Ontology.NewWriter(w.tx)
	if err := ow.DefineResource(ctx, externalIDs...); err != nil {
		return err
	}
	if opts.createWithoutGroupRelationship {
		return nil
	}
	return ow.DefineRelationship(
		ctx,
		group.OntologyID(w.svc.group.Key),
		ontology.RelationshipTypeParentOf,
		externalIDs...,
	)
}

func (w Writer) deleteByName(ctx context.Context, names []string, allowInternal bool) error {
	var res []Channel
	if err := w.svc.newRetrieve().
		Where(MatchNames(names...)).
		Entries(&res).
		Exec(ctx, w.tx); err != nil {
		return errors.Skip(err, query.ErrNotFound)
	}
	return w.delete(ctx, KeysFromChannels(res), allowInternal)
}

func (w Writer) delete(ctx context.Context, keys Keys, allowInternal bool) error {
	if !allowInternal {
		internalChannels := make([]Channel, 0, len(keys))
		if err := w.svc.newRetrieve().
			Where(MatchKeys(keys...)).
			Where(MatchInternal(true)).
			Entries(&internalChannels).
			Exec(ctx, w.tx); err != nil {
			return errors.Skip(err, query.ErrNotFound)
		}
		if len(internalChannels) > 0 {
			names := make([]string, 0, len(internalChannels))
			for _, ch := range internalChannels {
				names = append(names, ch.Name)
			}
			return errors.Newf("can't delete internal channel(s): %v", names)
		}
	}
	if err := w.svc.table.NewDelete().Where(gorp.MatchKeys[Key, Channel](keys...)).Exec(ctx, w.tx); err != nil {
		return err
	}
	if err := w.maybeDeleteResources(ctx, keys); err != nil {
		return err
	}
	// Storage deletion goes last, as it is the only operation that can fail without an
	// atomic guarantee.
	if err := w.svc.cfg.Channel.Delete(ctx, keys); err != nil {
		return err
	}
	w.svc.external.mu.Lock()
	w.svc.external.set.Remove(keys...)
	w.svc.external.mu.Unlock()
	return nil
}

func (w Writer) maybeDeleteResources(ctx context.Context, keys Keys) error {
	if w.svc.cfg.Ontology == nil {
		return nil
	}
	ids := lo.Map(keys, func(k Key, _ int) ontology.ID { return OntologyID(k) })
	ow := w.svc.cfg.Ontology.NewWriter(w.tx)
	return ow.DeleteResource(ctx, ids...)
}

func channelNameUpdater(allowInternal bool, keys Keys, names []string) gorp.ChangeFunc[Key, Channel] {
	return func(_ gorp.Context, c Channel) (Channel, error) {
		if c.Internal && !allowInternal {
			return c, errors.Wrapf(validate.ErrValidation, "cannot rename internal channel %v", c)
		}
		c.Name = names[lo.IndexOf(keys, c.Key())]
		return c, nil
	}
}

func (w Writer) rename(
	ctx context.Context,
	keys Keys,
	names []string,
	allowInternal bool,
) error {
	if len(keys) != len(names) {
		return errors.Wrap(validate.ErrValidation, "keys and names must be the same length")
	}
	if *w.svc.cfg.ValidateNames {
		if err := w.validateChannelNames(ctx, keys, names, false); err != nil {
			return err
		}
	}
	if err := w.svc.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Channel](keys...)).
		ChangeErr(channelNameUpdater(allowInternal, keys, names)).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.svc.cfg.Channel.Rename(ctx, keys, names)
}
