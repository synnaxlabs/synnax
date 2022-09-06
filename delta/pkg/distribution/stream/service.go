package stream

import (
	"context"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/freighter"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/ioutil"
	"github.com/arya-analytics/x/signal"
	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"io"
)

type (
	Inlet  = confluence.Inlet[[]Sample]
	Outlet = confluence.Outlet[[]Sample]
)

type Service struct {
	demand   *demandCoordinator
	write    Inlet
	delta    *confluence.DynamicDeltaMultiplier[[]Sample]
	wg       signal.WaitGroup
	shutdown context.CancelFunc
}

func Open(cfg Config) *Service {
	sCtx, cancel := signal.Background(
		signal.WithContextKey("stream"),
		signal.WithLogger(cfg.Logger),
	)

	writes := confluence.NewStream[[]Sample](1)
	transient := confluence.NewStream[error](1)

	svc := &Service{
		demand:   newDemandCoordinator(),
		write:    writes,
		delta:    &confluence.DynamicDeltaMultiplier[[]Sample]{},
		shutdown: cancel,
		wg:       sCtx,
	}

	_remoteReadCoordinator := newRemoteReadCoordinator(
		cfg.Transport.Reader(),
		cfg.Resolver,
		transient,
	)

	ops := confluence.NewStream[[]remoteReadOperation](1)

	receivers := confluence.NewStream[freighter.StreamReceiver[ReadResponse]](1)
	_remoteReadCoordinator.OutTo(receivers)
	_remoteReadCoordinator.InFrom(ops)
	svc.demand.OutTo(ops)

	receiver := &readReceiverAggregator{}
	receiver.InFrom(receivers)

	newServer(cfg, svc.write, svc.delta)

	writeHostSwitch := newHostSwitch(cfg.Resolver.HostID())

	remoteWrites := confluence.NewStream[[]Sample](1)
	remoteWrites.SetInletAddress("remote")
	localWrites := confluence.NewStream[[]Sample](1)
	localWrites.SetInletAddress("local")
	writeHostSwitch.InFrom(writes)
	writeHostSwitch.OutTo(remoteWrites)
	writeHostSwitch.OutTo(localWrites)
	svc.delta.InFrom(localWrites)

	_writeSender := newWriteSender(
		cfg.Transport.Writer(),
		cfg.Resolver,
		transient,
	)
	_writeSender.InFrom(remoteWrites)

	_remoteReadCoordinator.Flow(sCtx, confluence.CloseInletsOnExit())
	receiver.Flow(sCtx, confluence.CloseInletsOnExit())
	writeHostSwitch.Flow(sCtx, confluence.CloseInletsOnExit())
	_writeSender.Flow(sCtx, confluence.CloseInletsOnExit())
	svc.delta.Flow(sCtx, confluence.CloseInletsOnExit())

	return svc
}

func (s *Service) NewStreamReader(keys ...channel.Key) (Outlet, io.Closer) {
	stream := confluence.NewStream[[]Sample](10)
	stream.SetInletAddress(address.Address(uuid.New().String()))
	s.delta.OutTo(stream)
	s.demand.set(stream.InletAddress(), keys)
	return stream, ioutil.CloserFunc(func() error {
		s.delta.Disconnect(stream)
		s.demand.clear(stream.InletAddress())
		return nil
	})
}

func (s *Service) NewStreamWriter() Inlet { return s.write }

func (s *Service) Close() error {
	s.shutdown()
	if err := s.wg.Wait(); !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}
