// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mqtt

import (
	"context"
	"crypto/tls"
	"maps"
	"net"
	"net/url"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt/sparkplug"
	"github.com/synnaxlabs/x/breaker"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
	"golang.org/x/net/proxy"
)

const (
	// operationTimeout bounds one connect, subscribe, unsubscribe, or publish.
	operationTimeout = 10 * time.Second
	// maxReconnectInterval caps the backoff between connection attempts.
	maxReconnectInterval = 30 * time.Second
	// disconnectQuiesce is the time a client gets to finish its work on disconnect.
	disconnectQuiesce = 250 * time.Millisecond
	// stateTimeout bounds the offline STATE message of a connection that stops.
	stateTimeout = time.Second
)

var (
	// errNotConnected is a temporary error: the connection connects again on its own.
	errNotConnected = errors.Mark(
		errors.New("broker is not connected"), driver.ErrTemporary,
	)
	// errBrokerTimeout is the temporary error of a broker that does not answer an
	// operation in time.
	errBrokerTimeout = errors.Mark(
		errors.New("timed out waiting for the broker"), driver.ErrTemporary,
	)
)

// message is one message received from a broker.
type message struct {
	// sparkplug is the decoded form of a Sparkplug B message. Such a message has no
	// topic and no payload.
	sparkplug *sparkplug.Event
	topic     string
	payload   []byte
	// received is the time the message handler saw the message, after the time of
	// every earlier message of the connection.
	received telem.TimeStamp
	retained bool
	// session is the connect of the client that the message arrived in. It rises with
	// each connect.
	session uint64
}

// subscription is one topic filter on a connection and the attachments that use it.
type subscription struct {
	attachments set.Set[*attachment]
	// refused is the error of a broker that refused the filter, or nil.
	refused error
	qos     byte
}

// connection is the one MQTT client that the tasks of a broker device share. It
// connects again after a loss, subscribes again after each connect, and gives each
// attachment its own bounded queue. Safe for concurrent use.
type connection struct {
	// rebirths wakes the connect loop to send the rebirth requests of the host session.
	rebirths chan struct{}
	ins      alamos.Instrumentation
	// clock stamps arrivals in rising order. Only the delivery goroutine uses it.
	clock telem.MonoClock
	// subscribing orders each change to mu.subs with the broker call that carries it,
	// so the broker sees the changes in the same order. It never waits on lifecycle.
	subscribing sync.Mutex
	// lifecycle serializes configure, attach, and close, which wait on the broker.
	// The pool lock never waits on a broker.
	lifecycle struct {
		sync.Mutex
		// closed is true once the last attachment left. A closed connection never
		// takes a new attachment.
		closed bool
	}
	mu struct {
		// host is the Sparkplug B session state of the edge nodes in nodes.
		host *sparkplug.Host
		// nodes holds the attachments that follow each edge node.
		nodes map[sparkplug.NodeID]set.Set[*attachment]
		// client is nil while the connection is down.
		client paho.Client
		// session counts the connects of the client. Each connect starts a Sparkplug B
		// session that needs a new birth from each edge node.
		session uint64
		// stop ends the connect loop and waits until its client has left the broker.
		stop func()
		// settled closes when the first connection attempt of the loop ends.
		settled chan struct{}
		// subs holds every subscription by its topic filter.
		subs        map[string]*subscription
		attachments set.Set[*attachment]
		cfg         clientConfig
		// configErr is the error of the last read of the properties of the broker
		// device. Nil when the connection took them.
		configErr error
		sync.Mutex
	}
}

func newConnection(
	ins alamos.Instrumentation,
	rebirthInterval time.Duration,
) *connection {
	c := &connection{ins: ins, rebirths: make(chan struct{}, 1)}
	c.mu.host = sparkplug.NewHost(rebirthInterval)
	c.mu.nodes = make(map[sparkplug.NodeID]set.Set[*attachment])
	c.mu.subs = make(map[string]*subscription)
	c.mu.attachments = make(set.Set[*attachment])
	return c
}

