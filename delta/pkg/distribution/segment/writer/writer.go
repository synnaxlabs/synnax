package writer

import (
	"context"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	distribcore "github.com/arya-analytics/delta/pkg/distribution/core"
	"github.com/arya-analytics/delta/pkg/distribution/proxy"
	"github.com/arya-analytics/delta/pkg/distribution/segment/core"
	"github.com/arya-analytics/delta/pkg/storage"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/confluence/plumber"
	"github.com/arya-analytics/x/signal"
	"go.uber.org/zap"
)

type Writer interface {
	Requests() chan<- Request
	Responses() <-chan Response
	Close() error
}

type writer struct {
	requests  chan<- Request
	responses <-chan Response
	wg        signal.WaitGroup
}

func (w *writer) Requests() chan<- Request { return w.requests }

func (w *writer) Responses() <-chan Response { return w.responses }

func (w *writer) Close() error { return w.wg.Wait() }

func New(
	ctx context.Context,
	db storage.TS,
	svc *channel.Service,
	resolver distribcore.HostResolver,
	tran Transport,
	keys channel.Keys,
	logger *zap.Logger,
) (Writer, error) {
	sCtx, cancel := signal.WithCancel(
		ctx,
		signal.WithContextKey("writer"),
		signal.WithLogger(logger),
	)
	hostID := resolver.HostID()

	// First we need to check if all the channels exist and are retrievable in the
	//database.
	if err := core.ValidateChannelKeys(sCtx, svc, keys); err != nil {
		cancel()
		return nil, err
	}

	// TraverseTo we determine the IDs of all the target nodes we need to write to.
	batch := proxy.NewBatchFactory[channel.Key](hostID).Batch(keys)

	var (
		pipe              = plumber.New()
		needRemote        = len(batch.Remote) > 0
		needLocal         = len(batch.Local) > 0
		receiverAddresses []address.Address
	)

	transient := confluence.NewStream[error](0)

	if needRemote {
		sender, receivers, _receiverAddresses, err := openRemoteWriters(sCtx, tran, batch.Remote, resolver, transient)
		if err != nil {
			cancel()
			return nil, err
		}

		// Set up our sender as a sink for the request pipeline.
		plumber.SetSink[Request](pipe, "remoteSender", sender)
		receiverAddresses = _receiverAddresses
		for i, receiver := range receivers {
			plumber.SetSource[Response](pipe, _receiverAddresses[i], receiver)
		}
	}

	if needLocal {
		w, err := newLocalWriter(sCtx, hostID, db, batch.Local, transient)
		if err != nil {
			cancel()
			return nil, err
		}
		addr := address.Address("localWriter")
		plumber.SetSegment[Request, Response](pipe, addr, w)
		receiverAddresses = append(receiverAddresses, addr)
	}

	var routeRequestsTo address.Address

	if needRemote && needLocal {
		rls := newRemoteLocalSwitch(hostID)
		plumber.SetSegment[Request, Request](pipe, "remoteLocalSwitch", rls)
		routeRequestsTo = "remoteLocalSwitch"

		if err := (plumber.MultiRouter[Request]{
			SourceTargets: []address.Address{"remoteLocalSwitch"},
			SinkTargets:   []address.Address{"remoteSender", "localWriter"},
			Stitch:        plumber.StitchWeave,
		}).Route(pipe); err != nil {
			panic(err)
		}
	} else if needRemote {
		routeRequestsTo = "remoteSender"
	} else {
		routeRequestsTo = "localWriter"
	}

	receiverAddresses = append(receiverAddresses, "transient")
	plumber.SetSource[Response](pipe, "transient", &TransientSource{transient: transient})

	seg := &plumber.Segment[Request, Response]{Pipeline: pipe}
	if err := seg.RouteInletTo(routeRequestsTo); err != nil {
		panic(err)
	}
	if err := seg.RouteOutletFrom(receiverAddresses...); err != nil {
		panic(err)
	}

	input := confluence.NewStream[Request](1)
	output := confluence.NewStream[Response](1)
	seg.InFrom(input)
	seg.OutTo(output)

	seg.Flow(sCtx, confluence.CloseInletsOnExit(), confluence.CancelOnExitErr())

	return &writer{responses: output.Outlet(), requests: input.Inlet(), wg: sCtx}, nil
}
