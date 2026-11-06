// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kafka_test

import (
	"context"
	"encoding/json/v2"
	"net"
	"sync/atomic"
	"testing"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	calcgraph "github.com/synnaxlabs/synnax/pkg/service/channel/calculation/graph"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/kafka"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/twmb/franz-go/pkg/kfake"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kmsg"
)

var (
	db         *gorp.DB
	statusSvc  *status.Service
	rackSvc    *rack.Service
	deviceSvc  *device.Service
	channelSvc *channel.Service
	framerSvc  *framer.Service
	cluster    *kfake.Cluster
)

func TestKafka(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Kafka Suite", SuiteConfig())
}

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := testutil.NewNode(ctx)
	db = node.DB
	otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
	groupSvc := MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB:       db,
		Ontology: otg,
	}))
	labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    groupSvc,
	}))
	statusSvc = MustOpen(status.OpenService(ctx, status.ServiceConfig{
		Ontology: otg,
		DB:       db,
		Group:    groupSvc,
		Label:    labelSvc,
	}))
	rackSvc = MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
		DB:           db,
		Ontology:     otg,
		Group:        groupSvc,
		HostProvider: testutil.NewStaticHostProvider(1),
		Status:       statusSvc,
		DriverRole:   MustSucceed(racktestutil.CreateDriverRole(ctx, otg)),
	}))
	deviceSvc = MustOpen(device.OpenService(ctx, device.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    groupSvc,
		Status:   statusSvc,
		Rack:     rackSvc,
	}))
	channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
		Channel:      node.Channel,
		DB:           db,
		HostProvider: node.Cluster,
		Ontology:     otg,
		Group:        groupSvc,
		Status:       statusSvc,
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
	cluster = MustSucceed(kfake.NewCluster(
		kfake.NumBrokers(1),
		kfake.AllowAutoTopicCreation(),
	))
	DeferCleanup(cluster.Close)
})

var _ = ShouldNotLeakGoroutinesPerSpec()

// gatedDB is a kv.DB that, once gated, holds the next commit until release closes.
type gatedDB struct {
	kv.DB
	gated   atomic.Bool
	held    chan struct{}
	release chan struct{}
}

func (d *gatedDB) OpenTx() kv.Tx { return gatedTx{Tx: d.DB.OpenTx(), db: d} }

type gatedTx struct {
	kv.Tx
	db *gatedDB
}

func (t gatedTx) Commit(ctx context.Context, opts ...any) error {
	GinkgoHelper()
	// A task commits on its own goroutine, where a failed assertion needs a recover.
	defer GinkgoRecover()
	if t.db.gated.CompareAndSwap(true, false) {
		close(t.db.held)
		Eventually(t.db.release).WithTimeout(15 * time.Second).Should(BeClosed())
	}
	return t.Tx.Commit(ctx, opts...)
}

// newFactory opens a Kafka factory over the suite's services.
func newFactory(cfgs ...kafka.FactoryConfig) driver.Factory {
	GinkgoHelper()
	return MustSucceed(kafka.NewFactory(append([]kafka.FactoryConfig{{
		DB:           db,
		Device:       deviceSvc,
		Channel:      channelSvc,
		Framer:       framerSvc,
		Status:       statusSvc,
		PingTimeout:  2 * telem.Second,
		FetchMaxWait: 100 * telem.Millisecond,
	}}, cfgs...)...))
}

// clusterProperties are device properties pointing at the suite's fake cluster.
func clusterProperties() kafka.Properties {
	return kafka.Properties{Brokers: cluster.ListenAddrs()}
}

// toJSON converts v into the map a stored task config or device holds.
func toJSON(v any) msgpack.EncodedJSON {
	GinkgoHelper()
	b := MustSucceed(json.Marshal(v))
	var m msgpack.EncodedJSON
	Expect(json.Unmarshal(b, &m)).To(Succeed())
	return m
}

// createDevice creates a Kafka cluster device with props on the embedded rack.
func createDevice(ctx context.Context, props kafka.Properties) device.Device {
	GinkgoHelper()
	dev := device.Device{
		Key:        uuid.New().String(),
		Rack:       rackSvc.EmbeddedKey,
		Location:   "kafka",
		Make:       kafka.Make,
		Model:      "cluster",
		Name:       "cluster-" + uuid.New().String()[:8],
		Configured: true,
		Properties: toJSON(props),
	}
	Expect(deviceSvc.NewWriter(nil).Create(ctx, &dev)).To(Succeed())
	return dev
}

// createIndexed creates an index channel and one data channel of dt on it.
func createIndexed(
	ctx context.Context,
	dt telem.DataType,
) (index, data channel.Channel) {
	GinkgoHelper()
	suffix := uuid.New().String()[:8]
	index = channel.Channel{
		Name:     "idx_" + suffix,
		DataType: telem.TimestampT,
		IsIndex:  true,
	}
	Expect(channelSvc.NewWriter(nil).Create(ctx, &index)).To(Succeed())
	data = channel.Channel{
		Name:       "data_" + suffix,
		DataType:   dt,
		LocalIndex: index.LocalKey,
	}
	Expect(channelSvc.NewWriter(nil).Create(ctx, &data)).To(Succeed())
	return index, data
}

// newTask returns a stored task row of typ holding cfg.
func newTask(typ string, cfg any) task.Task {
	GinkgoHelper()
	return task.Task{
		Key:    uuid.New(),
		Rack:   rackSvc.EmbeddedKey,
		Name:   "test-" + typ,
		Type:   typ,
		Config: toJSON(cfg),
	}
}

