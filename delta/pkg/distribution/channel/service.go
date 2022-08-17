package channel

import (
	"github.com/arya-analytics/delta/pkg/distribution/core"
	"github.com/arya-analytics/delta/pkg/storage"
	"github.com/arya-analytics/x/gorp"
)

// Service is central entity for managing channels within delta's distribution layer. It provides facilities for creating
// and retrieving channels.
type Service struct {
	clusterDB *gorp.DB
	proxy     *leaseProxy
}

func New(cluster core.Cluster, clusterDB *gorp.DB, tsDB storage.TS, transport CreateTransport) *Service {
	return &Service{clusterDB: clusterDB, proxy: newLeaseProxy(cluster, clusterDB, tsDB, transport)}
}

func (s *Service) NewCreate() Create { return newCreate(s.proxy) }

func (s *Service) NewRetrieve() Retrieve {
	return newRetrieve(s.clusterDB)
}
