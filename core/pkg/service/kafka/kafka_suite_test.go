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
	"testing"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	calcgraph "github.com/synnaxlabs/synnax/pkg/service/channel/calculation/graph"
	"github.com/synnaxlabs/synnax/pkg/service/device"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/kafka"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/twmb/franz-go/pkg/kfake"
	"github.com/twmb/franz-go/pkg/kgo"
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
	RunSpecs(t, "Service Kafka Suite")
}

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

// openTestStreamer streams live frames for keys. The returned close function must be
// called before the spec ends.
func openTestStreamer(ctx context.Context, keys channel.Keys) (
	responses <-chan framer.StreamerResponse,
	close func(),
) {
	GinkgoHelper()
	streamer := MustSucceed(framerSvc.NewStreamer(ctx, framer.StreamerConfig{
		Keys:        keys,
		SendOpenAck: true,
	}))
	requests, res := confluence.Attach(streamer, 100)
	sCtx, cancel := signal.Isolated()
	closer := signal.NewHardShutdown(sCtx, cancel)
	streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
	Eventually(res.Outlet()).Should(Receive())
	return res.Outlet(), func() {
		requests.Close()
		confluence.Drain(res)
		Expect(closer.Close()).To(Succeed())
	}
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

// uniqueTopic returns a topic name no other spec uses.
func uniqueTopic() string { return "topic-" + uuid.New().String()[:8] }
