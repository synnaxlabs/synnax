package stream

import (
	"context"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/ioutil"
	"github.com/synnaxlabs/x/signal"
	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
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
	ctx      signal.Context
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
		delta:    confluence.NewDynamicDeltaMultiplier[[]Sample](),
		shutdown: cancel,
		ctx:      sCtx,
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

	newServer(cfg, svc.write, svc.delta, svc.demand)

	writeHostSwitch := newHostSwitch(cfg.Resolver.HostID())

	remoteWrites := confluence.NewStream[[]Sample](1)
	remoteWrites.SetInletAddress("remote")
	localWrites := confluence.NewStream[[]Sample](1)
	localWrites.SetInletAddress("local")
	writeHostSwitch.InFrom(writes)
	writeHostSwitch.OutTo(remoteWrites)
	writeHostSwitch.OutTo(localWrites)
	receiver.OutTo(localWrites)
	svc.delta.InFrom(localWrites)

	_writeSender := newWriteSender(
		cfg.Transport.Writer(),
		cfg.Resolver,
		transient,
	)
	_writeSender.InFrom(remoteWrites)

	_remoteReadCoordinator.Flow(
		sCtx,
		confluence.CloseInletsOnExit(),
		confluence.WithAddress("remoteReadCoordinator"),
	)
	receiver.Flow(
		sCtx,
		confluence.CloseInletsOnExit(),
		confluence.WithAddress("receiver"),
	)
	writeHostSwitch.Flow(
		sCtx,
		confluence.CloseInletsOnExit(),
		confluence.WithAddress("writeHostSwitch"),
	)
	_writeSender.Flow(
		sCtx,
		confluence.CloseInletsOnExit(),
		confluence.WithAddress("writeSender"),
	)
	svc.delta.Flow(
		sCtx,
		confluence.CloseInletsOnExit(),
		confluence.WithAddress("delta"),
	)

	go func() {
		for err := range transient.Outlet() {
			logrus.Warn(err)
		}
	}()

	return svc
}

func (s *Service) NewStreamReader(demands ...channel.Key) (Outlet, io.Closer) {
	stream := confluence.NewStream[[]Sample](10)
	stream.SetInletAddress(address.Address(uuid.New().String()))
	s.delta.Connect(stream)
	s.demand.set(stream.InletAddress(), demands)
	return stream, ioutil.CloserFunc(func() error {
		s.delta.Disconnect(stream)
		s.demand.clear(stream.InletAddress())
		stream.Close()
		return nil
	})
}

func (s *Service) NewFilteredStreamReader(keys ...channel.Key) (Outlet, io.Closer) {
	var (
		unfiltered = confluence.NewStream[[]Sample](10)
		filtered   = confluence.NewStream[[]Sample](10)
		filter     = newSampleFilter(keys)
	)
	filter.InFrom(unfiltered)
	filter.OutTo(filtered)

	unfiltered.SetInletAddress(address.Address(uuid.New().String()))
	s.delta.Connect(unfiltered)

	sCtx, cancel := signal.WithCancel(s.ctx)
	filter.Flow(sCtx, confluence.CloseInletsOnExit())

	s.demand.set(unfiltered.InletAddress(), keys)
	return unfiltered, ioutil.CloserFunc(func() error {
		s.delta.Disconnect(unfiltered)
		cancel()
		s.demand.clear(unfiltered.InletAddress())
		return nil
	})
}

func (s *Service) NewStreamWriter() Inlet { return s.write }

func (s *Service) Close() error {
	s.shutdown()
	if err := s.ctx.Wait(); !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}
