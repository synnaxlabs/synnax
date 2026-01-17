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
	"sync"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
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

type leaseProxy struct {
	cfg                ServiceConfig
	leasedCounter      *counter
	freeCounter        *counter
	analyzeCalculation CalculationAnalyzer
	group              group.Group
	mu                 struct {
		externalNonVirtualSet *set.Integer[Key]
		sync.RWMutex
	}
	createRouter proxy.BatchFactory[Channel]
	renameRouter proxy.BatchFactory[renameBatchEntry]
	keyRouter    proxy.BatchFactory[Key]
}

const calculatedIndexNameSuffix = "_time"

func newLeaseProxy(
	ctx context.Context,
	cfg ServiceConfig,
	group group.Group,
) (*leaseProxy, error) {
	leasedCounterKey := []byte(cfg.HostResolver.HostKey().String() + ".distribution.channel.leasedCounter")
	c, err := openCounter(ctx, cfg.ClusterDB, leasedCounterKey)
	if err != nil {
		return nil, err
	}
	keyRouter := proxy.BatchFactory[Key]{Host: cfg.HostResolver.HostKey()}
	var externalNonVirtualChannels []Channel
	if err := gorp.
		NewRetrieve[Key, Channel]().
		Where(func(_ gorp.Context, c *Channel) (bool, error) {
			return !c.Internal && !c.Virtual, nil
		}).
		Entries(&externalNonVirtualChannels).
		Exec(ctx, cfg.ClusterDB); err != nil {
		return nil, err
	}

	p := &leaseProxy{
		cfg:           cfg,
		createRouter:  proxy.BatchFactory[Channel]{Host: cfg.HostResolver.HostKey()},
		keyRouter:     keyRouter,
		renameRouter:  proxy.BatchFactory[renameBatchEntry]{Host: cfg.HostResolver.HostKey()},
		leasedCounter: c,
		group:         group,
	}
	p.mu.externalNonVirtualSet = set.NewInteger(KeysFromChannels(externalNonVirtualChannels))
	if cfg.HostResolver.HostKey() == cluster.NodeKeyBootstrapper {
		freeCounterKey := []byte(cfg.HostResolver.HostKey().String() + ".distribution.channel.counter.free")
		c, err := openCounter(ctx, cfg.ClusterDB, freeCounterKey)
		if err != nil {
			return nil, err
		}
		p.freeCounter = c
	}
	p.cfg.Transport.CreateServer().BindHandler(p.createHandler)
	p.cfg.Transport.DeleteServer().BindHandler(p.deleteHandler)
	p.cfg.Transport.RenameServer().BindHandler(p.renameHandler)
	return p, nil
}

func (lp *leaseProxy) createHandler(ctx context.Context, msg CreateMessage) (CreateMessage, error) {
	txn := lp.cfg.ClusterDB.OpenTx()
	err := lp.create(ctx, txn, &msg.Channels, msg.Opts)
	if err != nil {
		return CreateMessage{}, err
	}
	return CreateMessage{Channels: msg.Channels}, txn.Commit(ctx)
}

func (lp *leaseProxy) deleteHandler(ctx context.Context, msg DeleteRequest) (types.Nil, error) {
	txn := lp.cfg.ClusterDB.OpenTx()
	err := lp.delete(ctx, txn, msg.Keys, false)
	if err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, txn.Commit(ctx)
}

func (lp *leaseProxy) renameHandler(ctx context.Context, msg RenameRequest) (types.Nil, error) {
	txn := lp.cfg.ClusterDB.OpenTx()
	err := lp.rename(ctx, txn, msg.Keys, msg.Names, false)
	if err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, txn.Commit(ctx)
}

