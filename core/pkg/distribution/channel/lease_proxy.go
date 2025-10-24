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
	ServiceConfig
	createRouter       proxy.BatchFactory[Channel]
	renameRouter       proxy.BatchFactory[renameBatchEntry]
	keyRouter          proxy.BatchFactory[Key]
	leasedCounter      *counter
	freeCounter        *counter
	group              group.Group
	analyzeCalculation CalculationAnalyzer
	mu                 struct {
		sync.RWMutex
		externalNonVirtualSet *set.Integer[Key]
	}
}

const (
	leasedCounterSuffix       = ".distribution.channel.leasedCounter"
	freeCounterSuffix         = ".distribution.channel.counter.free"
	calculatedIndexNameSuffix = "_time"
)

func newLeaseProxy(
	ctx context.Context,
	cfg ServiceConfig,
	group group.Group,
) (*leaseProxy, error) {
	leasedCounterKey := []byte(cfg.HostResolver.HostKey().String() + leasedCounterSuffix)
	c, err := openCounter(ctx, cfg.ClusterDB, leasedCounterKey)
	if err != nil {
		return nil, err
	}
	keyRouter := proxy.BatchFactory[Key]{Host: cfg.HostResolver.HostKey()}
	var externalNonVirtualChannels []Channel
	if err := gorp.
		NewRetrieve[Key, Channel]().
		Where(func(ctx gorp.Context, c *Channel) (bool, error) {
			return !c.Internal && !c.Virtual, nil
		}).
		Entries(&externalNonVirtualChannels).
		Exec(ctx, cfg.ClusterDB); err != nil {
		return nil, err
	}

	p := &leaseProxy{
		ServiceConfig: cfg,
		createRouter:  proxy.BatchFactory[Channel]{Host: cfg.HostResolver.HostKey()},
		keyRouter:     keyRouter,
		renameRouter:  proxy.BatchFactory[renameBatchEntry]{Host: cfg.HostResolver.HostKey()},
		leasedCounter: c,
		group:         group,
	}
	p.mu.externalNonVirtualSet = set.NewInteger[Key](KeysFromChannels(externalNonVirtualChannels))
	if cfg.HostResolver.HostKey() == cluster.Bootstrapper {
		freeCounterKey := []byte(cfg.HostResolver.HostKey().String() + freeCounterSuffix)
		c, err := openCounter(ctx, cfg.ClusterDB, freeCounterKey)
		if err != nil {
			return nil, err
		}
		p.freeCounter = c
	}
	p.Transport.CreateServer().BindHandler(p.createHandler)
	p.Transport.DeleteServer().BindHandler(p.deleteHandler)
	p.Transport.RenameServer().BindHandler(p.renameHandler)
	return p, nil
}

func (lp *leaseProxy) createHandler(ctx context.Context, msg CreateMessage) (CreateMessage, error) {
	txn := lp.ClusterDB.OpenTx()
	err := lp.create(ctx, txn, &msg.Channels, msg.Opts)
	if err != nil {
		return CreateMessage{}, err
	}
	return CreateMessage{Channels: msg.Channels}, txn.Commit(ctx)
}

func (lp *leaseProxy) deleteHandler(ctx context.Context, msg DeleteRequest) (types.Nil, error) {
	txn := lp.ClusterDB.OpenTx()
	err := lp.delete(ctx, txn, msg.Keys, false)
	if err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, txn.Commit(ctx)
}

func (lp *leaseProxy) renameHandler(ctx context.Context, msg RenameRequest) (types.Nil, error) {
	txn := lp.ClusterDB.OpenTx()
	err := lp.rename(ctx, txn, msg.Keys, msg.Names, false)
	if err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, txn.Commit(ctx)
}

func (lp *leaseProxy) create(ctx context.Context, tx gorp.Tx, _channels *[]Channel, opts CreateOptions) error {
	channels := *_channels
	for i, ch := range channels {
		if ch.Leaseholder == 0 {
			channels[i].Leaseholder = lp.HostResolver.HostKey()
		}
		if ch.IsCalculated() {
			// Reject manually-specified indexes on calculated channels
			if ch.LocalIndex != 0 && ch.LocalKey == 0 {
				return validate.PathedError(
					errors.Wrap(validate.Error, "calculated channels cannot specify an index manually"),
					"local_index",
				)
			}
			channels[i].Leaseholder = cluster.Free
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
				Leaseholder: cluster.Free,
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
		if !lp.HostResolver.HostKey().IsBootstrapper() {
			remoteChannels, err := lp.createRemote(ctx, cluster.Bootstrapper, batch.Free, opts)
			if err != nil {
				return err
			}
			oChannels = append(oChannels, remoteChannels...)
		} else {
			err := lp.createAndUpdateFreeVirtual(ctx, tx, &batch.Free, opts)
			if err != nil {
				return err
			}
			oChannels = append(oChannels, batch.Free...)
		}
	}
	err := lp.createGateway(ctx, tx, &batch.Gateway, opts)
	if err != nil {
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
	if err := lp.validateFreeVirtual(ctx, channels, tx); err != nil {
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
					}
					return c, nil
				}).
			Exec(ctx, tx); err != nil && !errors.Is(err, query.NotFound) {
			return err
		}
	}

	if opt.OverwriteIfNameExistsAndDifferentProperties {
		if err := lp.deleteOverwritten(ctx, tx, channels); err != nil {
			return err
		}
	}

	toCreate, err := lp.retrieveExistingAndAssignKeys(ctx, tx, channels, lp.freeCounter, opt.RetrieveIfNameExists)
	if err != nil {
		return err
	}

	// Link calculated channels to their auto-created indexes
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

	if err := gorp.NewCreate[Key, Channel]().Entries(&toCreate).Exec(ctx,
		tx); err != nil {
		return err
	}
	return lp.maybeSetResources(ctx, tx, toCreate)
}