// configure points the connection at the broker of cfg. It does nothing when cfg is
// the current configuration. Otherwise it starts a new connect loop, and the new
// client takes over every subscription.
func (c *connection) configure(cfg clientConfig) {
	c.mu.Lock()
	if c.mu.stop != nil && c.mu.cfg.equal(cfg) {
		c.mu.Unlock()
		return
	}
	stop := c.mu.stop
	c.mu.cfg = cfg
	c.mu.Unlock()
	if stop != nil {
		// The old client must leave first: a broker allows one session per client ID.
		stop()
	}
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(c.ins))
	settled := make(chan struct{})
	c.mu.Lock()
	c.mu.stop = func() {
		cancel()
		if err := sCtx.Wait(); err != nil && !errors.Is(err, context.Canceled) {
			c.ins.L.Error("the connection loop failed", zap.Error(err))
		}
	}
	c.mu.settled = settled
	c.mu.Unlock()
	sCtx.Go(func(ctx context.Context) error {
		c.run(ctx, cfg, sync.OnceFunc(func() { close(settled) }))
		return nil
	}, signal.RecoverWithErrOnPanic())
}

// current reports whether the connect loop runs with cfg. It ignores the TLS config of
// cfg, which tlsID identifies.
func (c *connection) current(cfg clientConfig) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.mu.stop != nil && c.mu.cfg.equal(cfg)
}

func (cfg clientConfig) equal(other clientConfig) bool {
	return cfg.url == other.url &&
		cfg.clientID == other.clientID &&
		cfg.username == other.username &&
		cfg.password == other.password &&
		cfg.keepAlive == other.keepAlive &&
		cfg.tlsID == other.tlsID &&
		cfg.hostID == other.hostID
}

// run keeps one client connected to the broker of cfg until ctx is cancelled. Paho
// can connect again on its own, but its backoff sleeps cannot be interrupted, so a
// stopped connection would hold its goroutines for up to the maximum interval. run
// calls settle when its first attempt ends.
func (c *connection) run(ctx context.Context, cfg clientConfig, settle func()) {
	defer settle()
	// The offline STATE in the last will and the online STATE that follows the
	// connect carry the same timestamp.
	var stateStamp telem.TimeStamp
	reconnect(ctx, c.ins, reconnectHooks{
		options: func(lost chan<- error) *paho.ClientOptions {
			stateStamp = telem.Now()
			return withStateWill(newClientOptions(cfg), cfg.hostID, stateStamp).
				// Ordered delivery keeps the samples of a topic in order. It requires
				// that the handler never blocks.
				SetOrderMatters(true).
				SetDefaultPublishHandler(c.onMessage).
				SetConnectionLostHandler(reportLost(lost))
		},
		serve: func(ctx context.Context, client paho.Client, lost <-chan error) {
			if err := c.setClient(ctx, client); err != nil {
				if ctx.Err() == nil {
					c.ins.L.Warn("failed to subscribe after connect", zap.Error(err))
				}
				settle()
				return
			}
			c.publishState(ctx, client, cfg.hostID, true, stateStamp)
			settle()
			c.serve(ctx, client, lost)
			c.clearClient()
			if ctx.Err() != nil {
				// The broker sends the last will only for a connection that drops.
				stateCtx, cancel := context.WithTimeout(
					context.Background(), stateTimeout,
				)
				c.publishState(stateCtx, client, cfg.hostID, false, telem.Now())
				cancel()
			}
		},
		failed: settle,
	})
}

// reconnectHooks are the parts of one reconnecting client that differ by use.
type reconnectHooks struct {
	// options builds the options of one connection attempt. lost must receive the
	// error of a connection that drops.
	options func(lost chan<- error) *paho.ClientOptions
	// serve uses one connected client until ctx is cancelled or lost fires.
	serve func(ctx context.Context, client paho.Client, lost <-chan error)
	// failed runs after an attempt that could not connect. Optional.
	failed func()
}

// reportLost returns a connection lost handler that sends the error to lost.
func reportLost(lost chan<- error) paho.ConnectionLostHandler {
	return func(_ paho.Client, err error) {
		select {
		case lost <- err:
		default:
		}
	}
}

