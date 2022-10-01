package iterator

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/samber/lo"
	"github.com/synnaxlabs/aspen"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/proxy"
	"github.com/synnaxlabs/synnax/pkg/distribution/segment/core"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

type StreamIterator = confluence.Segment[Request, Response]

type Iterator interface {
	// Next retrieves the next segment of each channel's data.
	// Returns true if the current IteratorServer.View is pointing to any valid segments.
	// It's important to note that if channel data is non-contiguous, calls to Next
	// may return segments that occupy different ranges of time.
	Next() bool
	// Prev retrieves the previous segment of each channel's data.
	// Returns true if the current IteratorServer.View is pointing to any valid segments.
	// It's important to note that if channel data is non-contiguous, calls to Prev
	// may return segments that occupy different ranges of time.
	Prev() bool
	// First returns the first segment of each channel's data.
	// Returns true if the current IteratorServer.View is pointing to any valid segments.
	// It's important to note that if channel data is non-contiguous, calls to First
	// may return segments that occupy different ranges of time.
	First() bool
	// Last returns the last segment of each channel's data.
	// Returns true if the current IteratorServer.View is pointing to any valid segments.
	// It's important to note that if channel data is non-contiguous, calls to Last
	// may return segments that occupy different ranges of time.
	Last() bool
	// NextSpan reads all channel data occupying the next span of time. Returns true
	// if the current IteratorServer.View is pointing to any valid segments.
	NextSpan(span telem.TimeSpan) bool
	// PrevSpan reads all channel data occupying the previous span of time. Returns true
	// if the current IteratorServer.View is pointing to any valid segments.
	PrevSpan(span telem.TimeSpan) bool
	// Range seeks the Iterator to the start of the range and reads all channel data
	// until the end of the range.
	Range(tr telem.TimeRange) bool
	// SeekFirst seeks the iterator the start of the iterator range.
	// Returns true if the current IteratorServer.View is pointing to any valid segments.
	SeekFirst() bool
	// SeekLast seeks the iterator the end of the iterator range.
	// Returns true if the current IteratorServer.View is pointing to any valid segments.
	SeekLast() bool
	// SeekLT seeks the iterator to the first whose timestamp is less than or equal
	// to the given timestamp. Returns true if the current IteratorServer.View is pointing
	// to any valid segments.
	SeekLT(t telem.TimeStamp) bool
	// SeekGE seeks the iterator to the first whose timestamp is greater than the
	// given timestamp. Returns true if the current IteratorServer.View is pointing to
	// any valid segments.
	SeekGE(t telem.TimeStamp) bool
	// Close closes the Iterator, ensuring that all in-progress reads complete
	// before closing the Source outlet. All iterators must be Closed, or the
	// distribution layer will panic.
	Close() error
	// Valid returns true if the iterator is pointing at valid data and is error free.
	Valid() bool
	// Error returns any errors accumulated during the iterators lifetime.
	Error() error
	Value() []core.Segment
}

type Config struct {
	TimeRange       telem.TimeRange
	ChannelKeys     channel.Keys
	TS              storage.TS
	ChannelService  *channel.Service
	Resolver        aspen.HostResolver
	TransportServer TransportServer
	TransportClient TransportClient
	Logger          *zap.Logger
}

var _ config.Config[Config] = Config{}

func (cfg Config) Override(other Config) Config {
	cfg.TimeRange.Start = override.Numeric(cfg.TimeRange.Start, other.TimeRange.Start)
	cfg.TimeRange.End = override.Numeric(cfg.TimeRange.End, other.TimeRange.End)
	cfg.ChannelKeys = override.Slice(cfg.ChannelKeys, other.ChannelKeys)
	cfg.TS = override.Nil(cfg.TS, other.TS)
	cfg.ChannelService = override.Nil(cfg.ChannelService, other.ChannelService)
	cfg.TransportServer = override.Nil(cfg.TransportServer, other.TransportServer)
	cfg.TransportClient = override.Nil(cfg.TransportClient, other.TransportClient)
	cfg.Resolver = override.Nil(cfg.Resolver, other.Resolver)
	cfg.Logger = override.Nil(cfg.Logger, other.Logger)
	return cfg
}

