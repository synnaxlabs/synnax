package iterator

import (
	"context"
	"github.com/arya-analytics/aspen"
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/distribution/proxy"
	"github.com/arya-analytics/delta/pkg/distribution/segment/core"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/confluence/plumber"
	"github.com/arya-analytics/x/errutil"
	"github.com/arya-analytics/x/signal"
	"github.com/arya-analytics/x/telem"
	"github.com/cockroachdb/errors"
	"time"
)

type Iterator interface {
	// Responses emits segment data retrieved by method calls from the Iterator.
	// The channel is closed when iterator.Close is called.
	Responses() <-chan Response
	// Next retrieves the next segment of each channel's data.
	// Returns true if the current Iterator.View is pointing to any valid segments.
	// It's important to note that if channel data is non-contiguous, calls to Next
	// may return segments that occupy different ranges of time.
	Next() bool
	// Prev retrieves the previous segment of each channel's data.
	// Returns true if the current Iterator.View is pointing to any valid segments.
	// It's important to note that if channel data is non-contiguous, calls to Prev
	// may return segments that occupy different ranges of time.
	Prev() bool
	// First returns the first segment of each channel's data.
	// Returns true if the current Iterator.View is pointing to any valid segments.
	// It's important to note that if channel data is non-contiguous, calls to First
	// may return segments that occupy different ranges of time.
	First() bool
	// Last returns the last segment of each channel's data.
	// Returns true if the current Iterator.View is pointing to any valid segments.
	// It's important to note that if channel data is non-contiguous, calls to Last
	// may return segments that occupy different ranges of time.
	Last() bool
	// NextSpan reads all channel data occupying the next span of time. Returns true
	// if the current Iterator.View is pointing to any valid segments.
	NextSpan(span telem.TimeSpan) bool
	// PrevSpan reads all channel data occupying the previous span of time. Returns true
	// if the current Iterator.View is pointing to any valid segments.
	PrevSpan(span telem.TimeSpan) bool
	// NextRange seeks the Iterator to the start of the range and reads all channel data
	// until the end of the range.
	NextRange(tr telem.TimeRange) bool
	// SeekFirst seeks the iterator the start of the iterator range.
	// Returns true if the current Iterator.View is pointing to any valid segments.
	SeekFirst() bool
	// SeekLast seeks the iterator the end of the iterator range.
	// Returns true if the current Iterator.View is pointing to any valid segments.
	SeekLast() bool
	// SeekLT seeks the iterator to the first whose timestamp is less than or equal
	// to the given timestamp. Returns true if the current Iterator.View is pointing
	// to any valid segments.
	SeekLT(t telem.TimeStamp) bool
	// SeekGE seeks the iterator to the first whose timestamp is greater than the
	// given timestamp. Returns true if the current Iterator.View is pointing to
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
	// Exhaust seeks to the first position in the Iterator and iterates through all
	// segments until the Iterator is exhausted.
	Exhaust() bool
}