// reconnect connects with a backoff until ctx is cancelled, and connects again after
// each connection that drops.
func reconnect(ctx context.Context, ins alamos.Instrumentation, hooks reconnectHooks) {
	brk, err := breaker.NewBreaker(ctx, breaker.Config{
		BaseInterval: time.Second,
		Scale:        2,
		MaxInterval:  maxReconnectInterval,
		MaxRetries:   breaker.InfiniteRetries,
	})
	if err != nil {
		ins.L.DPanic("invalid reconnect breaker", zap.Error(err))
		return
	}
	for {
		lost := make(chan error, 1)
		client, disconnect, err := open(ctx, ins, hooks.options(lost))
		if ctx.Err() != nil {
			if err == nil {
				disconnect()
			}
			return
		}
		if err != nil {
			ins.L.Warn("failed to connect to the MQTT broker", zap.Error(err))
			if hooks.failed != nil {
				hooks.failed()
			}
			if !brk.Wait() {
				return
			}
			continue
		}
		connectedAt := time.Now()
		ins.L.Info("connected to the MQTT broker")
		hooks.serve(ctx, client, lost)
		disconnect()
		// A connection that held is healthy again. One that did not keeps its backoff,
		// so a broker that accepts and then drops the client is not hammered.
		if time.Since(connectedAt) > maxReconnectInterval {
			brk.Reset()
		}
		if !brk.Wait() {
			return
		}
	}
}

// serve sends the rebirth requests of the host session through client until ctx is
// cancelled or the connection is lost.
func (c *connection) serve(ctx context.Context, client paho.Client, lost <-chan error) {
	held := time.NewTimer(time.Hour)
	held.Stop()
	defer held.Stop()
	for {
		c.mu.Lock()
		due, wait := c.mu.host.TakeRebirths(time.Now())
		c.mu.Unlock()
		for _, node := range due {
			c.requestRebirth(ctx, client, node)
		}
		if wait > 0 {
			held.Reset(wait)
		}
		select {
		case <-ctx.Done():
			return
		case err := <-lost:
			c.ins.L.Warn("lost the connection to the MQTT broker", zap.Error(err))
			return
		case <-c.rebirths:
		case <-held.C:
		}
	}
}

// withStateWill sets the offline STATE message of hostID as the last will of opts. An
// empty hostID leaves opts as it is: the Core is then a passive host.
func withStateWill(
	opts *paho.ClientOptions,
	hostID string,
	stamp telem.TimeStamp,
) *paho.ClientOptions {
	if hostID == "" {
		return opts
	}
	will := sparkplug.EncodeState(hostID, false, stamp)
	return opts.SetBinaryWill(will.Topic, will.Payload, will.QoS, will.Retained)
}

func (c *connection) publishState(
	ctx context.Context,
	client paho.Client,
	hostID string,
	online bool,
	stamp telem.TimeStamp,
) {
	if hostID == "" {
		return
	}
	pub := sparkplug.EncodeState(hostID, online, stamp)
	err := wait(ctx, client.Publish(pub.Topic, pub.QoS, pub.Retained, pub.Payload))
	if err != nil && ctx.Err() == nil {
		c.ins.L.Warn("failed to publish the Sparkplug B STATE message", zap.Error(err))
	}
}

// wakeRebirths tells the connect loop that the host session holds a rebirth request.
func (c *connection) wakeRebirths() {
	select {
	case c.rebirths <- struct{}{}:
	default:
	}
}

// requestRebirth sends a rebirth request to node.
func (c *connection) requestRebirth(
	ctx context.Context,
	client paho.Client,
	node sparkplug.NodeID,
) {
	pub, err := sparkplug.EncodeRebirth(node, telem.Now())
	if err == nil {
		err = wait(ctx, client.Publish(pub.Topic, pub.QoS, pub.Retained, pub.Payload))
	}
	if err != nil && ctx.Err() == nil {
		c.ins.L.Warn(
			"failed to request a Sparkplug B rebirth",
			zap.Stringer("node", node),
			zap.Error(err),
		)
	}
}

