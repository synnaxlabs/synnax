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
	"net"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/service/device"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt/sparkplug"
	"github.com/synnaxlabs/x/breaker"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
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
	// rebirthBacklog is the count of rebirth requests that can wait to be sent.
	rebirthBacklog = 64
)

// errNotConnected is a temporary error: the connection connects again on its own.
var errNotConnected = errors.Wrap(driver.ErrTemporary, "broker is not connected")

// message is one message received from a broker.
type message struct {
	// sparkplug is the decoded form of a Sparkplug B message. Such a message has no
	// topic and no payload.
	sparkplug *sparkplug.Event
	topic     string
	payload   []byte
	// received is the time the message handler saw the message.
	received telem.TimeStamp
	retained bool
}

// subscription is one topic filter on a connection and the attachments that use it.
type subscription struct {
	attachments set.Set[*attachment]
	qos         byte
	wildcard    bool
	// sparkplug is true for the filter of an edge node. Its messages reach the
	// attachments decoded, through the host session, and never as they arrived.
	sparkplug bool
}

// rebirthRequest asks for a rebirth request to node.
type rebirthRequest struct {
	node sparkplug.NodeID
	// forced sends the request to a node that the host already holds a birth of.
	forced bool
}

// connection is the one MQTT client that the tasks of a broker device share. It
// connects again after a loss, subscribes again after each connect, and gives each
// attachment its own bounded queue. Safe for concurrent use, except that the caller
// serializes configure and close.
type connection struct {
	// rebirths holds the rebirth requests that the connect loop sends.
	rebirths chan rebirthRequest
	ins      alamos.Instrumentation
	mu       struct {
		// host is the Sparkplug B session state of the edge nodes in nodes.
		host *sparkplug.Host
		// nodes holds the attachments that follow each edge node.
		nodes map[sparkplug.NodeID]set.Set[*attachment]
		// client is nil while the connection is down.
		client paho.Client
		// stop ends the connect loop and waits until its client has left the broker.
		stop func()
		// settled closes when the first connection attempt of the loop ends.
		settled chan struct{}
		// subs holds every subscription by its topic filter.
		subs        map[string]*subscription
		attachments set.Set[*attachment]
		cfg         clientConfig
		// wildcards counts the subscriptions whose filter holds a wildcard, so a
		// connection with none skips the filter match on every message.
		wildcards int
		sync.Mutex
	}
}

