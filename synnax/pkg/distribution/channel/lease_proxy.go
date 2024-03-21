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
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/proxy"
	"github.com/synnaxlabs/x/errutil"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv"
)

type leaseProxy struct {
	ServiceConfig
	createRouter  proxy.BatchFactory[Channel]
	deleteRouter  proxy.BatchFactory[Key]
	leasedCounter *kv.AtomicInt64Counter
	freeCounter   *kv.AtomicInt64Counter
	group         group.Group
}

const leasedCounterSuffix = ".distribution.channel.leasedCounter"
const freeCounterSuffix = ".distribution.channel.counter.free"

func newLeaseProxy(cfg ServiceConfig, g group.Group) (*leaseProxy, error) {
	leasedCounterKey := []byte(cfg.HostResolver.HostKey().String() + leasedCounterSuffix)
	c, err := kv.OpenCounter(context.TODO(), cfg.ClusterDB, leasedCounterKey)
	if err != nil {
		return nil, err
	}

	p := &leaseProxy{
		ServiceConfig: cfg,
		createRouter:  proxy.BatchFactory[Channel]{Host: cfg.HostResolver.HostKey()},
		deleteRouter:  proxy.BatchFactory[Key]{Host: cfg.HostResolver.HostKey()},
		leasedCounter: c,
		group:         g,
	}
	if cfg.HostResolver.HostKey() == core.Bootstrapper {
		//freeCounterKey := []byte(cfg.HostResolver.HostKey().String() + freeCounterSuffix)
		//c, err := kv.OpenCounter(context.TODO(), cfg.ClusterDB, freeCounterKey)
		//if err != nil {
		//	return nil, err
		//}
		p.freeCounter = c
	}
	p.Transport.CreateServer().BindHandler(p.handle)
	return p, nil
}

func (lp *leaseProxy) handle(ctx context.Context, msg CreateMessage) (CreateMessage, error) {
	txn := lp.ClusterDB.OpenTx()
	err := lp.create(ctx, txn, &msg.Channels, msg.RetrieveIfNameExists)
	if err != nil {
		return CreateMessage{}, err
	}
	return CreateMessage{Channels: msg.Channels}, txn.Commit(ctx)
}

func (lp *leaseProxy) create(ctx context.Context, tx gorp.Tx, _channels *[]Channel, retrieveIfNameExists bool) error {
	channels := *_channels
	for i, ch := range channels {
		if ch.LocalKey != 0 {
			channels[i].LocalKey = 0
		}
		if ch.Leaseholder == 0 {
			channels[i].Leaseholder = lp.HostResolver.HostKey()
		}
	}
	batch := lp.createRouter.Batch(channels)
	oChannels := make([]Channel, 0, len(channels))
	for nodeKey, entries := range batch.Peers {
		remoteChannels, err := lp.createRemote(ctx, nodeKey, entries, retrieveIfNameExists)
		if err != nil {
			return err
		}
		oChannels = append(oChannels, remoteChannels...)
	}
	if len(batch.Free) > 0 {
		if !lp.HostResolver.HostKey().IsBootstrapper() {
			remoteChannels, err := lp.createRemote(ctx, core.Bootstrapper, batch.Free, retrieveIfNameExists)
			if err != nil {
				return err
			}
			oChannels = append(oChannels, remoteChannels...)
		} else {
			err := lp.createFreeVirtual(ctx, tx, &batch.Free, retrieveIfNameExists)
			if err != nil {
				return err
			}
			oChannels = append(oChannels, batch.Free...)
		}
	}
	err := lp.createGateway(ctx, tx, &batch.Gateway, retrieveIfNameExists)
	if err != nil {
		return err
	}
	oChannels = append(oChannels, batch.Gateway...)
	*_channels = oChannels
	return nil
}

func (lp *leaseProxy) createFreeVirtual(
	ctx context.Context,
	tx gorp.Tx,
	channels *[]Channel,
	retrieveIfNameExists bool,
) error {
	if lp.freeCounter == nil {
		panic("[leaseProxy] - tried to assign virtual keys on non-bootstrapper")
	}
	toCreate, err := lp.maybeRetrieveExisting(ctx, tx, channels, lp.freeCounter, retrieveIfNameExists)
	if err != nil {
		return err
	}
	if err := gorp.NewCreate[Key, Channel]().Entries(&toCreate).Exec(ctx, tx); err != nil {
		return err
	}
	return lp.maybeSetResources(ctx, tx, toCreate)
}

