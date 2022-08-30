package retrieve

import "github.com/arya-analytics/delta/pkg/distribution/segment"

type Service struct {
	internal *segment.Service
}

func NewRetrieve[V Value, F Format[V]](s *Service) Retrieve[V, F] {

}