// setClient makes client the client in use. It subscribes client to every filter and
// asks every edge node for a birth, because the host session missed the messages of the
// outage. When the broker does not answer the subscribe, it returns the error and
// leaves the connection down.
func (c *connection) setClient(ctx context.Context, client paho.Client) error {
	c.subscribing.Lock()
	defer c.subscribing.Unlock()
	c.mu.Lock()
	c.mu.client = client
	filters := make(map[string]byte, len(c.mu.subs))
	for filter, sub := range c.mu.subs {
		filters[filter] = sub.qos
	}
	c.mu.session++
	c.mu.host.Reset()
	c.mu.Unlock()
	defer c.notify()
	if len(filters) == 0 {
		return nil
	}
	token := client.SubscribeMultiple(filters, nil)
	err := wait(ctx, token)
	c.mu.Lock()
	defer c.mu.Unlock()
	if err != nil {
		c.mu.client = nil
		return err
	}
	refused := refusedFilters(token)
	for filter := range filters {
		if sub, ok := c.mu.subs[filter]; ok {
			sub.refused = refusal(refused, filter)
		}
	}
	return nil
}

// clearClient marks the connection down.
func (c *connection) clearClient() {
	c.mu.Lock()
	c.mu.client = nil
	c.mu.Unlock()
	c.notify()
}

// onMessage runs on the ordered delivery goroutine of the client, so it only puts
// the message on queues and never blocks. It decodes a Sparkplug B message before it
// takes the lock, which the lock then holds only for the session state.
func (c *connection) onMessage(_ paho.Client, m paho.Message) {
	msg := message{
		topic:    m.Topic(),
		payload:  m.Payload(),
		received: c.clock.Now(),
		retained: m.Retained(),
	}
	var (
		decoded   sparkplug.Message
		inSession bool
	)
	if strings.HasPrefix(msg.topic, sparkplug.Namespace+"/") {
		decoded, inSession = c.decodeSparkplug(msg)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	msg.session = c.mu.session
	if sub, ok := c.mu.subs[msg.topic]; ok {
		for a := range sub.attachments {
			a.enqueue(msg)
		}
	}
	if inSession {
		c.onSparkplugMessage(decoded, msg.received)
	}
}

// decodeSparkplug parses a message of the Sparkplug B namespace. ok is false for a
// message that no host session applies.
func (c *connection) decodeSparkplug(msg message) (m sparkplug.Message, ok bool) {
	topic, err := sparkplug.ParseTopic(msg.topic)
	if err != nil || !topic.Type.InSession() {
		return m, false
	}
	if m, err = sparkplug.Decode(topic, msg.payload); err != nil {
		c.ins.L.Debug("dropped a Sparkplug B message", zap.Error(err))
		return m, false
	}
	return m, true
}

// onSparkplugMessage runs a decoded message through the host session and gives the
// result to the attachments that follow its edge node. The caller holds c.mu.
func (c *connection) onSparkplugMessage(m sparkplug.Message, received telem.TimeStamp) {
	attachments, followed := c.mu.nodes[m.Topic.Node]
	if !followed {
		return
	}
	ev := c.mu.host.Handle(m)
	if ev.Rebirth {
		c.wakeRebirths()
	}
	if ev.Type == "" && !ev.BirthLost {
		return
	}
	decoded := message{sparkplug: &ev, received: received, session: c.mu.session}
	for a := range attachments {
		a.enqueue(decoded)
	}
}

// notify tells every attachment that the connection state changed.
func (c *connection) notify() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for a := range c.mu.attachments {
		select {
		case a.stateChanged <- struct{}{}:
		default:
		}
	}
}

// unused reports whether c has no attachments.
func (c *connection) unused() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.mu.attachments) == 0
}

// health returns the current session, with errNotConnected while the connection is
// down, and the refusal of each filter of a while it is up.
func (c *connection) health(a *attachment) (session uint64, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.mu.client == nil {
		return c.mu.session, errNotConnected
	}
	for _, filter := range slices.Sorted(maps.Keys(a.filters)) {
		err = errors.Join(err, c.mu.subs[filter].refused)
	}
	return c.mu.session, err
}

// attach returns a new attachment whose queue holds queueSize messages.
func (c *connection) attach(queueSize int) *attachment {
	a := &attachment{
		conn:         c,
		queue:        make(chan message, queueSize),
		stateChanged: make(chan struct{}, 1),
		filters:      make(set.Set[string]),
		nodes:        make(set.Set[sparkplug.NodeID]),
	}
	// The pending signal makes the first call to next report the state of c.
	a.stateChanged <- struct{}{}
	c.mu.Lock()
	c.mu.attachments.Add(a)
	c.mu.Unlock()
	return a
}

