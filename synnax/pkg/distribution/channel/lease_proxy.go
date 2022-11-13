package channel

import (
	"context"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/proxy"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/gorp"
)

type leaseProxy struct {
	cluster   core.Cluster
	clusterDB *gorp.DB
	tsDB      storage.TS
	client    CreateTransportClient
	server    CreateTransportServer
	router    proxy.BatchFactory[Channel]
	ontology  *ontology.Ontology
}

func newLeaseProxy(
	cluster core.Cluster,
	clusterDB *gorp.DB,
	tsDB storage.TS,
	client CreateTransportClient,
	server CreateTransportServer,
	ontology *ontology.Ontology,
) *leaseProxy {
	p := &leaseProxy{
		cluster:   cluster,
		clusterDB: clusterDB,
		tsDB:      tsDB,
		client:    client,
		server:    server,
		router:    proxy.NewBatchFactory[Channel](cluster.HostID()),
		ontology:  ontology,
	}
	p.server.BindHandler(p.handle)
	return p
}

func (lp *leaseProxy) handle(ctx context.Context, msg CreateMessage) (CreateMessage, error) {
	txn := lp.clusterDB.BeginTxn()
	err := lp.create(ctx, txn, &msg.Channels)
	if err != nil {
		return CreateMessage{}, err
	}
	return CreateMessage{Channels: msg.Channels}, txn.Commit()
}

func (lp *leaseProxy) create(ctx context.Context, txn gorp.Txn, _channels *[]Channel) error {
	channels := *_channels
	for i := range channels {
		if channels[i].NodeID == 0 {
			channels[i].NodeID = lp.cluster.HostID()
		}
	}
	batch := lp.router.Batch(channels)
	oChannels := make([]Channel, 0, len(channels))
	for nodeID, entries := range batch.Remote {
		remoteChannels, err := lp.createRemote(ctx, nodeID, entries)
		if err != nil {
			return err
		}
		oChannels = append(oChannels, remoteChannels...)
	}
	err := lp.createLocal(txn, &batch.Local)
	if err != nil {
		return err
	}
	oChannels = append(oChannels, batch.Local...)
	*_channels = oChannels
	return nil
}

func (lp *leaseProxy) createLocal(txn gorp.Txn, channels *[]Channel) error {
	storageChannels := toStorage(*channels)
	if err := lp.tsDB.CreateChannels(&storageChannels); err != nil {
		return err
	}
	for i := range storageChannels {
		(*channels)[i].StorageKey = storageChannels[i].Key
	}
	// TODO: add transaction rollback to cesium clusterDB if this fails.
	if err := gorp.NewCreate[Key, Channel]().Entries(channels).Exec(txn); err != nil {
		return err
	}
	return lp.maybeSetResources(txn, *channels)
}

func (lp *leaseProxy) maybeSetResources(
	txn gorp.Txn,
	channels []Channel,
) error {
	if lp.ontology != nil {
		w := lp.ontology.NewWriterUsingTxn(txn)
		for _, channel := range channels {
			rtk := OntologyID(channel.Key())
			if err := w.DefineResource(rtk); err != nil {
				return err
			}
			if err := w.DefineRelationship(
				core.NodeOntologyID(channel.NodeID),
				ontology.ParentOf,
				rtk,
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func (lp *leaseProxy) createRemote(ctx context.Context, target core.NodeID, channels []Channel) ([]Channel, error) {
	addr, err := lp.cluster.Resolve(target)
	if err != nil {
		return nil, err
	}
	res, err := lp.client.Send(ctx, addr, CreateMessage{Channels: channels})
	if err != nil {
		return nil, err
	}
	return res.Channels, nil
}
