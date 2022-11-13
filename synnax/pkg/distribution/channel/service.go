package channel

import (
	"context"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/gorp"
)

// Service is central entity for managing channels within delta's distribution layer. It provides facilities for creating
// and retrieving channels.
type Service struct {
	clusterDB *gorp.DB
	proxy     *leaseProxy
}

func New(
	cluster core.Cluster,
	clusterDB *gorp.DB,
	tsDB storage.TS,
	client CreateTransportClient,
	server CreateTransportServer,
	ontology *ontology.Ontology,
) *Service {
	return &Service{clusterDB: clusterDB, proxy: newLeaseProxy(cluster, clusterDB, tsDB, client, server, ontology)}
}

func (s *Service) Create(channel *Channel) error {
	return s.CreateWithTxn(s.clusterDB, channel)
}

func (s *Service) CreateMany(channels *[]Channel) error {
	return s.CreateManyWithTxn(s.clusterDB, channels)
}

func (s *Service) CreateWithTxn(txn gorp.Txn, ch *Channel) error {
	channels := []Channel{*ch}
	err := s.proxy.create(context.TODO(), txn, &channels)
	if err != nil {
		return err
	}
	*ch = channels[0]
	return nil
}

func (s *Service) CreateManyWithTxn(txn gorp.Txn, channels *[]Channel) error {
	return s.proxy.create(context.TODO(), txn, channels)
}

func (s *Service) NewRetrieve() Retrieve { return newRetrieve(s.clusterDB) }
