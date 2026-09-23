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
	"crypto/tls"
	"io"
	"log/slog"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	mochi "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/listeners"
	"github.com/mochi-mqtt/server/v2/packets"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	calcgraph "github.com/synnaxlabs/synnax/pkg/service/channel/calculation/graph"
	"github.com/synnaxlabs/synnax/pkg/service/device"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/gorp"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/types"
)

var (
	db         *gorp.DB
	rackSvc    *rack.Service
	deviceSvc  *device.Service
	channelSvc *channel.Service
	framerSvc  *framer.Service
	statusSvc  *status.Service
)

func TestMQTT(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service MQTT Suite")
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := mock.NewNode(ctx)
	db = node.DB
	otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
	searchIdx := MustOpen(search.OpenIndex())
	groupSvc := MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Search:   searchIdx,
	}))
	labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    groupSvc,
		Search:   searchIdx,
	}))
	statusSvc = MustOpen(status.OpenService(ctx, status.ServiceConfig{
		Ontology: otg,
		DB:       db,
		Group:    groupSvc,
		Label:    labelSvc,
		Search:   searchIdx,
	}))
	rackSvc = MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
		DB:           db,
		Ontology:     otg,
		Group:        groupSvc,
		HostProvider: mock.NewStaticHostProvider(1),
		Status:       statusSvc,
		Search:       searchIdx,
	}))
	deviceSvc = MustOpen(device.OpenService(ctx, device.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    groupSvc,
		Status:   statusSvc,
		Rack:     rackSvc,
		Search:   searchIdx,
	}))
	channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
		Channel:      node.Channel,
		DB:           db,
		HostProvider: node.Cluster,
		Ontology:     otg,
		Group:        groupSvc,
		Search:       searchIdx,
		Status:       statusSvc,
		// The suite creates more channels than a Core with no license key allows.
		IntOverflowCheck: func(types.Uint20) error { return nil },
	}))
	channelGraph := MustOpen(calcgraph.Open(ctx, calcgraph.Config{
		DB:      db,
		Channel: channelSvc,
		Status:  statusSvc,
	}))
	framerSvc = MustOpen(framer.OpenService(ctx, framer.ServiceConfig{
		DB:           db,
		Framer:       node.Framer,
		Channel:      channelSvc,
		ChannelGraph: channelGraph,
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
	server *mochi.Server
	port   int
	mu     sync.Mutex
	// messages holds what the collector of the broker received.
	messages []received
}

// brokerOptions selects what a test broker requires of its clients.
type brokerOptions struct {
	// tls makes the listener serve TLS.
	tls *tls.Config
	// username and password, when set, are the only credentials the broker accepts.
	username, password string
	port               int
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
		Expect(b.server.AddHook(new(auth.AllowHook), nil)).To(Succeed())
	} else {
		Expect(b.server.AddHook(new(auth.Hook), &auth.Options{
			Ledger: &auth.Ledger{Auth: auth.AuthRules{{
				Username: auth.RString(opts.username),
				Password: auth.RString(opts.password),
				Allow:    true,
			}}},
		})).To(Succeed())
	}
	tcp := listeners.NewTCP(listeners.Config{
		ID:        "tcp",
		Address:   net.JoinHostPort("127.0.0.1", strconv.Itoa(opts.port)),
		TLSConfig: opts.tls,
	})
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
	b.server = nil
}

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