func newConnection(
	ins alamos.Instrumentation,
	rebirthInterval time.Duration,
) *connection {
	c := &connection{ins: ins, rebirths: make(chan rebirthRequest, rebirthBacklog)}
	c.mu.host = sparkplug.NewHost()
	c.mu.host.RebirthInterval = rebirthInterval
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
	ctx, cancel := context.WithCancel(context.Background())
	done, settled := make(chan struct{}), make(chan struct{})
	c.mu.Lock()
	c.mu.stop = func() {
		cancel()
		<-done
	}
	c.mu.settled = settled
	c.mu.Unlock()
	go func() {
		defer close(done)
		c.run(ctx, cfg, sync.OnceFunc(func() { close(settled) }))
	}()
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
			c.setClient(ctx, client)
			c.publishState(ctx, client, cfg.hostID, true, stateStamp)
			settle()
			c.serve(ctx, client, lost)
			c.setClient(ctx, nil)
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
		client, disconnect, err := open(ctx, hooks.options(lost))
		if ctx.Err() != nil {
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

// serve sends rebirth requests through client until ctx is cancelled or the
// connection is lost.
func (c *connection) serve(ctx context.Context, client paho.Client, lost <-chan error) {
	for {
		select {
		case <-ctx.Done():
			return
		case err := <-lost:
			c.ins.L.Warn("lost the connection to the MQTT broker", zap.Error(err))
			return
		case req := <-c.rebirths:
			c.requestRebirth(ctx, client, req)
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
	payload := sparkplug.EncodeState(false, stamp)
	return opts.SetBinaryWill(sparkplug.StateTopic(hostID), payload, 1, true)
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
	payload := sparkplug.EncodeState(online, stamp)
	err := wait(ctx, client.Publish(sparkplug.StateTopic(hostID), 1, true, payload))
	if err != nil && ctx.Err() == nil {
		c.ins.L.Warn("failed to publish the Sparkplug B STATE message", zap.Error(err))
	}
}

// queueRebirth asks the connect loop for a rebirth request. A full backlog drops the
// request: the next message of the edge node asks again.
func (c *connection) queueRebirth(req rebirthRequest) {
	select {
	case c.rebirths <- req:
	default:
	}
}

// requestRebirth sends a rebirth request to the edge node of req, no more often than
// the host session allows. A request that comes too early is queued again for later.
func (c *connection) requestRebirth(
	ctx context.Context,
	client paho.Client,
	req rebirthRequest,
) {
	c.mu.Lock()
	_, followed := c.mu.nodes[req.node]
	if !followed || (!req.forced && c.mu.host.Born(req.node)) {
		c.mu.Unlock()
		return
	}
	left := c.mu.host.TakeRebirth(req.node, time.Now())
	c.mu.Unlock()
	if left > 0 {
		time.AfterFunc(left, func() { c.queueRebirth(req) })
		return
	}
	payload, err := sparkplug.EncodeRebirth(telem.Now())
	if err == nil {
		topic := sparkplug.CommandTopic(req.node, "").String()
		err = wait(ctx, client.Publish(topic, 0, false, payload))
	}
	if err != nil && ctx.Err() == nil {
		c.ins.L.Warn(
			"failed to request a Sparkplug B rebirth",
			zap.Stringer("node", req.node),
			zap.Error(err),
		)
	}
}

// setClient makes client the client in use, where nil means the connection is down.
// A new client subscribes to every filter and asks every edge node for a birth,
// because the host session missed the messages of the outage.
func (c *connection) setClient(ctx context.Context, client paho.Client) {
	c.mu.Lock()
	c.mu.client = client
	filters := make(map[string]byte, len(c.mu.subs))
	for filter, sub := range c.mu.subs {
		filters[filter] = sub.qos
	}
	var nodes []sparkplug.NodeID
	if client != nil {
		c.mu.host.Reset()
		for node := range c.mu.nodes {
			nodes = append(nodes, node)
		}
	}
	c.mu.Unlock()
	if client != nil && len(filters) > 0 {
		if err := wait(ctx, client.SubscribeMultiple(filters, nil)); err != nil {
			c.ins.L.Error("failed to subscribe after connect", zap.Error(err))
		}
	}
	for _, node := range nodes {
		c.queueRebirth(rebirthRequest{node: node})
	}
	c.notify()
}

// onMessage runs on the ordered delivery goroutine of the client, so it only puts
// the message on queues and never blocks.
func (c *connection) onMessage(_ paho.Client, m paho.Message) {
	msg := message{
		topic:    m.Topic(),
		payload:  m.Payload(),
		received: telem.Now(),
		retained: m.Retained(),
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if sub, ok := c.mu.subs[msg.topic]; ok {
		for a := range sub.attachments {
			a.enqueue(msg)
		}
	}
	if len(c.mu.nodes) > 0 && strings.HasPrefix(msg.topic, sparkplug.Namespace+"/") {
		c.onSparkplugMessage(msg)
	}
	if c.mu.wildcards == 0 {
		return
	}
	for filter, sub := range c.mu.subs {
		if !sub.wildcard || sub.sparkplug || !topicMatches(filter, msg.topic) {
			continue
		}
		for a := range sub.attachments {
			a.enqueue(msg)
		}
	}
}

// onSparkplugMessage runs msg through the host session and gives the result to the
// attachments that follow its edge node. The caller holds c.mu.
func (c *connection) onSparkplugMessage(msg message) {
	topic, err := sparkplug.ParseTopic(msg.topic)
	if err != nil {
		return
	}
	attachments, followed := c.mu.nodes[topic.Node]
	if !followed {
		return
	}
	ev, err := c.mu.host.Handle(topic, msg.payload)
	if err != nil {
		c.ins.L.Debug("dropped a Sparkplug B message", zap.Error(err))
		return
	}
	if ev.Rebirth {
		c.queueRebirth(rebirthRequest{node: topic.Node})
	}
	if ev.Type == "" {
		return
	}
	decoded := message{sparkplug: &ev, received: msg.received}
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

func (c *connection) connected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.mu.client != nil
}

// attach returns a new attachment whose queue holds queueSize messages.
func (c *connection) attach(queueSize int) *attachment {
	a := &attachment{
		conn:         c,
		queue:        make(chan message, queueSize),
		stateChanged: make(chan struct{}, 1),
		filters:      make(set.Set[string]),
		nodes:        make(set.Set[sparkplug.NodeID]),
		// An attachment assumes a connection until next sees that there is none.
		reportedConnected: true,
	}
	// The pending signal makes the first call to next report a connection that is
	// down.
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
	// stateChanged signals that the connection state may differ from
	// reportedConnected.
	stateChanged chan struct{}
	// filters and nodes are guarded by conn.mu.
	filters set.Set[string]
	nodes   set.Set[sparkplug.NodeID]
	// dropped counts the messages that a full queue pushed out.
	dropped atomic.Uint64
	// reportedConnected is the connection state that next last reported.
	reportedConnected bool
}

// enqueue puts msg on the queue. A full queue drops its oldest message. Only the
// delivery goroutine of the client calls enqueue.
func (a *attachment) enqueue(msg message) {
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

// next blocks until a message arrives, the connection state changes, or ctx is
// cancelled. It returns errNotConnected once for each loss of the connection, and a
// zero message with a nil error once for each recovery.
func (a *attachment) next(ctx context.Context) (message, error) {
	for {
		select {
		case <-ctx.Done():
			return message{}, ctx.Err()
		case msg := <-a.queue:
			return msg, nil
		case <-a.stateChanged:
			connected := a.conn.connected()
			if connected == a.reportedConnected {
				continue
			}
			a.reportedConnected = connected
			if !connected {
				return message{}, errNotConnected
			}
			return message{}, nil
		}
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

// subscribe adds filter to the attachment. The connection holds one subscription for
// each filter, at the highest quality of service that an attachment asked for.
func (a *attachment) subscribe(ctx context.Context, filter string, qos byte) error {
	return a.subscribeFilter(ctx, filter, qos, false)
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
		if err := a.subscribeFilter(ctx, filter, 0, true); err != nil {
			return err
		}
	}
	c.queueRebirth(rebirthRequest{node: node, forced: true})
	return nil
}

func (a *attachment) subscribeFilter(
	ctx context.Context,
	filter string,
	qos byte,
	isSparkplug bool,
) error {
	c := a.conn
	c.mu.Lock()
	a.filters.Add(filter)
	sub, ok := c.mu.subs[filter]
	if !ok {
		sub = &subscription{
			attachments: make(set.Set[*attachment]),
			wildcard:    strings.ContainsAny(filter, "+#"),
			sparkplug:   isSparkplug,
		}
		c.mu.subs[filter] = sub
		if sub.wildcard {
			c.mu.wildcards++
		}
	}
	sub.attachments.Add(a)
	needed := !ok || qos > sub.qos
	if qos > sub.qos {
		sub.qos = qos
	}
	client := c.mu.client
	c.mu.Unlock()
	// A connection that is down subscribes to every filter when it connects.
	if !needed || client == nil {
		return nil
	}
	return wait(ctx, client.Subscribe(filter, qos, nil))
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
	c.mu.Lock()
	delete(c.mu.attachments, a)
	var orphaned []string
	for filter := range a.filters {
		sub := c.mu.subs[filter]
		delete(sub.attachments, a)
		if len(sub.attachments) == 0 {
			delete(c.mu.subs, filter)
			if sub.wildcard {
				c.mu.wildcards--
			}
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
		return errors.Wrap(driver.ErrTemporary, "timed out waiting for the broker")
	}
}

// topicMatches reports whether topic matches an MQTT topic filter. As the MQTT
// specification requires, a filter that starts with a wildcard does not match a
// topic that starts with "$".
func topicMatches(filter, topic string) bool {
	if strings.HasPrefix(topic, "$") && (filter[0] == '+' || filter[0] == '#') {
		return false
	}
	for {
		level, rest, more := strings.Cut(filter, "/")
		if level == "#" {
			return true
		}
		topicLevel, topicRest, topicMore := strings.Cut(topic, "/")
		if level != "+" && level != topicLevel {
			return false
		}
		if !more || !topicMore {
			// "a/#" also matches "a", the parent level.
			return more == topicMore || (more && rest == "#")
		}
		filter, topic = rest, topicRest
	}
}

// pool holds the shared connection of each broker device. Safe for concurrent use.
type pool struct {
	ins   alamos.Instrumentation
	conns map[device.Key]*connection
	// rebirthInterval is the shortest time between two Sparkplug B rebirth requests to
	// one edge node.
	rebirthInterval time.Duration
	mu              sync.Mutex
}

func newPool(ins alamos.Instrumentation, rebirthInterval time.Duration) *pool {
	return &pool{
		ins:             ins,
		conns:           make(map[device.Key]*connection),
		rebirthInterval: rebirthInterval,
	}
}

// attach returns an attachment to the connection of dev. It opens the connection for
// the first attachment, and points an open connection at the current properties of
// dev.
func (p *pool) attach(dev device.Device, queueSize int) (*attachment, error) {
	cfg, err := newClientConfig(dev)
	if err != nil {
		return nil, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	conn, ok := p.conns[dev.Key]
	if !ok {
		conn = newConnection(p.ins.Child(dev.Key), p.rebirthInterval)
		p.conns[dev.Key] = conn
	}
	conn.configure(cfg)
	return conn.attach(queueSize), nil
}

// state reports whether dev has an open connection and whether it is connected. It
// first points an open connection at the current properties of dev.
func (p *pool) state(dev device.Device) (open, connected bool, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	conn, ok := p.conns[dev.Key]
	if !ok {
		return false, false, nil
	}
	cfg, err := newClientConfig(dev)
	if err != nil {
		return true, false, err
	}
	conn.configure(cfg)
	return true, conn.connected(), nil
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
	opts *paho.ClientOptions,
) (client paho.Client, disconnect func(), err error) {
	netCtx, closeNet := context.WithCancel(context.Background())
	opts.SetCustomOpenConnectionFn(
		func(uri *url.URL, opts paho.ClientOptions) (net.Conn, error) {
			return dial(netCtx, uri, opts)
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
	context.AfterFunc(ctx, func() { _ = conn.Close() })
	return conn, nil
}

// release closes a, and closes its connection when a was the last attachment.
func (p *pool) release(key device.Key, a *attachment) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !a.close() {
		return
	}
	a.conn.close()
	if p.conns[key] == a.conn {
		delete(p.conns, key)
	}
}
