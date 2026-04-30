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
	"go/types"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/proxy"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	xtypes "github.com/synnaxlabs/x/types"
	"github.com/synnaxlabs/x/validate"
)

const calculatedIndexNameSuffix = "_time"

func (s *Service) createHandler(ctx context.Context, msg CreateMessage) (CreateMessage, error) {
	txn := s.cfg.ClusterDB.OpenTx()
	err := s.create(ctx, txn, &msg.Channels, msg.Opts)
	if err != nil {
		return CreateMessage{}, err
	}
	return CreateMessage{Channels: msg.Channels}, txn.Commit(ctx)
}

func (s *Service) deleteHandler(ctx context.Context, msg DeleteRequest) (types.Nil, error) {
	txn := s.cfg.ClusterDB.OpenTx()
	err := s.delete(ctx, txn, msg.Keys, false)
	if err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, txn.Commit(ctx)
}

func (s *Service) renameHandler(ctx context.Context, msg RenameRequest) (types.Nil, error) {
	txn := s.cfg.ClusterDB.OpenTx()
	err := s.rename(ctx, txn, msg.Keys, msg.Names, false)
	if err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, txn.Commit(ctx)
}

func (s *Service) create(ctx context.Context, tx gorp.Tx, _channels *[]Channel, opts CreateOptions) error {
	channels := *_channels
	if *s.cfg.ValidateNames {
		keys := KeysFromChannels(channels)
		names := Names(channels)
		if err := s.validateChannelNames(ctx, tx, keys, names, opts.RetrieveIfNameExists || opts.OverwriteIfNameExistsAndDifferentProperties); err != nil {
			return err
		}
	}
	for i, ch := range channels {
		if ch.Leaseholder == 0 {
			channels[i].Leaseholder = s.cfg.HostResolver.HostKey()
		}
		if ch.IsCalculated() {
			// Reject manually-specified indexes on calculated channels
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

	// Auto-create index channels for calculated channels (only for new calculated channels)
	indexChannels := make([]Channel, 0, len(channels))
	for _, ch := range channels {
		if ch.IsCalculated() && ch.LocalKey == 0 {
			indexCh := Channel{
				Name:        ch.Name + calculatedIndexNameSuffix,
				DataType:    telem.TimeStampT,
				IsIndex:     true,
				Virtual:     true,
				Leaseholder: node.KeyFree,
				Internal:    ch.Internal,
			}
			indexChannels = append(indexChannels, indexCh)
		}
	}

	// Append index channels to be created alongside calculated channels
	channels = append(channels, indexChannels...)

	batch := s.createRouter.Batch(channels)
	oChannels := make([]Channel, 0, len(channels))
	for nodeKey, entries := range batch.Peers {
		remoteChannels, err := s.createRemote(ctx, nodeKey, entries, opts)
		if err != nil {
			return err
		}
		oChannels = append(oChannels, remoteChannels...)
	}
	if len(batch.Free) > 0 {
		if !s.cfg.HostResolver.HostKey().IsBootstrapper() {
			remoteChannels, err := s.createRemote(ctx, node.KeyBootstrapper, batch.Free, opts)
			if err != nil {
				return err
			}
			oChannels = append(oChannels, remoteChannels...)
		} else {
			if err := s.createAndUpdateFreeVirtual(ctx, tx, &batch.Free, opts); err != nil {
				return err
			}
			oChannels = append(oChannels, batch.Free...)
		}
	}
	if err := s.createGateway(ctx, tx, &batch.Gateway, opts); err != nil {
		return err
	}
	oChannels = append(oChannels, batch.Gateway...)
	*_channels = oChannels
	return s.maybeSetResources(ctx, tx, oChannels, opts)
}

func (s *Service) createAndUpdateFreeVirtual(
	ctx context.Context,
	tx gorp.Tx,
	channels *[]Channel,
	opts CreateOptions,
) error {
	if s.freeCounter == nil {
		panic("[channel.Service] - tried to assign virtual keys on non-bootstrapper")
	}
	if err := s.validateFreeVirtual(channels); err != nil {
		return err
	}

	// If existing channels are passed in, update the name and expression (for calculated channels)
	keys := KeysFromChannels(*channels)
	// Filter out zero keys (channels that don't exist yet)
	existingKeys := lo.Filter(keys, func(k Key, _ int) bool { return k != 0 })
	if len(existingKeys) > 0 {
		if err := s.table.NewUpdate().
			WhereKeys(existingKeys...).
			ChangeErr(
				func(_ gorp.Context, c Channel) (Channel, error) {
					idx := lo.IndexOf(keys, c.Key())
					ic := (*channels)[idx]
					// If RetrieveIfNameExists is true and user has provided channels to update, we need
					// to reset those channels to the actual values to ensure the user does not mistakenly
					// think the update was successful.
					if opts.RetrieveIfNameExists {
						(*channels)[idx] = c
						return c, nil
					}
					c.Name = ic.Name
					// Update expression for calculated channels
					if c.IsCalculated() && ic.IsCalculated() {
						c.Expression = ic.Expression
						c.Operations = ic.Operations
						c.LocalIndex = ic.LocalIndex
						c.DataType = ic.DataType
					}
					return c, nil
				}).
			Exec(ctx, tx); err != nil && !errors.Is(err, query.ErrNotFound) {
			return err
		}
	}

	if opts.OverwriteIfNameExistsAndDifferentProperties {
		if err := s.deleteOverwritten(ctx, tx, channels); err != nil {
			return err
		}
	}

	// Check for existing calculated channels that need index channels created
	indexChannelsForExisting := make([]Channel, 0)
	existingCalcChannelIndices := make([]int, 0) // Track which channels need linking
	for i, ch := range *channels {
		if ch.LocalKey != 0 && ch.IsCalculated() && ch.LocalIndex == 0 {
			indexCh := Channel{
				Name:        ch.Name + calculatedIndexNameSuffix,
				DataType:    telem.TimeStampT,
				IsIndex:     true,
				Virtual:     true,
				Leaseholder: node.KeyFree,
				Internal:    ch.Internal,
			}
			indexChannelsForExisting = append(indexChannelsForExisting, indexCh)
			existingCalcChannelIndices = append(existingCalcChannelIndices, i)
		}
	}
	// Add these index channels to the list to be created
	*channels = append(*channels, indexChannelsForExisting...)

	toCreate, err := s.retrieveExistingAndAssignKeys(
		ctx,
		tx,
		channels,
		s.freeCounter,
		opts.RetrieveIfNameExists,
	)
	if err != nil {
		return err
	}

	// Link calculated channels to their auto-created indexes
	// This handles both new calculated channels (in toCreate) and existing ones (not in toCreate)
	for i, ch := range toCreate {
		if ch.IsCalculated() && ch.LocalIndex == 0 {
			// Find the matching index channel by name
			indexName := ch.Name + calculatedIndexNameSuffix
			for _, potentialIndex := range toCreate {
				if potentialIndex.Name == indexName && potentialIndex.IsIndex {
					toCreate[i].LocalIndex = potentialIndex.LocalKey
					// Also update in the input channels slice so caller sees the link
					for k := range *channels {
						if (*channels)[k].Name == ch.Name && (*channels)[k].IsCalculated() {
							(*channels)[k].LocalIndex = potentialIndex.LocalKey
							break
						}
					}
					break
				}
			}
		}
	}

	// Link existing calculated channels to their newly created indexes
	// These channels are NOT in toCreate (they already have keys)
	existingChannelsToUpdate := make([]Channel, 0, len(existingCalcChannelIndices))
	for _, idx := range existingCalcChannelIndices {
		ch := (*channels)[idx]
		indexName := ch.Name + calculatedIndexNameSuffix
		// Find the index channel in the channels slice (it got a key assigned)
		for j := range *channels {
			if (*channels)[j].Name == indexName && (*channels)[j].IsIndex {
				indexKey := (*channels)[j].LocalKey
				// Update the calculated channel with the new index
				(*channels)[idx].LocalIndex = indexKey
				existingChannelsToUpdate = append(existingChannelsToUpdate, (*channels)[idx])
				break
			}
		}
	}

	if err := s.table.NewCreate().Entries(&toCreate).Exec(ctx,
		tx); err != nil {
		return err
	}

	// Update existing calculated channels with their new LocalIndex values
	if len(existingChannelsToUpdate) > 0 {
		for _, ch := range existingChannelsToUpdate {
			if err := s.table.NewUpdate().
				WhereKeys(ch.Key()).
				Change(func(_ gorp.Context, c Channel) Channel {
					c.LocalIndex = ch.LocalIndex
					return c
				}).
				Exec(ctx, tx); err != nil {
				return err
			}
		}
	}

	return s.maybeSetResources(ctx, tx, toCreate, opts)
}

func (s *Service) validateChannelNames(
	ctx context.Context,
	tx gorp.Tx,
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
	if err := s.table.NewRetrieve().
		Where(func(_ gorp.Context, c *Channel) (bool, error) {
			return namesSeen.Contains(c.Name), nil
		}).
		Entries(&conflictingChannels).Exec(ctx, tx); err != nil {
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

func (s *Service) validateFreeVirtual(channels *[]Channel) error {
	for _, ch := range *channels {
		if len(ch.Name) == 0 {
			return validate.PathedError(validate.ErrRequired, "name")
		}
	}
	return nil
}

func (s *Service) retrieveExistingAndAssignKeys(
	ctx context.Context,
	tx gorp.Tx,
	channels *[]Channel,
	counter *counter,
	retrieveIfNameExists bool,
) (toCreate []Channel, err error) {
	// This is the value we would increment by if RetrieveIfNameExists is false or
	// if we don't find any names that already exist.
	incCounterBy := LocalKey(len(*channels))
	if retrieveIfNameExists {
		names := Names(*channels)
		if err = s.table.NewRetrieve().Where(func(_ gorp.Context, c *Channel) (bool, error) {
			v := lo.IndexOf(names, c.Name)
			exists := v != -1
			if exists {
				// If it exists, replace it with the existing channel and decrement the
				// number of channels we need to create.
				(*channels)[v] = *c
				if incCounterBy != 0 {
					incCounterBy--
				}
			}
			return exists, nil
		}).Exec(ctx, tx); err != nil {
			return
		}
	}
	nextCounterValue, err := counter.add(ctx, incCounterBy)
	if err != nil {
		return
	}

	originalCounterValue := nextCounterValue - incCounterBy
	toCreate = make([]Channel, 0, incCounterBy)
	for i, ch := range *channels {
		if ch.LocalKey == 0 {
			ch.LocalKey = originalCounterValue + LocalKey(len(toCreate)) + 1
			if ch.IsIndex {
				ch.LocalIndex = ch.LocalKey
			}
			toCreate = append(toCreate, ch)
		} else if ch.IsIndex {
			ch.LocalIndex = ch.LocalKey
		}
		(*channels)[i] = ch
	}
	return toCreate, nil
}

func (s *Service) deleteOverwritten(
	ctx context.Context,
	tx gorp.Tx,
	channels *[]Channel,
) error {
	storageToDelete := make([]ts.ChannelKey, 0, len(*channels))
	if err := s.table.NewDelete().
		Where(func(_ gorp.Context, c *Channel) (bool, error) {
			ch, i, found := lo.FindIndexOf(*channels, func(ch Channel) bool {
				return ch.Name == c.Name && ch.Key() != c.Key()
			})
			equal := ch.Equals(*c, "LocalKey", "LocalIndex", "Leaseholder")
			shouldDelete := found && !equal
			if shouldDelete {
				storageToDelete = append(storageToDelete, c.Storage().Key)
			}
			if equal {
				(*channels)[i] = *c
			}
			return shouldDelete, nil
		}).Exec(ctx, tx); err != nil {
		return err
	}
	return s.cfg.TSChannel.DeleteChannels(storageToDelete)
}

func (s *Service) createGateway(
	ctx context.Context,
	tx gorp.Tx,
	channels *[]Channel,
	opts CreateOptions,
) error {
	if opts.OverwriteIfNameExistsAndDifferentProperties {
		if err := s.deleteOverwritten(ctx, tx, channels); err != nil {
			return err
		}
	}

	if err := s.validateFreeVirtual(channels); err != nil {
		return err
	}

	toCreate, err := s.retrieveExistingAndAssignKeys(ctx, tx, channels, s.leasedCounter, opts.RetrieveIfNameExists)
	if err != nil {
		return err
	}

	externalCreatedKeys := make(Keys, 0, len(toCreate))
	for _, ch := range toCreate {
		if !ch.Internal && !ch.Virtual {
			externalCreatedKeys = append(externalCreatedKeys, ch.Key())
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	count := s.mu.externalNonVirtualSet.Size()
	if err = s.cfg.IntOverflowCheck(xtypes.Uint20(int(count) + len(externalCreatedKeys))); err != nil {
		return err
	}
	storageChannels := toStorage(toCreate)
	if err = s.cfg.TSChannel.CreateChannel(ctx, storageChannels...); err != nil {
		return err
	}
	if err = s.table.NewCreate().
		Entries(&toCreate).
		Exec(ctx, tx); err != nil {
		return err
	}
	s.mu.externalNonVirtualSet.Insert(externalCreatedKeys...)
	return nil

}

func (s *Service) maybeSetResources(
	ctx context.Context,
	txn gorp.Tx,
	channels []Channel,
	opts CreateOptions,
) error {
	if s.cfg.Ontology == nil || s.cfg.Group == nil {
		return nil
	}
	externalIDs := lo.FilterMap(channels, func(ch Channel, _ int) (ontology.ID, bool) {
		return OntologyID(ch.Key()), !ch.Internal
	})
	w := s.cfg.Ontology.NewWriter(txn)
	if err := w.DefineManyResources(ctx, externalIDs); err != nil {
		return err
	}
	if opts.CreateWithoutGroupRelationship {
		return nil
	}
	return w.DefineFromOneToManyRelationships(
		ctx,
		group.OntologyID(s.group.Key),
		ontology.RelationshipTypeParentOf,
		externalIDs,
	)
}

func (s *Service) createRemote(
	ctx context.Context,
	target node.Key,
	channels []Channel,
	opts CreateOptions,
) ([]Channel, error) {
	addr, err := s.cfg.HostResolver.Resolve(target)
	if err != nil {
		return nil, err
	}
	cm := CreateMessage{Channels: channels, Opts: opts}
	res, err := s.cfg.Transport.CreateClient().Send(ctx, addr, cm)
	if err != nil {
		return nil, err
	}
	return res.Channels, nil
}

func (s *Service) deleteByName(ctx context.Context, tx gorp.Tx, names []string, allowInternal bool) error {
	var res []Channel
	if err := s.table.NewRetrieve().Entries(&res).Where(func(ctx gorp.Context, c *Channel) (bool, error) {
		return lo.Contains(names, c.Name), nil
	}).Exec(ctx, tx); err != nil {
		return err
	}
	keys := KeysFromChannels(res)
	return s.delete(ctx, tx, keys, allowInternal)
}

func (s *Service) delete(ctx context.Context, tx gorp.Tx, keys Keys, allowInternal bool) error {
	if !allowInternal {
		internalChannels := make([]Channel, 0, len(keys))
		if err := s.table.NewRetrieve().
			WhereKeys(keys...).
			Where(func(ctx gorp.Context, c *Channel) (bool, error) {
				return c.Internal, nil
			}).
			Entries(&internalChannels).
			Exec(ctx, tx); err != nil {
			return err
		}
		if len(internalChannels) > 0 {
			names := make([]string, 0, len(internalChannels))
			for _, ch := range internalChannels {
				names = append(names, ch.Name)
			}
			return errors.Newf("can't delete internal channel(s): %v", names)
		}
	}

	batch := s.keyRouter.Batch(keys)
	for nodeKey, entries := range batch.Peers {
		err := s.deleteRemote(ctx, nodeKey, entries)
		if err != nil {
			return err
		}
	}
	if len(batch.Free) > 0 {
		err := s.deleteFreeVirtual(ctx, tx, batch.Free)
		if err != nil {
			return err
		}
	}
	if err := s.deleteGateway(ctx, tx, batch.Gateway); err != nil {
		return err
	}
	return s.maybeDeleteResources(ctx, tx, keys)
}

func (s *Service) deleteFreeVirtual(ctx context.Context, tx gorp.Tx, channels Keys) error {
	return s.table.NewDelete().WhereKeys(channels...).Exec(ctx, tx)
}

func (s *Service) deleteGateway(ctx context.Context, tx gorp.Tx, keys Keys) error {
	if err := s.table.NewDelete().WhereKeys(keys...).Exec(ctx, tx); err != nil {
		return err
	}
	if err := s.maybeDeleteResources(ctx, tx, keys); err != nil {
		return err
	}
	// It's very important that this goes last, as it's the only operation that can fail
	// without an atomic guarantee.
	if err := s.cfg.TSChannel.DeleteChannels(keys.Storage()); err != nil {
		return err
	}
	s.mu.Lock()
	s.mu.externalNonVirtualSet.Remove(keys...)
	s.mu.Unlock()
	return nil
}

func (s *Service) maybeDeleteResources(
	ctx context.Context,
	tx gorp.Tx,
	keys Keys,
) error {
	if s.cfg.Ontology == nil {
		return nil
	}
	ids := lo.Map(keys, func(k Key, _ int) ontology.ID { return OntologyID(k) })
	w := s.cfg.Ontology.NewWriter(tx)
	return w.DeleteManyResources(ctx, ids)
}

func (s *Service) deleteRemote(ctx context.Context, target node.Key, keys Keys) error {
	addr, err := s.cfg.HostResolver.Resolve(target)
	if err != nil {
		return err
	}
	_, err = s.cfg.Transport.DeleteClient().Send(ctx, addr, DeleteRequest{Keys: keys})
	return err
}

type renameBatchEntry struct {
	name string
	key  Key
}

var _ proxy.Entry = renameBatchEntry{}

func (r renameBatchEntry) Lease() node.Key { return r.key.Lease() }

func unzipRenameBatch(entries []renameBatchEntry) ([]Key, []string) {
	return lo.UnzipBy2(entries, func(e renameBatchEntry) (Key, string) {
		return e.key, e.name
	})
}

func newRenameBatch(keys Keys, names []string) []renameBatchEntry {
	return lo.ZipBy2(keys, names, func(k Key, n string) renameBatchEntry {
		return renameBatchEntry{key: k, name: n}
	})
}

func (s *Service) rename(
	ctx context.Context,
	tx gorp.Tx,
	keys Keys,
	names []string,
	allowInternal bool,
) error {
	if len(keys) != len(names) {
		return errors.Wrap(validate.ErrValidation, "keys and names must be the same length")
	}
	if *s.cfg.ValidateNames {
		if err := s.validateChannelNames(ctx, tx, keys, names, false); err != nil {
			return err
		}
	}

	batch := s.renameRouter.Batch(newRenameBatch(keys, names))
	for nodeKey, entries := range batch.Peers {
		keys, names := unzipRenameBatch(entries)
		if err := s.renameRemote(ctx, nodeKey, keys, names); err != nil {
			return err
		}
	}
	if len(batch.Free) > 0 {
		keys, names := unzipRenameBatch(batch.Free)
		if err := s.renameFreeVirtual(ctx, tx, keys, names, allowInternal); err != nil {
			return err
		}
	}
	if len(batch.Gateway) > 0 {
		keys, names := unzipRenameBatch(batch.Gateway)
		return s.renameGateway(ctx, tx, keys, names, allowInternal)
	}
	return nil
}

func (s *Service) renameRemote(ctx context.Context, target node.Key, keys Keys, names []string) error {
	addr, err := s.cfg.HostResolver.Resolve(target)
	if err != nil {
		return err
	}
	_, err = s.cfg.Transport.RenameClient().Send(ctx, addr, RenameRequest{Keys: keys, Names: names})
	return err
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

func (s *Service) renameFreeVirtual(ctx context.Context, tx gorp.Tx, channels Keys, names []string, allowInternal bool) error {
	return s.table.NewUpdate().
		WhereKeys(channels...).
		ChangeErr(channelNameUpdater(allowInternal, channels, names)).
		Exec(ctx, tx)
}

func (s *Service) renameGateway(ctx context.Context, tx gorp.Tx, keys Keys, names []string, allowInternal bool) error {
	if err := s.table.NewUpdate().
		WhereKeys(keys...).
		ChangeErr(channelNameUpdater(allowInternal, keys, names)).
		Exec(ctx, tx); err != nil {
		return err
	}
	return s.cfg.TSChannel.RenameChannels(ctx, keys.Storage(), names)
}
