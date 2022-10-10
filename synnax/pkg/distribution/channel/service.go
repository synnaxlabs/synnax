package channel

import (
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
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
) *Service {
	return &Service{clusterDB: clusterDB, proxy: newLeaseProxy(cluster, clusterDB, tsDB, client, server)}
}

func (s *Service) NewCreate() Create { return newCreate(s.proxy) }

func (s *Service) NewRetrieve() Retrieve {
	return newRetrieve(s.clusterDB)
}