func (lp *leaseProxy) create(ctx context.Context, tx gorp.Tx, _channels *[]Channel, opts CreateOptions) error {
	channels := *_channels
	if *lp.cfg.ValidateNames {
		keys := KeysFromChannels(channels)
		names := Names(channels)
		if err := lp.validateChannelNames(ctx, tx, keys, names, opts.RetrieveIfNameExists || opts.OverwriteIfNameExistsAndDifferentProperties); err != nil {
			return err
		}
	}
	for i, ch := range channels {
		if ch.Leaseholder == 0 {
			channels[i].Leaseholder = lp.cfg.HostResolver.HostKey()
		}
		if ch.IsCalculated() {
			// Reject manually-specified indexes on calculated channels
			if ch.LocalIndex != 0 && ch.LocalKey == 0 {
				return validate.PathedError(
					errors.Wrap(validate.ErrValidation, "calculated channels cannot specify an index manually"),
					"local_index",
				)
			}
			channels[i].Leaseholder = cluster.NodeKeyFree
			channels[i].Virtual = true
			if lp.analyzeCalculation != nil {
				dt, err := lp.analyzeCalculation(ctx, ch.Expression)
				if err != nil {
					return err
				}
				channels[i].DataType = dt
			}
			// Perform analysis on calculated channels.
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
				Leaseholder: cluster.NodeKeyFree,
				Internal:    ch.Internal,
			}
			indexChannels = append(indexChannels, indexCh)
		}
	}

	// Append index channels to be created alongside calculated channels
	channels = append(channels, indexChannels...)

	batch := lp.createRouter.Batch(channels)
	oChannels := make([]Channel, 0, len(channels))
	for nodeKey, entries := range batch.Peers {
		remoteChannels, err := lp.createRemote(ctx, nodeKey, entries, opts)
		if err != nil {
			return err
		}
		oChannels = append(oChannels, remoteChannels...)
	}
	if len(batch.Free) > 0 {
		if !lp.cfg.HostResolver.HostKey().IsBootstrapper() {
			remoteChannels, err := lp.createRemote(ctx, cluster.NodeKeyBootstrapper, batch.Free, opts)
			if err != nil {
				return err
			}
			oChannels = append(oChannels, remoteChannels...)
		} else {
			if err := lp.createAndUpdateFreeVirtual(ctx, tx, &batch.Free, opts); err != nil {
				return err
			}
			oChannels = append(oChannels, batch.Free...)
		}
	}
	if err := lp.createGateway(ctx, tx, &batch.Gateway, opts); err != nil {
		return err
	}
	oChannels = append(oChannels, batch.Gateway...)
	*_channels = oChannels
	return lp.maybeSetResources(ctx, tx, oChannels)
}

