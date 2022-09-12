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
	transport CreateTransport
	router    proxy.BatchFactory[Channel]
	resources *ontology.Ontology
}

func newLeaseProxy(cluster core.Cluster, clusterDB *gorp.DB, tsDB storage.TS, transport CreateTransport) *leaseProxy {
	p := &leaseProxy{
		cluster:   cluster,
		clusterDB: clusterDB,
		tsDB:      tsDB,
		transport: transport,
		router:    proxy.NewBatchFactory[Channel](cluster.HostID()),
	}
	p.transport.BindHandler(p.handle)
	return p
}

func (lp *leaseProxy) handle(ctx context.Context, msg CreateMessage) (CreateMessage, error) {
	txn := lp.clusterDB.BeginTxn()
	channels, err := lp.create(ctx, txn, msg.Channels)
	if err != nil {
		return CreateMessage{}, err
	}
	return CreateMessage{Channels: channels}, txn.Commit()
}

func (lp *leaseProxy) create(ctx context.Context, txn gorp.Txn, channels []Channel) ([]Channel, error) {
	batch := lp.router.Batch(channels)
	oChannels := make([]Channel, 0, len(channels))
	for nodeID, entries := range batch.Remote {
		remoteChannels, err := lp.createRemote(ctx, nodeID, entries)
		if err != nil {
			return nil, err
		}
		oChannels = append(oChannels, remoteChannels...)
	}
	ch, err := lp.createLocal(txn, batch.Local)
	if err != nil {
		return oChannels, err
	}
	oChannels = append(oChannels, ch...)
	return oChannels, nil
}

func (lp *leaseProxy) createLocal(txn gorp.Txn, channels []Channel) ([]Channel, error) {
	for i := range channels {
		channels[i].Channel.Density = channels[i].DataType.Density()
		if err := lp.tsDB.CreateChannel(&channels[i].Channel); err != nil {
			return nil, err
		}
	}
	// TODO: add transaction rollback to cesium clusterDB if this fails.
	if err := gorp.NewCreate[Key, Channel]().Entries(&channels).Exec(txn); err != nil {
		return nil, err
	}
	return channels, lp.maybeSetResources(txn, channels)
}

func (lp *leaseProxy) maybeSetResources(
	txn gorp.Txn,
	channels []Channel,
) error {
	if lp.resources != nil {
		w := lp.resources.NewWriterUsingTxn(txn)
		for _, channel := range channels {
			rtk := OntologyID(channel.Key())
			if err := w.DefineResource(rtk); err != nil {
				return err
			}
			if err := w.DefineRelationship(
				core.NodeOntologyID(channel.NodeID),
				rtk,
				ontology.Parent,
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
	res, err := lp.transport.Send(ctx, addr, CreateMessage{Channels: channels})
	if err != nil {
		return nil, err
	}
	return res.Channels, nil
}
