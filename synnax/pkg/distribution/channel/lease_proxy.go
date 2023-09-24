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

	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/proxy"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/validate"
)

type leaseProxy struct {
	ServiceConfig
	router  proxy.BatchFactory[Channel]
	counter *keyCounter
	group   group.Group
}

func newLeaseProxy(cfg ServiceConfig, g group.Group) (*leaseProxy, error) {
	c, err := openCounter(cfg.HostResolver.HostKey(), cfg.ClusterDB)
	if err != nil {
		return nil, err
	}
	p := &leaseProxy{
		ServiceConfig: cfg,
		router:        proxy.BatchFactory[Channel]{Host: cfg.HostResolver.HostKey()},
		counter:       c,
		group:         g,
	}
	p.Transport.CreateServer().BindHandler(p.handle)
	return p, nil
}

func (lp *leaseProxy) handle(ctx context.Context, msg CreateMessage) (CreateMessage, error) {
	txn := lp.ClusterDB.OpenTx()
	err := lp.create(ctx, txn, &msg.Channels)
	if err != nil {
		return CreateMessage{}, err
	}
	return CreateMessage{Channels: msg.Channels}, txn.Commit(ctx)
}

func (lp *leaseProxy) create(ctx context.Context, tx gorp.Tx, _channels *[]Channel) error {
	channels := *_channels
	for i := range channels {
		if channels[i].Leaseholder == 0 {
			channels[i].Leaseholder = lp.HostResolver.HostKey()
		}
	}
	batch := lp.router.Batch(channels)
	oChannels := make([]Channel, 0, len(channels))
	for nodeKey, entries := range batch.Peers {
		remoteChannels, err := lp.createRemote(ctx, nodeKey, entries)
		if err != nil {
			return err
		}
		oChannels = append(oChannels, remoteChannels...)
	}
	err := lp.createLocal(ctx, tx, &batch.Gateway)
	if err != nil {
		return err
	}
	oChannels = append(oChannels, batch.Gateway...)
	*_channels = oChannels
	return nil
}

func (lp *leaseProxy) createLocal(ctx context.Context, tx gorp.Tx, channels *[]Channel) error {
	// Check for name conflicts
	for _, ch := range *channels {
		var res Channel
		gorp.NewRetrieve[Key, Channel]().
			Where(func(c *Channel) bool { return c.Name == ch.Name }).
			Entry(&res).
			Exec(ctx, tx)
		if res.Name != "" {
			return errors.Wrapf(
				validate.Error,
				"channel with name %s already exists",
				ch.Name)
		}
	}
	if err := lp.assignLocalKeys(channels); err != nil {
		return err
	}
	for i, ch := range *channels {
		// Index of an index channel is itself.
		if ch.IsIndex {
			ch.LocalIndex = ch.LocalKey
			(*channels)[i] = ch
		}
	}
	// Create the channels in the storage engine
	storageChannels := toStorage(*channels)
	if err := lp.TSChannel.CreateChannel(ctx, storageChannels...); err != nil {
		return err
	}
	// Create the channels in the meta data store
	if err := gorp.NewCreate[Key, Channel]().Entries(channels).Exec(ctx, tx); err != nil {
		return err
	}
	return lp.maybeSetResources(ctx, tx, *channels)
}

func (lp *leaseProxy) assignLocalKeys(channels *[]Channel) error {
	v, err := lp.counter.Add(uint16(len(*channels)))
	if err != nil {
		return err
	}
	for i, ch := range *channels {
		ch.LocalKey = v - uint16(i)
		(*channels)[i] = ch
	}
	return nil
}

func (lp *leaseProxy) maybeSetResources(
	ctx context.Context,
	txn gorp.Tx,
	channels []Channel,
) error {
	if lp.Ontology != nil {
		w := lp.Ontology.NewWriter(txn)
		for _, ch := range channels {
			rtk := OntologyID(ch.Key())
			if err := w.DefineResource(ctx, rtk); err != nil {
				return err
			}
			if err := w.DefineRelationship(
				ctx,
				core.NodeOntologyID(ch.Leaseholder),
				ontology.ParentOf,
				rtk,
			); err != nil {
				return err
			}
			if err := w.DefineRelationship(
				ctx, group.OntologyID(lp.group.Key),
				ontology.ParentOf,
				rtk,
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func (lp *leaseProxy) createRemote(ctx context.Context, target core.NodeKey, channels []Channel) ([]Channel, error) {
	addr, err := lp.HostResolver.Resolve(target)
	if err != nil {
		return nil, err
	}
	res, err := lp.Transport.CreateClient().Send(ctx, addr, CreateMessage{Channels: channels})
	if err != nil {
		return nil, err
	}
	return res.Channels, nil
}