// taskStatus retrieves the status of t.
func taskStatus(ctx context.Context, t task.Task) task.Status {
	GinkgoHelper()
	var stat task.Status
	Expect(statusSvc.NewRetrieve[task.StatusDetails]().
		Where(status.MatchKeys[task.StatusDetails](t.OntologyID().String())).
		Entry(&stat).
		Exec(ctx, nil)).To(Succeed())
	return stat
}

// deviceStatus retrieves the status of dev.
func deviceStatus(ctx context.Context, dev device.Device) device.Status {
	GinkgoHelper()
	var stat device.Status
	Expect(statusSvc.NewRetrieve[device.StatusDetails]().
		Where(status.MatchKeys[device.StatusDetails](dev.OntologyID().String())).
		Entry(&stat).
		Exec(ctx, nil)).To(Succeed())
	return stat
}

// newKafkaClient opens a franz-go client on the suite's fake cluster.
func newKafkaClient(opts ...kgo.Opt) *kgo.Client {
	GinkgoHelper()
	// A short fetch wait lets the fake broker release a parked fetch, and its
	// connection goroutines, before the leak check runs.
	cl := MustSucceed(kgo.NewClient(append([]kgo.Opt{
		kgo.SeedBrokers(cluster.ListenAddrs()...),
		kgo.FetchMaxWait(100 * time.Millisecond),
	}, opts...)...))
	DeferCleanup(cl.Close)
	return cl
}

// produceJSON produces one JSON record with key to topic and waits for the ack.
func produceJSON(ctx context.Context, cl *kgo.Client, topic, key string, doc any) {
	GinkgoHelper()
	body := MustSucceed(json.Marshal(doc))
	rec := &kgo.Record{Topic: topic, Value: body}
	if key != "" {
		rec.Key = []byte(key)
	}
	Expect(cl.ProduceSync(ctx, rec).FirstErr()).To(Succeed())
}

// produceInTransaction produces count JSON records to topic in one transaction and
// commits it. The commit writes a control record after them.
func produceInTransaction(ctx context.Context, topic string, count int) {
	GinkgoHelper()
	cl := newKafkaClient(kgo.TransactionalID(uniqueTopic()))
	Expect(cl.BeginTransaction()).To(Succeed())
	for i := range count {
		produceJSON(ctx, cl, topic, "", map[string]int{"v": i})
	}
	Expect(cl.EndTransaction(ctx, kgo.TryCommit)).To(Succeed())
}

// newSlowCluster starts a cluster that holds topic with one record and holds every
// fetch until the spec ends.
func newSlowCluster(ctx context.Context, topic string) *kfake.Cluster {
	GinkgoHelper()
	slow := MustSucceed(
		kfake.NewCluster(kfake.NumBrokers(1), kfake.SeedTopics(1, topic)),
	)
	DeferCleanup(slow.Close)
	cl := newKafkaClient(kgo.SeedBrokers(slow.ListenAddrs()...))
	produceJSON(ctx, cl, topic, "", map[string]int{"v": 1})
	release := make(chan struct{})
	DeferCleanup(func() { close(release) })
	slow.ControlKey(int16(kmsg.Fetch), func(kmsg.Request) (kmsg.Response, error, bool) {
		slow.KeepControl()
		slow.SleepControl(func() { <-release })
		return nil, nil, false
	})
	return slow
}

// stepsOf returns the steps of the connection test that stat answers.
func stepsOf(stat task.Status) []task.TestStep {
	GinkgoHelper()
	var res task.TestResult
	Expect(stat.Details.Data.Unmarshal(&res)).To(Succeed())
	return res.Steps
}

// stages returns the stage and variant of each step.
func stages(steps []task.TestStep) []string {
	out := make([]string, len(steps))
	for i, s := range steps {
		out[i] = s.Stage + ":" + string(s.Variant)
	}
	return out
}

// closedAddress returns a local address that refuses connections.
func closedAddress() string {
	GinkgoHelper()
	l := MustSucceed(net.Listen("tcp", "127.0.0.1:0"))
	addr := l.Addr().String()
	Expect(l.Close()).To(Succeed())
	return addr
}

// uniqueTopic returns a topic name no other spec uses.
func uniqueTopic() string { return "topic-" + uuid.New().String()[:8] }

// topicID returns the ID the cluster gave topic. Fetch requests carry the ID in place
// of the name.
func topicID(ctx context.Context, cl *kgo.Client, topic string) [16]byte {
	GinkgoHelper()
	req := kmsg.NewPtrMetadataRequest()
	t := kmsg.NewMetadataRequestTopic()
	t.Topic = new(topic)
	req.Topics = append(req.Topics, t)
	res := MustSucceed(req.RequestWith(ctx, cl))
	Expect(res.Topics).To(HaveLen(1))
	return res.Topics[0].TopicID
}

// cutConnections closes each connection that sends a request to broker until restore
// runs. broker keeps its state, unlike a kfake.DataDir restart, which fails on Windows
// because kfake renames its files while they are open.
func cutConnections(broker *kfake.Cluster) (restore func()) {
	var cut atomic.Bool
	cut.Store(true)
	broker.Control(func(kmsg.Request) (kmsg.Response, error, bool) {
		broker.KeepControl()
		if cut.Load() {
			return nil, errors.New("connection cut"), true
		}
		return nil, nil, false
	})
	return func() { cut.Store(false) }
}
