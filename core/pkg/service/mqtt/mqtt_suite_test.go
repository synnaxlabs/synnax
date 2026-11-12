// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mqtt_test

import (
	"context"
	"crypto/tls"
	"io"
	"log/slog"
	"net"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	mochi "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/listeners"
	"github.com/mochi-mqtt/server/v2/packets"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gleak"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	calcgraph "github.com/synnaxlabs/synnax/pkg/service/channel/calculation/graph"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	taskconfig "github.com/synnaxlabs/synnax/pkg/service/task/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	db         *gorp.DB
	rackSvc    *rack.Service
	deviceSvc  *device.Service
	channelSvc *channel.Service
	framerSvc  *framer.Service
	statusSvc  *status.Service
	taskSvc    *task.Service
	// host is the key of the Core that the services run on.
	host node.Key
)

func TestMQTT(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service MQTT Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	core := testutil.NewNode(ctx)
	host = core.Identity.HostKey
	db = core.DB
	MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB: db,
	}))
	statusSvc = MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB: db,
	}))
	roleSvc := MustOpen(role.OpenService(ctx, role.ServiceConfig{
		DB: db,
	}))
	rackSvc = MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
		DB:         db,
		HostKey:    1,
		Status:     statusSvc,
		DriverRole: MustSucceed(racktestutil.CreateDriverRole(ctx, db, roleSvc)),
		// No Driver reports for the rack, so a check would overwrite task statuses.
		HealthCheckInterval: telem.Hour,
	}))
	deviceSvc = MustOpen(device.OpenService(ctx, device.ServiceConfig{
		DB:     db,
		Status: statusSvc,
		Rack:   rackSvc,
	}))
	channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
		Channel: core.Channel,
		DB:      db,
		HostKey: core.Identity.HostKey,
		Status:  statusSvc,
	}))
	channelGraph := MustOpen(calcgraph.Open(ctx, calcgraph.Config{
		DB:      db,
		Channel: channelSvc,
		Status:  statusSvc,
	}))
	framerSvc = MustOpen(framer.OpenService(ctx, framer.ServiceConfig{
		DB:           db,
		Framer:       core.Framer,
		Channel:      channelSvc,
		ChannelGraph: channelGraph,
	}))
	mqttSvc := MustOpen(mqtt.OpenService(ctx, mqtt.ServiceConfig{
		DB:      db,
		Channel: channelSvc,
	}))
	taskSvc = MustOpen(task.OpenService(ctx, task.ServiceConfig{
		DB:      db,
		Rack:    rackSvc,
		Status:  statusSvc,
		Channel: channelSvc,
		ImEx:    imex.NewService(),
		Configs: MustSucceed(taskconfig.NewRegistry(mqttSvc.Stores()...)),
	}))
})

// received is one message that a test broker delivered to a collector.
type received struct {
	topic    string
	payload  string
	retained bool
}

// testBroker is an MQTT broker embedded in the test process.
type testBroker struct {
	server   *mochi.Server
	listener *tcpListener
	port     int
	mu       sync.Mutex
	// messages holds what the collector of the broker received.
	messages []received
	// clientIDs holds the client ID of each connection, in order.
	clientIDs []string
	// keepAlives holds the keep-alive interval in seconds of each connection, in order.
	keepAlives []uint16
	// refusing makes the broker refuse each new connection.
	refusing atomic.Bool
}

// connectHook records the client ID and the keep-alive interval of each connection to a
// test broker. It removes the earlier client of an ID before the new one attaches.
// Mochi can end the cleanup of a client that left after its successor attached, and
// that cleanup then deletes the successor, which drops every message sent to it.
type connectHook struct {
	mochi.HookBase
	broker *testBroker
}

func (h *connectHook) ID() string { return "connect" }

func (h *connectHook) Provides(b byte) bool { return b == mochi.OnConnect }

func (h *connectHook) OnConnect(cl *mochi.Client, pk packets.Packet) error {
	if h.broker.refusing.Load() {
		return packets.ErrServerUnavailable
	}
	if old, ok := h.broker.server.Clients.Get(cl.ID); ok && !old.Net.Inline {
		old.Stop(packets.ErrSessionTakenOver)
		h.broker.listener.awaitServed(old.Net.Conn)
	}
	h.broker.mu.Lock()
	defer h.broker.mu.Unlock()
	h.broker.clientIDs = append(h.broker.clientIDs, cl.ID)
	h.broker.keepAlives = append(h.broker.keepAlives, pk.Connect.Keepalive)
	return nil
}

// openHook lets every client connect, and refuses each subscription to one of refused.
type openHook struct {
	mochi.HookBase
	refused []string
}

func (h *openHook) ID() string { return "open" }

func (h *openHook) Provides(b byte) bool {
	return b == mochi.OnConnectAuthenticate || b == mochi.OnACLCheck
}

func (h *openHook) OnConnectAuthenticate(*mochi.Client, packets.Packet) bool {
	return true
}

func (h *openHook) OnACLCheck(_ *mochi.Client, topic string, write bool) bool {
	return write || !slices.Contains(h.refused, topic)
}