func (lp *leaseProxy) maybeRetrieveExisting(
	ctx context.Context,
	tx gorp.Tx,
	channels *[]Channel,
	counter *kv.AtomicInt64Counter,
	retrieveIfNameExists bool,
) (toCreate []Channel, err error) {
	// This is the value we would increment by if retrieveIfNameExists is false or
	// if we don't find any names that already exist.
	incCounterBy := uint16(len(*channels))

	if retrieveIfNameExists {
		names := NamesFromChannels(*channels)
		if err = gorp.NewRetrieve[Key, Channel]().Where(func(c *Channel) bool {
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
			return exists
		}).Exec(ctx, tx); err != nil {
			return
		}
	}

	v, err := counter.Add(int64(incCounterBy))
	if err != nil {
		return
	}

	toCreate = make([]Channel, 0, incCounterBy)
	for i, ch := range *channels {
		if ch.LocalKey == 0 {
			ch.LocalKey = uint16(v) - incCounterBy + uint16(len(toCreate)) + 1
			toCreate = append(toCreate, ch)
		} else if ch.IsIndex {
			ch.LocalIndex = ch.LocalKey
		}
		(*channels)[i] = ch
	}

	return toCreate, nil
}

func (lp *leaseProxy) createGateway(
	ctx context.Context,
	tx gorp.Tx,
	channels *[]Channel,
	retrieveIfNameExists bool,
) error {
	toCreate, err := lp.maybeRetrieveExisting(ctx, tx, channels, lp.leasedCounter, retrieveIfNameExists)
	if err != nil {
		return err
	}
	storageChannels := toStorage(toCreate)
	if err := lp.TSChannel.CreateChannel(ctx, storageChannels...); err != nil {
		return err
	}
	if err := gorp.NewCreate[Key, Channel]().Entries(&toCreate).Exec(ctx, tx); err != nil {
		return err
	}
	return lp.maybeSetResources(ctx, tx, toCreate)
}

func (lp *leaseProxy) maybeSetResources(
	ctx context.Context,
	txn gorp.Tx,
	channels []Channel,
) error {
	if lp.Ontology == nil {
		return nil
	}
	ids := lo.Map(channels, func(ch Channel, i int) ontology.ID {
		return OntologyID(ch.Key())
	})
	w := lp.Ontology.NewWriter(txn)
	if err := w.DefineManyResources(ctx, ids); err != nil {
		return err
	}
	if err := w.DefineFromOneToManyRelationships(
		ctx,
		group.OntologyID(lp.group.Key),
		ontology.ParentOf,
		ids,
	); err != nil {
		return err
	}
	return w.DefineFromOneToManyRelationships(
		ctx,
		core.NodeOntologyID(lp.HostResolver.HostKey()),
		ontology.ParentOf,
		ids,
	)
}

func (lp *leaseProxy) createRemote(
	ctx context.Context,
	target core.NodeKey,
	channels []Channel,
	retrieveIfNameExists bool,
) ([]Channel, error) {
	addr, err := lp.HostResolver.Resolve(target)
	if err != nil {
		return nil, err
	}
	cm := CreateMessage{Channels: channels, RetrieveIfNameExists: retrieveIfNameExists}
	res, err := lp.Transport.CreateClient().Send(ctx, addr, cm)
	if err != nil {
		return nil, err
	}
	return res.Channels, nil
}

func (lp *leaseProxy) deleteByName(ctx context.Context, tx gorp.Tx, names []string) error {
	var res []Channel
	if err := gorp.NewRetrieve[Key, Channel]().Entries(&res).Where(func(c *Channel) bool {
		return lo.Contains(names, c.Name)
	}).Exec(ctx, tx); err != nil {
		return err
	}
	keys := KeysFromChannels(res)
	return lp.delete(ctx, tx, keys)
}

func (lp *leaseProxy) delete(ctx context.Context, tx gorp.Tx, keys Keys) error {
	batch := lp.deleteRouter.Batch(keys)
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
	return lp.deleteGateway(ctx, tx, batch.Gateway)
}

func (lp *leaseProxy) deleteFreeVirtual(ctx context.Context, tx gorp.Tx, channels Keys) error {
	return gorp.NewDelete[Key, Channel]().WhereKeys(channels...).Exec(ctx, tx)
}

func (lp *leaseProxy) deleteGateway(ctx context.Context, tx gorp.Tx, keys Keys) error {
	if err := gorp.NewDelete[Key, Channel]().WhereKeys(keys...).Exec(ctx, tx); err != nil {
		return err
	}
	c := errutil.NewCatch(errutil.WithAggregation())
	for _, key := range keys {
		c.Exec(func() error { return lp.TSChannel.DeleteChannel(key.StorageKey()) })
	}
	return c.Error()
}

func (lp *leaseProxy) deleteRemote(ctx context.Context, target core.NodeKey, keys Keys) error {
	addr, err := lp.HostResolver.Resolve(target)
	if err != nil {
		return err
	}
	_, err = lp.Transport.DeleteClient().Send(ctx, addr, DeleteRequest{Keys: keys})
	return err
}
