package stream

import (
	"context"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	distribcore "github.com/arya-analytics/delta/pkg/distribution/core"
	"github.com/arya-analytics/freighter"
	"github.com/arya-analytics/freighter/freightfluence"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/signal"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"sync"
)

type demandCoordinator struct {
	mu             sync.Mutex
	demands        map[address.Address][]channel.Key
	prevOperations []remoteReadOperation
	confluence.AbstractUnarySource[[]remoteReadOperation]
}

func newDemandCoordinator() *demandCoordinator {
	return &demandCoordinator{
		demands: make(map[address.Address][]channel.Key),
	}
}

type remoteReadOperation struct {
	close  bool
	nodeID distribcore.NodeID
	keys   []channel.Key
}

func (r remoteReadOperation) next(nextState map[distribcore.NodeID]channel.Keys) (op remoteReadOperation, diff bool) {
	keys, ok := nextState[r.nodeID]
	if !ok {
		return remoteReadOperation{close: true}, true
	}
	defer delete(nextState, r.nodeID)
	if len(keys) != len(r.keys) {
		return remoteReadOperation{nodeID: r.nodeID, keys: keys}, true
	}
	for _, key := range keys {
		if !lo.Contains(r.keys, key) {
			return remoteReadOperation{nodeID: r.nodeID, keys: keys}, true
		}
	}
	return op, false
}

func (d *demandCoordinator) set(addr address.Address, keys []channel.Key) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.demands[addr] = keys
	ops := d.parseOps()
	logrus.Info(ops)
	d.Out.Inlet() <- d.parseOps()
}

func (d *demandCoordinator) clear(addr address.Address) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.demands, addr)
	ops := d.parseOps()
	logrus.Info(ops)
	d.Out.Inlet() <- d.parseOps()
}

func (d *demandCoordinator) parseOps() []remoteReadOperation {
	var (
		nextState = d.buildNextState()
		nextOps   = make([]remoteReadOperation, 0, len(nextState))
	)

	for _, prevOp := range d.prevOperations {
		if nextOp, ok := prevOp.next(nextState); ok {
			nextOps = append(nextOps, nextOp)
		}
	}

	for nodeID, keys := range nextState {
		nextOps = append(nextOps, remoteReadOperation{nodeID: nodeID, keys: keys})
	}
	d.prevOperations = nextOps
	return nextOps
}

func (d *demandCoordinator) buildNextState() map[distribcore.NodeID]channel.Keys {
	nextState := make(map[distribcore.NodeID]channel.Keys)
	for _, keys := range d.demands {
		for _, key := range keys {
			nodeID := key.NodeID()
			nextState[nodeID] = append(nextState[nodeID], key)
		}
	}
	return nextState
}

// remoteReadCoordinator coordinates demands for remoted reads based on instruction
// sets issued by the readDemandController. This involves opening up new read streams
// and closing old ones.
type remoteReadCoordinator struct {
	confluence.UnarySink[[]remoteReadOperation]
	confluence.AbstractUnarySource[freighter.StreamReceiver[ReadResponse]]
	confluence.TransientProvider
	transport ReadTransport
	senders   map[distribcore.NodeID]freighter.StreamSenderCloser[ReadRequest]
	resolver  distribcore.HostResolver
}

func newRemoteReadCoordinator(
	transport ReadTransport,
	resolver distribcore.HostResolver,
	transient confluence.Inlet[error],
) confluence.Segment[[]remoteReadOperation, freighter.StreamReceiver[ReadResponse]] {
	rc := &remoteReadCoordinator{
		transport: transport,
		resolver:  resolver,
		senders:   make(map[distribcore.NodeID]freighter.StreamSenderCloser[ReadRequest]),
	}
	rc.UnarySink.Sink = rc.processOp
	return confluence.InjectTransient[
		[]remoteReadOperation,
		freighter.StreamReceiver[ReadResponse],
	](transient, rc)
}

func (r *remoteReadCoordinator) processOp(ctx context.Context, ops []remoteReadOperation) error {
	for _, op := range ops {
		if op.nodeID == r.resolver.HostID() {
			return nil
		}
		if op.close {
			return r.closeSender(op.nodeID)
		}
		sender, err := r.getSender(ctx, op.nodeID)
		if err != nil {
			r.Transient() <- err
			return nil
		}
		if err := sender.Send(ReadRequest{Keys: op.keys}); err != nil {
			r.Transient() <- err
		}
	}
	return nil
}

func (r *remoteReadCoordinator) getSender(
	ctx context.Context,
	target distribcore.NodeID,
) (freighter.StreamSenderCloser[ReadRequest], error) {
	stream, ok := r.senders[target]
	if ok {
		return stream, nil
	}
	return r.openStream(ctx, target)
}

func (r *remoteReadCoordinator) openStream(
	ctx context.Context,
	target distribcore.NodeID,
) (freighter.ClientStream[ReadRequest, ReadResponse], error) {
	addr, err := r.resolver.Resolve(target)
	if err != nil {
		return nil, err
	}
	stream, err := r.transport.Stream(ctx, addr)
	if err != nil {
		return nil, err
	}
	r.Out.Inlet() <- stream
	r.senders[target] = stream
	return stream, nil
}

func (r *remoteReadCoordinator) closeSender(nodeID distribcore.NodeID) error {
	err := r.senders[nodeID].CloseSend()
	delete(r.senders, nodeID)
	return err
}

// readReceiverAggregator aggregates received samples from remote nodes into one, coherent
// stream. It receives 'receivers' from the remoteReadCoordinator and starts receiving
// values from them.
type readReceiverAggregator struct {
	confluence.UnarySink[freighter.StreamReceiver[ReadResponse]]
	confluence.AbstractUnarySource[[]Sample]
}

func (r *readReceiverAggregator) Flow(sCtx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	// We explicitly ignore attaching our output as a closable here, as we don't send
	// any values to it. Instead, we place the responsibility of acquiring and closing
	// the outlets on each read receiver.
	sCtx.Go(func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case stream := <-r.In.Outlet():
				rcv := newReadReceiver(stream)
				rcv.OutTo(r.Out)
				rcv.Flow(sCtx, opts...)
			}
		}
	}, o.Signal...)
}

type readReceiver struct {
	freightfluence.TransformReceiver[[]Sample, ReadResponse]
}

func newReadReceiver(stream freighter.StreamReceiver[ReadResponse]) *readReceiver {
	rcv := &readReceiver{}
	rcv.Receiver = stream
	rcv.Transform = rcv.transform
	return rcv
}

func (r *readReceiver) transform(ctx context.Context, resp ReadResponse) ([]Sample, bool, error) {
	return resp.Samples, true, nil
}