// close stops the connect loop and disconnects the client. The caller makes sure no
// attachment remains.
func (c *connection) close() {
	c.mu.Lock()
	stop := c.mu.stop
	c.mu.stop = nil
	c.mu.Unlock()
	if stop != nil {
		stop()
	}
}

// attachment is the use of a connection by one task: its subscriptions and the queue
// its messages arrive on. One goroutine calls next. The other methods are safe for
// concurrent use.
type attachment struct {
	conn  *connection
	queue chan message
	// stateChanged signals that the health of the attachment may differ from reported.
	stateChanged chan struct{}
	// filters and nodes are guarded by conn.mu.
	filters set.Set[string]
	nodes   set.Set[sparkplug.NodeID]
	// dropped counts the messages that a full queue pushed out.
	dropped atomic.Uint64
	// reported is the text of the error that next last reported. Empty while the
	// attachment is healthy.
	reported string
	// session is the session that next last reported.
	session uint64
}

// enqueue puts msg on the queue. A full queue drops its oldest message. Only the
// delivery goroutine of the client calls enqueue.
func (a *attachment) enqueue(msg message) {
	if cap(a.queue) == 0 {
		a.dropped.Add(1)
		return
	}
	for {
		select {
		case a.queue <- msg:
			return
		default:
		}
		select {
		case <-a.queue:
			a.dropped.Add(1)
		default:
		}
	}
}

// next blocks until a message arrives, the health of the attachment changes, the
// connection starts a new session, or ctx is cancelled. For a change of health or
// session it returns a message with only its session set, and the error of the
// attachment.
func (a *attachment) next(ctx context.Context) (message, error) {
	for {
		select {
		case <-ctx.Done():
			return message{}, ctx.Err()
		case msg := <-a.queue:
			return msg, nil
		case <-a.stateChanged:
			session, err := a.conn.health(a)
			reported := ""
			if err != nil {
				reported = err.Error()
			}
			if reported == a.reported && session == a.session {
				continue
			}
			a.reported, a.session = reported, session
			return message{session: session}, err
		}
	}
}

// poll returns a message that waits on the queue, with no wait. ok is false for an
// empty queue.
func (a *attachment) poll() (msg message, ok bool) {
	select {
	case msg = <-a.queue:
		return msg, true
	default:
		return message{}, false
	}
}