func (cfg Config) Validate() error {
	v := validate.New("iterator")
	if cfg.TimeRange.IsZero() {
		return errors.New("[iterator] no range provided")
	}
	validate.NotNil(v, "TS", cfg.TS)
	validate.NotNil(v, "channel", cfg.ChannelService)
	validate.NotNil(v, "transportServer", cfg.TransportServer)
	validate.NotNil(v, "transportClient", cfg.TransportClient)
	validate.NotNil(v, "resolver", cfg.Resolver)
	validate.NotNil(v, "logger", cfg.Logger)
	return v.Error()
}

func NewStream(ctx context.Context, _cfg ...Config) (StreamIterator, error) {
	cfg, err := config.OverrideAndValidate(Config{}, _cfg...)
	if err != nil {
		return nil, err
	}

	// First we need to check if all the channels exist and are retrievable in the
	// database.
	if err := core.ValidateChannelKeys(ctx, cfg.ChannelService, cfg.ChannelKeys); err != nil {
		return nil, err
	}

	// Determine IDs of all the target nodes we need to open iterators on.
	batch := proxy.NewBatchFactory[channel.Key](cfg.Resolver.HostID()).Batch(cfg.ChannelKeys)

	var (
		pipe              = plumber.New()
		needRemote        = len(batch.Remote) > 0
		needLocal         = len(batch.Local) > 0
		receiverAddresses []address.Address
	)

	if needRemote {
		sender, receivers, err := openRemoteIterators(ctx, batch.Remote, cfg)
		if err != nil {
			return nil, err
		}

		// Set up our sender as a sink for the request pipeline.
		plumber.SetSink[Request](pipe, "sender", sender)

		// Set up our remote receivers as sources for the response pipeline.
		receiverAddresses = make([]address.Address, len(receivers))
		for i, c := range receivers {
			addr := address.Newf("client-%v", i+1)
			receiverAddresses[i] = addr
			plumber.SetSource[Response](pipe, addr, c)
		}
	}

	if needLocal {
		localIter, err := newLocalIterator(batch.Local, cfg)
		if err != nil {
			return nil, err
		}
		addr := address.Address("local")
		plumber.SetSegment[Request, Response](pipe, addr, localIter)
		receiverAddresses = append(receiverAddresses, addr)
	}

	plumber.SetSegment[Response, Response](pipe, "filter", newAckFilter())

	// The synchronizer checks that all nodes have acknowledged an iteration
	// request. This is used to return ok = true from the iterator methods.
	plumber.SetSegment[Response, Response](
		pipe,
		"synchronizer",
		newSynchronizer(cfg.ChannelKeys.UniqueNodeIDs()),
	)

	var routeInletTo address.Address

	// We need to configure different pipelines to optimize for particular cases.
	if needRemote && needLocal {
		// Open a broadcaster that will multiply requests to both the local and remote
		// iterators.
		plumber.SetSegment[Request, Request](
			pipe,
			"broadcaster",
			&confluence.DeltaMultiplier[Request]{},
		)
		routeInletTo = "broadcaster"

		// We use confluence.StitchWeave here to dedicate a channel to both the
		// sender and local, so that they both receive a copy of the emitted request.
		plumber.MultiRouter[Request]{
			SourceTargets: []address.Address{"broadcaster"},
			SinkTargets:   []address.Address{"sender", "local"},
			Stitch:        plumber.StitchWeave,
		}.MustRoute(pipe)
	} else if needRemote {
		// If we only have remote iterators, we can skip the broadcasting step
		// and forward requests from the emitter directly to the sender.
		routeInletTo = "sender"
	} else {
		// If we only have local iterators, we can skip the broadcasting step
		// and forward requests from the emitter directly to the local iterator.
		routeInletTo = "local"
	}

	plumber.MultiRouter[Response]{
		SourceTargets: receiverAddresses,
		SinkTargets:   []address.Address{"filter"},
		Stitch:        plumber.StitchUnary,
		Capacity:      len(receiverAddresses),
	}.MustRoute(pipe)

	plumber.UnaryRouter[Response]{
		SourceTarget: "filter",
		SinkTarget:   "synchronizer",
	}.MustRoute(pipe)

	seg := &plumber.Segment[Request, Response]{Pipeline: pipe}
	lo.Must0(seg.RouteOutletFrom("filter", "synchronizer"))
	lo.Must0(seg.RouteInletTo(routeInletTo))

	return seg, nil
}

