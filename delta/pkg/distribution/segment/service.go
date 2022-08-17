package segment

import (
	"context"
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/distribution/core"
	"github.com/arya-analytics/delta/pkg/distribution/segment/iterator"
	"github.com/arya-analytics/delta/pkg/distribution/segment/writer"
	"github.com/arya-analytics/x/query"
	"github.com/arya-analytics/x/telem"
	"go.uber.org/zap"
)

type Service struct {
	channel   *channel.Service
	db        cesium.DB
	transport Transport
	resolver  core.HostResolver
	logger    *zap.Logger
}

func New(
	channel *channel.Service,
	db cesium.DB,
	transport Transport,
	resolver core.HostResolver,
	logger *zap.Logger,
) *Service {
	s := &Service{
		channel:   channel,
		db:        db,
		transport: transport,
		resolver:  resolver,
		logger:    logger,
	}
	hostID := resolver.HostID()
	iterator.NewServer(db, hostID, transport.Iterator())
	writer.NewServer(db, hostID, transport.Writer())
	return s
}

func (s *Service) NewCreate() Create { return newCreate(s) }

func (s *Service) NewRetrieve() Retrieve { return newRetrieve(s) }

type Create struct {
	query.Query
	svc *Service
}

func newCreate(svc *Service) Create {
	return Create{svc: svc, Query: query.New()}
}

func (c Create) WhereChannels(keys ...channel.Key) Create {
	setKeys(c, keys)
	return c
}

func (c Create) Write(ctx context.Context) (Writer, error) {
	return writer.New(
		ctx,
		c.svc.db,
		c.svc.channel,
		c.svc.resolver,
		c.svc.transport.Writer(),
		getKeys(c),
		c.svc.logger,
	)
}

type Retrieve struct {
	query.Query
	svc *Service
}

func newRetrieve(svc *Service) Retrieve {
	return Retrieve{svc: svc, Query: query.New()}
}

func (r Retrieve) WhereChannels(keys ...channel.Key) Retrieve {
	setKeys(r, keys)
	return r
}

func (r Retrieve) WhereTimeRange(rng telem.TimeRange) Retrieve {
	telem.SetTimeRange(r, rng)
	return r
}

func (r Retrieve) Iterate(ctx context.Context) (Iterator, error) {
	tr, err := telem.GetTimeRange(r)
	if err != nil {
		tr = telem.TimeRangeMax
	}
	return iterator.New(
		ctx,
		r.svc.db,
		r.svc.channel,
		r.svc.resolver,
		r.svc.transport.Iterator(),
		tr,
		getKeys(r),
	)
}

// |||||| KEYS ||||||

const keysKey = "keys"

func setKeys(q query.Query, keys channel.Keys) { q.Set(keysKey, keys) }

func getKeys(q query.Query) channel.Keys { return q.GetRequired(keysKey).(channel.Keys) }