func (lp *leaseProxy) validateFreeVirtual(
	ctx context.Context,
	channels *[]Channel,
	tx gorp.Tx,
) error {
	for _, ch := range *channels {
		if len(ch.Name) == 0 {
			return validate.PathedError(validate.RequiredError, "name")
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
	return lp.TSChannel.DeleteChannels(storageToDelete)
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

	if err := lp.validateFreeVirtual(ctx, channels, tx); err != nil {
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
	count := lp.mu.externalNonVirtualSet.Size()
	if err = lp.IntOverflowCheck(xtypes.Uint20(int(count) + len(externalCreatedKeys))); err != nil {
		lp.mu.Unlock()
		return err
	}
	lp.mu.externalNonVirtualSet.Insert(externalCreatedKeys...)
	lp.mu.Unlock()

	storageChannels := toStorage(toCreate)
	if err = lp.TSChannel.CreateChannel(ctx, storageChannels...); err != nil {
		return err
	}
	if err = gorp.
		NewCreate[Key, Channel]().
		Entries(&toCreate).
		Exec(ctx, tx); err != nil {
		return err
	}
	return nil
}

func (lp *leaseProxy) maybeSetResources(
	ctx context.Context,
	txn gorp.Tx,
	channels []Channel,
) error {
	if lp.Ontology == nil || lp.Group == nil {
		return nil
	}
	externalIds := lo.FilterMap(channels, func(ch Channel, _ int) (ontology.ID, bool) {
		return OntologyID(ch.Key()), !ch.Internal
	})
	w := lp.Ontology.NewWriter(txn)
	if err := w.DefineManyResources(ctx, externalIds); err != nil {
		return err
	}
	return w.DefineFromOneToManyRelationships(
		ctx,
		group.OntologyID(lp.group.Key),
		ontology.ParentOf,
		externalIds,
	)
}

func (lp *leaseProxy) createRemote(
	ctx context.Context,
	target cluster.NodeKey,
	channels []Channel,
	opts CreateOptions,
) ([]Channel, error) {
	addr, err := lp.HostResolver.Resolve(target)
	if err != nil {
		return nil, err
	}
	cm := CreateMessage{Channels: channels, Opts: opts}
	res, err := lp.Transport.CreateClient().Send(ctx, addr, cm)
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
	lp.mu.Lock()
	lp.mu.externalNonVirtualSet.Remove(keys...)
	lp.mu.Unlock()
	// It's very important that this goes last, as it's the only operation that can fail
	// without an atomic guarantee.
	return lp.TSChannel.DeleteChannels(keys.Storage())
}

func (lp *leaseProxy) maybeDeleteResources(
	ctx context.Context,
	tx gorp.Tx,
	keys Keys,
) error {
	if lp.Ontology == nil {
		return nil
	}
	ids := lo.Map(keys, func(k Key, _ int) ontology.ID { return OntologyID(k) })
	w := lp.Ontology.NewWriter(tx)
	return w.DeleteManyResources(ctx, ids)
}

func (lp *leaseProxy) deleteRemote(ctx context.Context, target cluster.NodeKey, keys Keys) error {
	addr, err := lp.HostResolver.Resolve(target)
	if err != nil {
		return err
	}
	_, err = lp.Transport.DeleteClient().Send(ctx, addr, DeleteRequest{Keys: keys})
	return err
}

type renameBatchEntry struct {
	key  Key
	name string
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
		return errors.Wrap(validate.Error, "keys and names must be the same length")
	}
	batch := lp.renameRouter.Batch(newRenameBatch(keys, names))
	for nodeKey, entries := range batch.Peers {
		keys, names := unzipRenameBatch(entries)
		err := lp.renameRemote(ctx, nodeKey, keys, names)
		if err != nil {
			return err
		}
	}
	if len(batch.Free) > 0 {
		keys, names := unzipRenameBatch(batch.Free)
		err := lp.renameFreeVirtual(ctx, tx, keys, names, allowInternal)
		if err != nil {
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
	addr, err := lp.HostResolver.Resolve(target)
	if err != nil {
		return err
	}
	_, err = lp.Transport.RenameClient().Send(ctx, addr, RenameRequest{Keys: keys, Names: names})
	return err
}

func channelNameUpdater(allowInternal bool, keys Keys, names []string) gorp.ChangeFunc[Key, Channel] {
	return func(_ gorp.Context, c Channel) (Channel, error) {
		if c.Internal && !allowInternal {
			return c, errors.Wrapf(validate.Error, "cannot rename internal channel %v", c)
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
	return lp.TSChannel.RenameChannels(ctx, keys.Storage(), names)
}