func New(ctx context.Context, _cfg ...Config) (Iterator, error) {
	cfg, err := config.OverrideAndValidate(Config{}, _cfg...)
	if err != nil {
		return nil, err
	}
	stream, err := NewStream(ctx, cfg)
	if err != nil {
		return nil, err
	}
	sCtx, cancel := signal.Background(
		signal.WithLogger(cfg.Logger),
		signal.WithContextKey("iterator"),
	)
	requests := confluence.NewStream[Request]()
	responses := confluence.NewStream[Response]()
	stream.InFrom(requests)
	stream.OutTo(responses)
	stream.Flow(
		sCtx,
		confluence.CloseInletsOnExit(),
		confluence.CancelOnExitErr(),
	)
	return &iterator{
		requests:  requests,
		responses: responses,
		internal:  stream,
		shutdown:  cancel,
		wg:        sCtx,
	}, nil
}

type iterator struct {
	requests  confluence.Inlet[Request]
	responses confluence.Outlet[Response]
	internal  StreamIterator
	shutdown  context.CancelFunc
	wg        signal.WaitGroup
	value     []Response
}

// Next implements Iterator.
func (i *iterator) Next() bool {
	i.value = nil
	return i.exec(Request{Command: Next})
}

// Prev implements Iterator.
func (i *iterator) Prev() bool {
	i.value = nil
	return i.exec(Request{Command: Prev})
}

// First implements Iterator.
func (i *iterator) First() bool {
	i.value = nil
	return i.exec(Request{Command: First})
}

// Last implements Iterator.
func (i *iterator) Last() bool {
	i.value = nil
	return i.exec(Request{Command: Last})
}

// NextSpan implements Iterator.
func (i *iterator) NextSpan(span telem.TimeSpan) bool {
	i.value = nil
	return i.exec(Request{Command: NextSpan, Span: span})
}

// PrevSpan implements Iterator.
func (i *iterator) PrevSpan(span telem.TimeSpan) bool {
	i.value = nil
	return i.exec(Request{Command: PrevSpan, Span: span})
}

// Range implements Iterator.
func (i *iterator) Range(tr telem.TimeRange) bool {
	i.value = nil
	return i.exec(Request{Command: NextRange, Range: tr})
}

// SeekFirst implements Iterator.
func (i *iterator) SeekFirst() bool {
	i.value = nil
	return i.exec(Request{Command: SeekFirst})
}

// SeekLast implements Iterator.
func (i *iterator) SeekLast() bool {
	i.value = nil
	return i.exec(Request{Command: SeekLast})
}

// SeekLT implements Iterator.
func (i *iterator) SeekLT(stamp telem.TimeStamp) bool {
	i.value = nil
	return i.exec(Request{Command: SeekLT, Stamp: stamp})
}

// SeekGE implements Iterator.
func (i *iterator) SeekGE(stamp telem.TimeStamp) bool {
	i.value = nil
	return i.exec(Request{Command: SeekGE, Stamp: stamp})
}

// Valid implements Iterator.
func (i *iterator) Valid() bool {
	return i.exec(Request{Command: Valid})
}

// Error implements Iterator.
func (i *iterator) Error() error {
	_, err := i.execErr(Request{Command: Error})
	return err
}

// Close implements Iterator.
func (i *iterator) Close() error {
	defer i.shutdown()
	i.requests.Close()
	return i.wg.Wait()
}

func (i *iterator) Value() []core.Segment {
	var segments []core.Segment
	for _, resp := range i.value {
		segments = append(segments, resp.Segments...)
	}
	return segments
}

func (i *iterator) exec(req Request) bool {
	ok, _ := i.execErr(req)
	return ok
}

func (i *iterator) execErr(req Request) (bool, error) {
	i.requests.Inlet() <- req
	for res := range i.responses.Outlet() {
		if res.Variant == AckResponse {
			return res.Ack, res.Error
		}
		i.value = append(i.value, res)
	}
	return false, nil
}