// unackedHook leaves the publishes on topic unanswered, so that each stays in flight,
// until released is set.
type unackedHook struct {
	mochi.HookBase
	topic    string
	held     atomic.Int64
	released atomic.Bool
}

func (h *unackedHook) ID() string { return "unacked" }

func (h *unackedHook) Provides(b byte) bool { return b == mochi.OnPublish }

func (h *unackedHook) OnPublish(
	_ *mochi.Client,
	pk packets.Packet,
) (packets.Packet, error) {
	if pk.TopicName != h.topic || h.released.Load() {
		return pk, nil
	}
	h.held.Add(1)
	return pk, packets.ErrRejectPacket
}

// tcpListener is a TCP listener that returns from Close only after every connection it
// accepted is done. Mochi counts a connection in Listeners.ClientsWg from the goroutine
// that serves it, so a connection accepted during Server.Close races the Wait of that
// group. listeners.TCP is not used, because it leaves open a connection that it accepts
// during Close, and the client of that connection then waits for a CONNACK.
type tcpListener struct {
	cfg    listeners.Config
	listen net.Listener
	// accepting closes when Serve stops accepting connections.
	accepting chan struct{}
	mu        struct {
		sync.Mutex
		closed bool
		// conns holds each connection that is served, with a channel that closes
		// when mochi is done with it.
		conns map[net.Conn]chan struct{}
	}
	serving sync.WaitGroup
	// closeErr holds the errors of the listener and of the connections that Close
	// closed.
	closeErr error
}

var _ listeners.Listener = (*tcpListener)(nil)

func newTCPListener(cfg listeners.Config) *tcpListener {
	l := &tcpListener{cfg: cfg, accepting: make(chan struct{})}
	l.mu.conns = make(map[net.Conn]chan struct{})
	return l
}

func (l *tcpListener) ID() string { return l.cfg.ID }

func (l *tcpListener) Address() string { return l.listen.Addr().String() }

func (l *tcpListener) Protocol() string { return "tcp" }

func (l *tcpListener) Init(*slog.Logger) (err error) {
	if l.cfg.TLSConfig != nil {
		l.listen, err = tls.Listen("tcp", l.cfg.Address, l.cfg.TLSConfig)
	} else {
		l.listen, err = net.Listen("tcp", l.cfg.Address)
	}
	return err
}

func (l *tcpListener) Serve(establish listeners.EstablishFn) {
	defer close(l.accepting)
	for {
		c, err := l.listen.Accept()
		if err != nil {
			return
		}
		l.mu.Lock()
		if l.mu.closed {
			l.closeErr = errors.Combine(l.closeErr, c.Close())
			l.mu.Unlock()
			continue
		}
		served := make(chan struct{})
		l.mu.conns[c] = served
		l.serving.Add(1)
		l.mu.Unlock()
		go func() {
			// Mochi runs its hooks for c on this goroutine.
			defer GinkgoRecover()
			defer func() {
				l.mu.Lock()
				delete(l.mu.conns, c)
				l.mu.Unlock()
				close(served)
				l.serving.Done()
			}()
			// A client that leaves ends establish with an error, so the error only
			// shows in the output of a failed spec.
			if err := establish(l.cfg.ID, c); err != nil {
				GinkgoWriter.Println(err)
			}
		}()
	}
}

// awaitServed blocks until mochi is done with c.
func (l *tcpListener) awaitServed(c net.Conn) {
	GinkgoHelper()
	l.mu.Lock()
	served, ok := l.mu.conns[c]
	l.mu.Unlock()
	if ok {
		Eventually(served).Should(BeClosed())
	}
}

func (l *tcpListener) Close(closeClients listeners.CloseFn) {
	GinkgoHelper()
	l.mu.Lock()
	l.mu.closed = true
	l.closeErr = l.listen.Close()
	for c := range l.mu.conns {
		l.closeErr = errors.Combine(l.closeErr, errors.Skip(c.Close(), net.ErrClosed))
	}
	l.mu.Unlock()
	Eventually(l.accepting).Should(BeClosed())
	// Clients.GetByListener takes its read lock two times, so mochi deadlocks when it
	// closes its clients while one of them leaves. Every client must be gone first.
	l.serving.Wait()
	closeClients(l.cfg.ID)
}

// brokerOptions selects what a test broker requires of its clients.
type brokerOptions struct {
	// tls makes the listener serve TLS.
	tls *tls.Config
	// username and password, when set, are the only credentials the broker accepts.
	username, password string
	// refused holds the topic filters that an open broker refuses to subscribe to.
	refused []string
	port    int
}

// startBroker starts an open broker on port, or on a free port when port is zero. It
// stops the broker at the end of the spec.
func startBroker(port int) *testBroker {
	GinkgoHelper()
	return startBrokerWith(brokerOptions{port: port})
}