func (lp *leaseProxy) createAndUpdateFreeVirtual(
	ctx context.Context,
	tx gorp.Tx,
	channels *[]Channel,
	opt CreateOptions,
) error {
	if lp.freeCounter == nil {
		panic("[leaseProxy] - tried to assign virtual keys on non-bootstrapper")
	}
	if err := lp.validateFreeVirtual(channels); err != nil {
		return err
	}

	// If existing channels are passed in, update the name and expression (for calculated channels)
	keys := KeysFromChannels(*channels)
	// Filter out zero keys (channels that don't exist yet)
	existingKeys := lo.Filter(keys, func(k Key, _ int) bool { return k != 0 })
	if len(existingKeys) > 0 {
		if err := gorp.NewUpdate[Key, Channel]().
			WhereKeys(existingKeys...).
			ChangeErr(
				func(_ gorp.Context, c Channel) (Channel, error) {
					idx := lo.IndexOf(keys, c.Key())
					ic := (*channels)[idx]
					// If RetrieveIfNameExists is true and user has provided channels to update, we need
					// to reset those channels to the actual values to ensure the user does not mistakenly
					// think the update was successful.
					if opt.RetrieveIfNameExists {
						(*channels)[idx] = c
						return c, nil
					}
					c.Name = ic.Name
					// Update expression for calculated channels
					if c.IsCalculated() && ic.IsCalculated() {
						c.Expression = ic.Expression
						c.Operations = ic.Operations
						c.LocalIndex = ic.LocalIndex
					}
					return c, nil
				}).
			Exec(ctx, tx); err != nil && !errors.Is(err, query.ErrNotFound) {
			return err
		}
	}

	if opt.OverwriteIfNameExistsAndDifferentProperties {
		if err := lp.deleteOverwritten(ctx, tx, channels); err != nil {
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
				Leaseholder: cluster.NodeKeyFree,
				Internal:    ch.Internal,
			}
			indexChannelsForExisting = append(indexChannelsForExisting, indexCh)
			existingCalcChannelIndices = append(existingCalcChannelIndices, i)
		}
	}
	// Add these index channels to the list to be created
	*channels = append(*channels, indexChannelsForExisting...)

	toCreate, err := lp.retrieveExistingAndAssignKeys(
		ctx,
		tx,
		channels,
		lp.freeCounter,
		opt.RetrieveIfNameExists,
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

	if err := gorp.NewCreate[Key, Channel]().Entries(&toCreate).Exec(ctx,
		tx); err != nil {
		return err
	}

	// Update existing calculated channels with their new LocalIndex values
	if len(existingChannelsToUpdate) > 0 {
		for _, ch := range existingChannelsToUpdate {
			if err := gorp.NewUpdate[Key, Channel]().
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

	return lp.maybeSetResources(ctx, tx, toCreate)
}

func (lp *leaseProxy) validateChannelNames(
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
	if err := gorp.NewRetrieve[Key, Channel]().
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

func (lp *leaseProxy) validateFreeVirtual(channels *[]Channel) error {
	for _, ch := range *channels {
		if len(ch.Name) == 0 {
			return validate.PathedError(validate.ErrRequired, "name")
		}
	}
	return nil
}

func (lp *leaseProxy) retrieveExistingAndAssignKeys(
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
		if err = gorp.NewRetrieve[Key, Channel]().Where(func(_ gorp.Context, c *Channel) (bool, error) {
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
	nextCounterValue, err := counter.add(incCounterBy)
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

func (lp *leaseProxy) deleteOverwritten(
	ctx context.Context,
	tx gorp.Tx,
	channels *[]Channel,
) error {
	storageToDelete := make([]ts.ChannelKey, 0, len(*channels))
	if err := gorp.NewDelete[Key, Channel]().
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
	return lp.cfg.TSChannel.DeleteChannels(storageToDelete)
}

func (lp *leaseProxy) createGateway(
	ctx context.Context,
	tx gorp.Tx,
	channels *[]Channel,
	opts CreateOptions,
) error {
	if opts.OverwriteIfNameExistsAndDifferentProperties {
		if err := lp.deleteOverwritten(ctx, tx, channels); err != nil {
			return err
		}
	}

	if err := lp.validateFreeVirtual(channels); err != nil {
		return err
	}

	toCreate, err := lp.retrieveExistingAndAssignKeys(ctx, tx, channels, lp.leasedCounter, opts.RetrieveIfNameExists)
	if err != nil {
		return err
	}

	externalCreatedKeys := make(Keys, 0, len(toCreate))
	for _, ch := range toCreate {
		if !ch.Internal && !ch.Virtual {
			externalCreatedKeys = append(externalCreatedKeys, ch.Key())
		}
	}
	lp.mu.Lock()
	defer lp.mu.Unlock()
	count := lp.mu.externalNonVirtualSet.Size()
	if err = lp.cfg.IntOverflowCheck(xtypes.Uint20(int(count) + len(externalCreatedKeys))); err != nil {
		return err
	}
	storageChannels := toStorage(toCreate)
	if err = lp.cfg.TSChannel.CreateChannel(ctx, storageChannels...); err != nil {
		return err
	}
	if err = gorp.
		NewCreate[Key, Channel]().
		Entries(&toCreate).
		Exec(ctx, tx); err != nil {
		return err
	}
	lp.mu.externalNonVirtualSet.Insert(externalCreatedKeys...)
	return nil

}

func (lp *leaseProxy) maybeSetResources(
	ctx context.Context,
	txn gorp.Tx,
	channels []Channel,
) error {
	if lp.cfg.Ontology == nil || lp.cfg.Group == nil {
		return nil
	}
	externalIDs := lo.FilterMap(channels, func(ch Channel, _ int) (ontology.ID, bool) {
		return OntologyID(ch.Key()), !ch.Internal
	})
	w := lp.cfg.Ontology.NewWriter(txn)
	if err := w.DefineManyResources(ctx, externalIDs); err != nil {
		return err
	}
	return w.DefineFromOneToManyRelationships(
		ctx,
		group.OntologyID(lp.group.Key),
		ontology.RelationshipTypeParentOf,
		externalIDs,
	)
}

func (lp *leaseProxy) createRemote(
	ctx context.Context,
	target cluster.NodeKey,
	channels []Channel,
	opts CreateOptions,
) ([]Channel, error) {
	addr, err := lp.cfg.HostResolver.Resolve(target)
	if err != nil {
		return nil, err
	}
	cm := CreateMessage{Channels: channels, Opts: opts}
	res, err := lp.cfg.Transport.CreateClient().Send(ctx, addr, cm)
	if err != nil {
		return nil, err
	}
	return res.Channels, nil
}

func (lp *leaseProxy) deleteByName(ctx context.Context, tx gorp.Tx, names []string, allowInternal bool) error {
	var res []Channel
	if err := gorp.NewRetrieve[Key, Channel]().Entries(&res).Where(func(ctx gorp.Context, c *Channel) (bool, error) {
		return lo.Contains(names, c.Name), nil
	}).Exec(ctx, tx); err != nil {
		return err
	}
	keys := KeysFromChannels(res)
	return lp.delete(ctx, tx, keys, allowInternal)
}

func (lp *leaseProxy) delete(ctx context.Context, tx gorp.Tx, keys Keys, allowInternal bool) error {
	if !allowInternal {
		internalChannels := make([]Channel, 0, len(keys))
		if err := gorp.
			NewRetrieve[Key, Channel]().
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

	batch := lp.keyRouter.Batch(keys)
	for nodeKey, entries := range batch.Peers {
		err := lp.deleteRemote(ctx, nodeKey, entries)
		if err != nil {
			return err
		}
	}
	if len(batch.Free) > 0 {
		err := lp.deleteFreeVirtual(ctx, tx, batch.Free)
		if err != nil {
			return err
		}
	}
	if err := lp.deleteGateway(ctx, tx, batch.Gateway); err != nil {
		return err
	}
	return lp.maybeDeleteResources(ctx, tx, keys)
}

func (lp *leaseProxy) deleteFreeVirtual(ctx context.Context, tx gorp.Tx, channels Keys) error {
	return gorp.NewDelete[Key, Channel]().WhereKeys(channels...).Exec(ctx, tx)
}

func (lp *leaseProxy) deleteGateway(ctx context.Context, tx gorp.Tx, keys Keys) error {
	if err := gorp.NewDelete[Key, Channel]().WhereKeys(keys...).Exec(ctx, tx); err != nil {
		return err
	}
	if err := lp.maybeDeleteResources(ctx, tx, keys); err != nil {
		return err
	}
	// It's very important that this goes last, as it's the only operation that can fail
	// without an atomic guarantee.
	if err := lp.cfg.TSChannel.DeleteChannels(keys.Storage()); err != nil {
		return err
	}
	lp.mu.Lock()
	lp.mu.externalNonVirtualSet.Remove(keys...)
	lp.mu.Unlock()
	return nil
}

func (lp *leaseProxy) maybeDeleteResources(
	ctx context.Context,
	tx gorp.Tx,
	keys Keys,
) error {
	if lp.cfg.Ontology == nil {
		return nil
	}
	ids := lo.Map(keys, func(k Key, _ int) ontology.ID { return OntologyID(k) })
	w := lp.cfg.Ontology.NewWriter(tx)
	return w.DeleteManyResources(ctx, ids)
}

func (lp *leaseProxy) deleteRemote(ctx context.Context, target cluster.NodeKey, keys Keys) error {
	addr, err := lp.cfg.HostResolver.Resolve(target)
	if err != nil {
		return err
	}
	_, err = lp.cfg.Transport.DeleteClient().Send(ctx, addr, DeleteRequest{Keys: keys})
	return err
}

type renameBatchEntry struct {
	name string
	key  Key
}

var _ proxy.Entry = renameBatchEntry{}

func (r renameBatchEntry) Lease() cluster.NodeKey { return r.key.Lease() }

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

func (lp *leaseProxy) rename(
	ctx context.Context,
	tx gorp.Tx,
	keys Keys,
	names []string,
	allowInternal bool,
) error {
	if len(keys) != len(names) {
		return errors.Wrap(validate.ErrValidation, "keys and names must be the same length")
	}
	if *lp.cfg.ValidateNames {
		if err := lp.validateChannelNames(ctx, tx, keys, names, false); err != nil {
			return err
		}
	}

	batch := lp.renameRouter.Batch(newRenameBatch(keys, names))
	for nodeKey, entries := range batch.Peers {
		keys, names := unzipRenameBatch(entries)
		if err := lp.renameRemote(ctx, nodeKey, keys, names); err != nil {
			return err
		}
	}
	if len(batch.Free) > 0 {
		keys, names := unzipRenameBatch(batch.Free)
		if err := lp.renameFreeVirtual(ctx, tx, keys, names, allowInternal); err != nil {
			return err
		}
	}
	if len(batch.Gateway) > 0 {
		keys, names := unzipRenameBatch(batch.Gateway)
		return lp.renameGateway(ctx, tx, keys, names, allowInternal)
	}
	return nil
}

func (lp *leaseProxy) renameRemote(ctx context.Context, target cluster.NodeKey, keys Keys, names []string) error {
	addr, err := lp.cfg.HostResolver.Resolve(target)
	if err != nil {
		return err
	}
	_, err = lp.cfg.Transport.RenameClient().Send(ctx, addr, RenameRequest{Keys: keys, Names: names})
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

func (lp *leaseProxy) renameFreeVirtual(ctx context.Context, tx gorp.Tx, channels Keys, names []string, allowInternal bool) error {
	return gorp.NewUpdate[Key, Channel]().
		WhereKeys(channels...).
		ChangeErr(channelNameUpdater(allowInternal, channels, names)).
		Exec(ctx, tx)
}

func (lp *leaseProxy) renameGateway(ctx context.Context, tx gorp.Tx, keys Keys, names []string, allowInternal bool) error {
	if err := gorp.NewUpdate[Key, Channel]().
		WhereKeys(keys...).
		ChangeErr(channelNameUpdater(allowInternal, keys, names)).
		Exec(ctx, tx); err != nil {
		return err
	}
	return lp.cfg.TSChannel.RenameChannels(ctx, keys.Storage(), names)
}