func New(
	ctx context.Context,
	db cesium.DB,
	svc *channel.Service,
	resolver aspen.HostResolver,
	tran Transport,
	rng telem.TimeRange,
	keys channel.Keys,
) (Iterator, error) {
	sCtx, cancel := signal.WithCancel(ctx)

	// First we need to check if all the channels exist and are retrievable in the
	// database.
	if err := core.ValidateChannelKeys(ctx, svc, keys); err != nil {
		return nil, err
	}

	// TraverseTo we determine IDs of all the target nodes we need to open iterators on.
	batch := proxy.NewBatchFactory[channel.Key](resolver.HostID()).Batch(keys)

	var (
		pipe              = plumber.New()
		needRemote        = len(batch.Remote) > 0
		needLocal         = len(batch.Local) > 0
		numSenders        = 0
		numReceivers      = 0
		receiverAddresses []address.Address
	)

	if needRemote {
		numSenders += 1
		numReceivers += len(batch.Remote)

		sender, receivers, err := openRemoteIterators(sCtx, tran, batch.Remote, rng, resolver)
		if err != nil {
			cancel()
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
		numSenders += 1
		numReceivers += 1
		localIter, err := newLocalIterator(db, resolver.HostID(), rng, batch.Local)
		if err != nil {
			cancel()
			return nil, err
		}
		addr := address.Address("local")
		plumber.SetSegment[Request, Response](pipe, addr, localIter)
		receiverAddresses = append(receiverAddresses, addr)
	}

	// The synchronizer checks that all nodes have acknowledged an iteration
	// request. This is used to return ok = true from the iterator methods.
	sync := &synchronizer{nodeIDs: keys.UniqueNodeIDs(), timeout: 2 * time.Second}

	// Open a ackFilter that will route acknowledgement responses to the iterator
	// synchronizer. We expect an ack from each remote iterator as well as the
	// local iterator, so we set our buffer cap at numReceivers.
	syncMessages := confluence.NewStream[Response](numReceivers)
	sync.InFrom(syncMessages)

	// Send rejects from the ackFilter to the synchronizer.
	filter := newAckRouter(syncMessages)
	plumber.SetSegment[Response, Response](pipe, "filter", filter)

	// emitter emits method calls as requests to stream.
	emit := &emitter{}
	plumber.SetSource[Request](pipe, "emitter", emit)

	var (
		routeEmitterTo address.Address
		c              = errutil.NewCatchSimple()
	)

	// We need to configure different pipelines to optimize for particular cases.
	if needRemote && needLocal {
		// Open a broadcaster that will multiply requests to both the local and remote
		// iterators.
		plumber.SetSegment[Request, Request](
			pipe,
			"broadcaster",
			&confluence.DeltaMultiplier[Request]{},
		)
		routeEmitterTo = "broadcaster"

		// We use confluence.StitchWeave here to dedicate a channel to both the
		// sender and local, so that they both receive a copy of the emitted request.
		c.Exec(plumber.MultiRouter[Request]{
			SourceTargets: []address.Address{"broadcaster"},
			SinkTargets:   []address.Address{"sender", "local"},
			Capacity:      1,
			Stitch:        plumber.StitchWeave,
		}.PreRoute(pipe))
	} else if needRemote {
		// If we only have remote iterators, we can skip the broadcasting step
		// and forward requests from the emitter directly to the sender.
		routeEmitterTo = "sender"
	} else {
		// If we only have local iterators, we can skip the broadcasting step
		// and forward requests from the emitter directly to the local iterator.
		routeEmitterTo = "local"
	}

	c.Exec(plumber.UnaryRouter[Request]{
		SourceTarget: "emitter",
		SinkTarget:   routeEmitterTo,
	}.PreRoute(pipe))

	c.Exec(plumber.MultiRouter[Response]{
		SourceTargets: receiverAddresses,
		SinkTargets:   []address.Address{"filter"},
		Stitch:        plumber.StitchUnary,
		Capacity:      numReceivers,
	}.PreRoute(pipe))

	if c.Error() != nil {
		panic(c.Error())
	}

	seg := &plumber.Segment[Request, Response]{Pipeline: pipe}
	if err := seg.RouteOutletFrom("filter"); err != nil {
		panic(err)
	}

	res := confluence.NewStream[Response](numReceivers)

	seg.OutTo(res)
	seg.Flow(sCtx, confluence.CloseInletsOnExit())

	return &iterator{
		emitter:   emit,
		sync:      sync,
		wg:        sCtx,
		cancel:    cancel,
		responses: res.Outlet(),
	}, nil
}

type iterator struct {
	emitter   *emitter
	sync      *synchronizer
	cancel    context.CancelFunc
	wg        signal.WaitGroup
	_error    error
	responses <-chan Response
}

func (i *iterator) Responses() <-chan Response { return i.responses }

// Next implements Iterator.
func (i *iterator) Next() bool {
	i.emitter.next()
	return i.ack(Next)
}

// Prev implements Iterator.
func (i *iterator) Prev() bool {
	i.emitter.Prev()
	return i.ack(Prev)
}

// First implements Iterator.
func (i *iterator) First() bool {
	i.emitter.First()
	return i.ack(First)
}

// Last implements Iterator.
func (i *iterator) Last() bool {
	i.emitter.Last()
	return i.ack(Last)
}

// NextSpan implements Iterator.
func (i *iterator) NextSpan(span telem.TimeSpan) bool {
	i.emitter.NextSpan(span)
	return i.ack(NextSpan)
}

// PrevSpan implements Iterator.
func (i *iterator) PrevSpan(span telem.TimeSpan) bool {
	i.emitter.PrevSpan(span)
	return i.ack(PrevSpan)
}

// NextRange implements Iterator.
func (i *iterator) NextRange(tr telem.TimeRange) bool {
	i.emitter.NextRange(tr)
	return i.ack(NextRange)
}

// SeekFirst implements Iterator.
func (i *iterator) SeekFirst() bool {
	i.emitter.SeekFirst()
	return i.ack(SeekFirst)
}

// SeekLast implements Iterator.
func (i *iterator) SeekLast() bool {
	i.emitter.SeekLast()
	return i.ack(SeekLast)
}

// SeekLT implements Iterator.
func (i *iterator) SeekLT(stamp telem.TimeStamp) bool {
	i.emitter.SeekLT(stamp)
	return i.ack(SeekLT)
}

// SeekGE implements Iterator.
func (i *iterator) SeekGE(stamp telem.TimeStamp) bool {
	i.emitter.SeekGE(stamp)
	return i.ack(SeekGE)
}

// Exhaust implements Iterator.
func (i *iterator) Exhaust() bool {
	i.emitter.Exhaust()
	return i.ack(Exhaust)
}

// Valid implements Iterator.
func (i *iterator) Valid() bool {
	i.emitter.Valid()
	return i.ack(Valid) && i.error() == nil
}

// Error implements Iterator.
func (i *iterator) Error() error {
	if i.error() != nil {
		return i.error()
	}
	i.emitter.Error()
	if ok, err := i.ackWithErr(Error); !ok || err != nil {
		return errors.CombineErrors(err, errors.New("[iterator] - non positive ack"))
	}
	return nil
}

// Close implements Iterator.
func (i *iterator) Close() error {
	defer i.cancel()

	// Let all iterators (remote and local) know that it's time to stop.
	i.emitter.Close()

	// Wait for all nodes to acknowledge a safe closure.
	if ok := i.ack(Close); !ok {
		return errors.New("[segment.iterator] - negative ack on close. node probably unreachable")
	}

	// Prevent any further commands from being sent.
	i.emitter.CloseInlets()

	// Wait on all goroutines to exit.
	return i.wg.Wait()
}

func (i *iterator) error() error {
	if i._error != nil {
		return i._error
	}
	select {
	case <-i.wg.Stopped():
		i._error = i.wg.Wait()
		return i._error
	default:
		return nil
	}
}

func (i *iterator) ack(cmd Command) bool {
	ok, _ := i.ackWithErr(cmd)
	return ok
}

func (i *iterator) ackWithErr(cmd Command) (bool, error) {
	return i.sync.sync(context.Background(), cmd)
}