func startBrokerWith(opts brokerOptions) *testBroker {
	GinkgoHelper()
	b := &testBroker{}
	b.server = mochi.New(&mochi.Options{
		InlineClient: true,
		Logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if opts.username == "" {
		Expect(b.server.AddHook(&openHook{refused: opts.refused}, nil)).To(Succeed())
	} else {
		Expect(b.server.AddHook(new(auth.Hook), &auth.Options{
			Ledger: &auth.Ledger{Auth: auth.AuthRules{{
				Username: auth.RString(opts.username),
				Password: auth.RString(opts.password),
				Allow:    true,
			}}},
		})).To(Succeed())
	}
	Expect(b.server.AddHook(&connectHook{broker: b}, nil)).To(Succeed())
	tcp := newTCPListener(listeners.Config{
		ID:        "tcp",
		Address:   net.JoinHostPort("127.0.0.1", strconv.Itoa(opts.port)),
		TLSConfig: opts.tls,
	})
	b.listener = tcp
	Expect(b.server.AddListener(tcp)).To(Succeed())
	Expect(b.server.Serve()).To(Succeed())
	_, portText := MustSucceed2(net.SplitHostPort(tcp.Address()))
	b.port = MustSucceed(strconv.Atoi(portText))
	DeferCleanup(b.stop)
	return b
}

// stop closes the broker. It is safe to call more than once.
func (b *testBroker) stop() {
	GinkgoHelper()
	if b.server == nil {
		return
	}
	Expect(b.server.Close()).To(Succeed())
	Expect(b.listener.closeErr).ToNot(HaveOccurred())
	b.server = nil
}

var _ = Describe("Test broker", func() {
	It("Should stop while clients keep dialing", func(ctx SpecContext) {
		for range 50 {
			b := startBroker(0)
			addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(b.port))
			dialCtx, stopDialing := context.WithCancel(ctx)
			var connected sync.Once
			dialing, dialed := make(chan struct{}), make(chan struct{})
			go func() {
				defer GinkgoRecover()
				defer close(dialed)
				var d net.Dialer
				for dialCtx.Err() == nil {
					c, err := d.DialContext(dialCtx, "tcp", addr)
					if err != nil {
						continue
					}
					connected.Do(func() { close(dialing) })
					// The broker drops a connection whose first packet is not CONNECT.
					// Closing second leaves no local port in TIME_WAIT.
					_, err = c.Write([]byte{0xE0, 0})
					for buf := make([]byte, 1); err == nil; {
						_, err = c.Read(buf)
					}
					Expect(c.Close()).To(Succeed())
				}
			}()
			Eventually(dialing).Should(BeClosed())
			b.stop()
			stopDialing()
			Eventually(dialed).Should(BeClosed())
		}
	})

	It("Should stop with a connection that never sends CONNECT", func() {
		b := startBroker(0)
		c := MustSucceed(net.Dial(
			"tcp",
			net.JoinHostPort("127.0.0.1", strconv.Itoa(b.port)),
		))
		DeferCleanup(c.Close)
		Eventually(func() bool {
			return slices.ContainsFunc(
				gleak.Goroutines(),
				func(g gleak.Goroutine) bool {
					return strings.Contains(
						g.Backtrace,
						"(*Server).readConnectionPacket",
					)
				},
			)
		}).Should(BeTrue())
		stopped := make(chan struct{})
		go func() {
			defer GinkgoRecover()
			defer close(stopped)
			b.stop()
		}()
		Eventually(stopped).Should(BeClosed())
	})
})

func (b *testBroker) publish(topic, payload string, retained bool) {
	GinkgoHelper()
	Expect(b.server.Publish(topic, []byte(payload), retained, 0)).To(Succeed())
}

// collect records every message that matches filter.
func (b *testBroker) collect(filter string) {
	GinkgoHelper()
	Expect(b.server.Subscribe(filter, 1, func(
		_ *mochi.Client,
		_ packets.Subscription,
		pk packets.Packet,
	) {
		b.mu.Lock()
		defer b.mu.Unlock()
		b.messages = append(b.messages, received{
			topic:    pk.TopicName,
			payload:  string(pk.Payload),
			retained: pk.FixedHeader.Retain,
		})
	})).To(Succeed())
}

// connectedIDs returns the client ID of each connection so far, in order.
func (b *testBroker) connectedIDs() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]string(nil), b.clientIDs...)
}

// connectedKeepAlives returns the keep-alive interval in seconds of each connection so
// far, in order.
func (b *testBroker) connectedKeepAlives() []uint16 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]uint16(nil), b.keepAlives...)
}

func (b *testBroker) collected() []received {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]received(nil), b.messages...)
}

// retained returns the payloads of the retained messages that match filter.
func (b *testBroker) retained(filter string) []string {
	var out []string
	for _, pk := range b.server.Topics.Messages(filter) {
		out = append(out, string(pk.Payload))
	}
	return out
}

// clientCount returns the count of clients connected over the network.
func (b *testBroker) clientCount() int {
	count := 0
	for _, cl := range b.server.Clients.GetAll() {
		if !cl.Net.Inline && !cl.Closed() {
			count++
		}
	}
	return count
}

// subscriptions returns the count of subscriptions that network clients hold.
func (b *testBroker) subscriptions() int {
	return int(atomic.LoadInt64(&b.server.Info.Subscriptions))
}