// settle blocks until the first connection attempt to the broker ends, so that the
// state the attachment reports first is a known one.
func (a *attachment) settle(ctx context.Context) error {
	a.conn.mu.Lock()
	settled := a.conn.mu.settled
	a.conn.mu.Unlock()
	select {
	case <-settled:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// follow makes the attachment receive the decoded messages of node. It asks the node
// for a birth, which gives the attachment the current value of every tag.
func (a *attachment) follow(ctx context.Context, node sparkplug.NodeID) error {
	c := a.conn
	c.mu.Lock()
	a.nodes.Add(node)
	attachments, ok := c.mu.nodes[node]
	if !ok {
		attachments = make(set.Set[*attachment])
		c.mu.nodes[node] = attachments
	}
	attachments.Add(a)
	c.mu.Unlock()
	for _, filter := range node.Filters() {
		if err := a.subscribe(ctx, filter, 0); err != nil {
			return err
		}
	}
	c.mu.Lock()
	c.mu.host.WantRebirth(node)
	c.mu.Unlock()
	c.wakeRebirths()
	return nil
}

// subscribe adds filter to the attachment. The connection holds one subscription for
// each filter, at the highest quality of service that an attachment asked for.
func (a *attachment) subscribe(ctx context.Context, filter string, qos byte) error {
	c := a.conn
	c.subscribing.Lock()
	defer c.subscribing.Unlock()
	c.mu.Lock()
	a.filters.Add(filter)
	sub, ok := c.mu.subs[filter]
	if !ok {
		sub = &subscription{attachments: make(set.Set[*attachment])}
		c.mu.subs[filter] = sub
	}
	sub.attachments.Add(a)
	needed := !ok || qos > sub.qos
	if qos > sub.qos {
		sub.qos = qos
	}
	client := c.mu.client
	refused := sub.refused
	c.mu.Unlock()
	// A connection that is down subscribes to every filter when it connects.
	if client == nil {
		return nil
	}
	if !needed {
		return refused
	}
	token := client.Subscribe(filter, qos, nil)
	if err := wait(ctx, token); err != nil {
		return err
	}
	c.mu.Lock()
	sub.refused = refusal(refusedFilters(token), filter)
	refused = sub.refused
	c.mu.Unlock()
	// Other attachments of the filter report the refusal through next.
	c.notify()
	return refused
}

// publish sends one message. It returns errNotConnected when the connection is down:
// a command that waits out an outage is stale when it arrives.
func (a *attachment) publish(
	ctx context.Context,
	topic string,
	qos byte,
	retained bool,
	payload []byte,
) error {
	c := a.conn
	c.mu.Lock()
	client := c.mu.client
	c.mu.Unlock()
	if client == nil {
		return errNotConnected
	}
	return wait(ctx, client.Publish(topic, qos, retained, payload))
}

// close removes the attachment and the subscriptions that no other attachment uses.
// It reports whether the connection has no attachments left.
func (a *attachment) close() (unused bool) {
	c := a.conn
	c.subscribing.Lock()
	defer c.subscribing.Unlock()
	c.mu.Lock()
	delete(c.mu.attachments, a)
	var orphaned []string
	for filter := range a.filters {
		sub := c.mu.subs[filter]
		delete(sub.attachments, a)
		if len(sub.attachments) == 0 {
			delete(c.mu.subs, filter)
			orphaned = append(orphaned, filter)
		}
	}
	for node := range a.nodes {
		delete(c.mu.nodes[node], a)
		if len(c.mu.nodes[node]) == 0 {
			delete(c.mu.nodes, node)
			// A host that no longer hears a node holds a stale birth of it.
			c.mu.host.Forget(node)
		}
	}
	client := c.mu.client
	unused = len(c.mu.attachments) == 0
	c.mu.Unlock()
	if len(orphaned) > 0 && client != nil && !unused {
		ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
		defer cancel()
		if err := wait(ctx, client.Unsubscribe(orphaned...)); err != nil {
			c.ins.L.Warn("failed to unsubscribe", zap.Error(err))
		}
	}
	return unused
}

// subscribeRefused is the lowest return code of a subscription that the broker refused.
const subscribeRefused = 0x80

// refusedFilters returns the filters that the broker refused in the answer to a
// completed subscribe token.
func refusedFilters(token paho.Token) set.Set[string] {
	refused := make(set.Set[string])
	for filter, code := range token.(*paho.SubscribeToken).Result() {
		if code >= subscribeRefused {
			refused.Add(filter)
		}
	}
	return refused
}

// refusal returns the temporary error of filter when refused holds it, and nil when it
// does not.
func refusal(refused set.Set[string], filter string) error {
	if !refused.Contains(filter) {
		return nil
	}
	return errors.Mark(
		errors.Newf("the broker refused the subscription to %s", filter),
		driver.ErrTemporary,
	)
}

// waitSubscribe blocks like wait for a subscribe token, and then returns the refusal of
// each filter that the broker refused.
func waitSubscribe(ctx context.Context, token paho.Token) error {
	if err := wait(ctx, token); err != nil {
		return err
	}
	refused := refusedFilters(token)
	var err error
	for _, filter := range slices.Sorted(maps.Keys(refused)) {
		err = errors.Join(err, refusal(refused, filter))
	}
	return err
}

// wait blocks until token completes, ctx is cancelled, or operationTimeout passes.
func wait(ctx context.Context, token paho.Token) error {
	timer := time.NewTimer(operationTimeout)
	defer timer.Stop()
	select {
	case <-token.Done():
		return token.Error()
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return errBrokerTimeout
	}
}

// pool holds the shared connection of each broker device. Safe for concurrent use.
type pool struct {
	ins     alamos.Instrumentation
	devices *device.Service
	conns   map[device.Key]*connection
	// rebirthInterval is the shortest time between two Sparkplug B rebirth requests to
	// one edge node.
	rebirthInterval time.Duration
	mu              sync.Mutex
}

func newPool(
	ins alamos.Instrumentation,
	devices *device.Service,
	rebirthInterval time.Duration,
) *pool {
	return &pool{
		ins:             ins,
		devices:         devices,
		conns:           make(map[device.Key]*connection),
		rebirthInterval: rebirthInterval,
	}
}

// attach returns an attachment to the connection of the broker device key. It opens the
// connection for the first attachment, and points an open connection at the current
// properties of the device.
func (p *pool) attach(
	ctx context.Context,
	key device.Key,
	queueSize int,
) (*attachment, error) {
	for {
		conn := p.connection(key)
		conn.lifecycle.Lock()
		if conn.lifecycle.closed {
			// The last attachment left between the lookup and the lock.
			conn.lifecycle.Unlock()
			continue
		}
		if err := p.refresh(ctx, conn, key); err != nil {
			if conn.unused() {
				p.discard(key, conn)
			}
			conn.lifecycle.Unlock()
			return nil, err
		}
		a := conn.attach(queueSize)
		conn.lifecycle.Unlock()
		return a, nil
	}
}

// connection returns the connection of the broker device key, and opens one when
// the pool holds none.
func (p *pool) connection(key device.Key) *connection {
	p.mu.Lock()
	defer p.mu.Unlock()
	conn, ok := p.conns[key]
	if !ok {
		conn = newConnection(p.ins.Child(key), p.rebirthInterval)
		p.conns[key] = conn
	}
	return conn
}

// state reports whether the broker device key has an open connection and whether it is
// connected. err is the error of the last time the connection read the properties of
// the device.
func (p *pool) state(key device.Key) (open, connected bool, err error) {
	p.mu.Lock()
	conn, ok := p.conns[key]
	p.mu.Unlock()
	if !ok {
		return false, false, nil
	}
	conn.mu.Lock()
	defer conn.mu.Unlock()
	return conn.mu.stop != nil, conn.mu.client != nil, conn.mu.configErr
}

// live reports whether the broker device key has an open connection with the settings
// of cfg, and whether that connection is connected.
func (p *pool) live(key device.Key, cfg clientConfig) (live, connected bool) {
	p.mu.Lock()
	conn, ok := p.conns[key]
	p.mu.Unlock()
	if !ok {
		return false, false
	}
	conn.mu.Lock()
	defer conn.mu.Unlock()
	return conn.mu.stop != nil && conn.mu.cfg.equal(cfg), conn.mu.client != nil
}

// reconfigure points the open connection of the broker device key at the current
// properties of the device. It does nothing for a device with no open connection. On an
// error the connection keeps its configuration, and state reports the error.
func (p *pool) reconfigure(ctx context.Context, key device.Key) {
	p.mu.Lock()
	conn, ok := p.conns[key]
	p.mu.Unlock()
	if !ok {
		return
	}
	conn.lifecycle.Lock()
	defer conn.lifecycle.Unlock()
	if conn.lifecycle.closed {
		return
	}
	if err := p.refresh(ctx, conn, key); err != nil && ctx.Err() == nil {
		conn.ins.L.Warn(
			"the connection cannot take the properties of the broker device",
			zap.Error(err),
		)
	}
}

// refresh points conn at the current properties of the broker device key, and reads the
// TLS files of the device only when those properties changed. The caller holds
// conn.lifecycle, so a refresh that starts later reads a later version of the device.
func (p *pool) refresh(
	ctx context.Context,
	conn *connection,
	key device.Key,
) (err error) {
	defer func() {
		conn.mu.Lock()
		conn.mu.configErr = err
		conn.mu.Unlock()
	}()
	var dev device.Device
	if err = p.devices.NewRetrieve().
		Where(device.MatchKeys(key)).
		Entry(&dev).
		Exec(ctx, nil); err != nil {
		return err
	}
	props, err := parseProperties(dev)
	if err != nil {
		return err
	}
	cfg, err := newClientSettings(dev, props)
	if err != nil || conn.current(cfg) {
		return err
	}
	if cfg.tls, err = loadTLS(props); err != nil {
		return err
	}
	conn.configure(cfg)
	return nil
}

// release closes a, and closes its connection when a was the last attachment.
func (p *pool) release(key device.Key, a *attachment) {
	conn := a.conn
	conn.lifecycle.Lock()
	defer conn.lifecycle.Unlock()
	if a.close() {
		p.discard(key, conn)
	}
}

// discard closes conn, which has no attachments, and removes it from the pool. The
// caller holds conn.lifecycle.
func (p *pool) discard(key device.Key, conn *connection) {
	conn.close()
	conn.lifecycle.closed = true
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.conns[key] == conn {
		delete(p.conns, key)
	}
}

// newClientOptions returns the options that every client of the broker of cfg uses.
func newClientOptions(cfg clientConfig) *paho.ClientOptions {
	return paho.NewClientOptions().
		AddBroker(cfg.url).
		SetClientID(cfg.clientID).
		SetUsername(cfg.username).
		SetPassword(cfg.password).
		SetTLSConfig(cfg.tls).
		SetKeepAlive(cfg.keepAlive).
		// Version 4 is MQTT 3.1.1. An explicit version stops the fallback to 3.1.
		SetProtocolVersion(4).
		SetCleanSession(true).
		SetAutoReconnect(false).
		SetConnectRetry(false).
		SetConnectTimeout(operationTimeout).
		SetWriteTimeout(operationTimeout)
}

// open connects a new client with opts. It makes one attempt, and cancelling ctx
// aborts that attempt. The caller calls disconnect when it is done with the client.
func open(
	ctx context.Context,
	ins alamos.Instrumentation,
	opts *paho.ClientOptions,
) (client paho.Client, disconnect func(), err error) {
	netCtx, closeNet := context.WithCancel(context.Background())
	opts.SetCustomOpenConnectionFn(
		func(uri *url.URL, opts paho.ClientOptions) (net.Conn, error) {
			return dial(netCtx, ins, uri, opts)
		},
	)
	client = paho.NewClient(opts)
	disconnect = func() {
		client.Disconnect(uint(disconnectQuiesce.Milliseconds()))
		closeNet()
	}
	token := client.Connect()
	select {
	case <-token.Done():
	case <-ctx.Done():
		closeNet()
		<-token.Done()
	}
	if err = errors.Combine(ctx.Err(), describeRefusal(token.Error())); err != nil {
		disconnect()
		return nil, nil, err
	}
	return client, disconnect, nil
}

// describeRefusal adds the cause that an operator can act on to a connection that the
// broker refused.
func describeRefusal(err error) error {
	switch {
	case errors.Is(err, packets.ErrorRefusedNotAuthorised),
		errors.Is(err, packets.ErrorRefusedBadUsernameOrPassword):
		return errors.Wrap(err, "the broker rejected the username or password")
	case errors.Is(err, packets.ErrorRefusedIDRejected):
		return errors.Wrap(err, "the broker rejected the client ID")
	case errors.Is(err, packets.ErrorRefusedServerUnavailable):
		return errors.Wrap(err, "the broker is not available")
	default:
		return err
	}
}

// dial opens the network connection of a client. Cancelling ctx aborts the dial, and
// closes the connection at any later time.
func dial(
	ctx context.Context,
	ins alamos.Instrumentation,
	uri *url.URL,
	opts paho.ClientOptions,
) (net.Conn, error) {
	dialCtx, cancel := context.WithTimeout(ctx, opts.ConnectTimeout)
	defer cancel()
	conn, err := proxy.Dial(dialCtx, "tcp", uri.Host)
	if err != nil {
		return nil, err
	}
	if uri.Scheme == "ssl" {
		tlsCfg := opts.TLSConfig.Clone()
		if tlsCfg == nil {
			tlsCfg = &tls.Config{}
		}
		if tlsCfg.ServerName == "" {
			tlsCfg.ServerName = uri.Hostname()
		}
		tlsConn := tls.Client(conn, tlsCfg)
		if err = tlsConn.HandshakeContext(dialCtx); err != nil {
			return nil, errors.Combine(err, conn.Close())
		}
		conn = tlsConn
	}
	context.AfterFunc(ctx, func() {
		// Paho closes the connection first on a clean disconnect.
		if err := conn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			ins.L.Debug("failed to close the broker connection", zap.Error(err))
		}
	})
	return conn, nil
}
