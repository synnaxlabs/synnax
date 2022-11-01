package channel

import (
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

func (s *Service) NewCreate() Create { return newCreate(s.proxy) }

func (s *Service) NewRetrieve() Retrieve {
	return newRetrieve(s.clusterDB)
}
