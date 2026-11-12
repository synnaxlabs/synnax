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
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	calcgraph "github.com/synnaxlabs/synnax/pkg/service/channel/calculation/graph"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/kafka"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	taskconfig "github.com/synnaxlabs/synnax/pkg/service/task/config"
	"github.com/synnaxlabs/synnax/pkg/storage"
	storagetestutil "github.com/synnaxlabs/synnax/pkg/storage/testutil"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/hookkv"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kfake"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kmsg"
)

var (
	// hookedKV stores the node, so a spec can intercept the commits of db.
	hookedKV *hookkv.DB
	// faultyFS holds the time-series data, so a spec can hold a write in progress.
	faultyFS   *FaultyFS
	db         *gorp.DB
	statusSvc  *status.Service
	rackSvc    *rack.Service
	deviceSvc  *device.Service
	channelSvc *channel.Service
	framerSvc  *framer.Service
	taskSvc    *task.Service
	cluster    *kfake.Cluster
)

func TestKafka(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Kafka Suite", SuiteConfig())
}

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	st := DeferClose(storagetestutil.OpenLayer(ctx))
	hookedKV = hookkv.Wrap(st.KV)
	faultyFS = WrapFaultyFS(xfs.NewMem())
	node := testutil.NewNode(ctx, distribution.LayerConfig{
		Storage: &storage.Layer{
			KV: hookedKV,
			TS: DeferClose(MustSucceed(ts.Open(ctx, ts.Config{FS: faultyFS}))),
		},
	})
	db = node.DB
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
		Channel: node.Channel,
		DB:      db,
		HostKey: node.Identity.HostKey,
		Status:  statusSvc,
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
	kafkaSvc := MustOpen(kafka.OpenService(ctx, kafka.ServiceConfig{
		DB:      db,
		Channel: channelSvc,
	}))
	taskSvc = MustOpen(task.OpenService(ctx, task.ServiceConfig{
		DB:      db,
		Rack:    rackSvc,
		Status:  statusSvc,
		Channel: channelSvc,
		ImEx:    imex.NewService(),
		Configs: MustSucceed(taskconfig.NewRegistry(kafkaSvc.Stores()...)),
	}))
	cluster = MustSucceed(kfake.NewCluster(
		kfake.NumBrokers(1),
		kfake.AllowAutoTopicCreation(),
	))
	DeferCleanup(cluster.Close)
})

var _ = ShouldNotLeakGoroutinesPerSpec()

var errCommit = errors.New("commit failed")

// refuse fails the commit it intercepts.
func refuse(func() error) error { return errCommit }

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

// readField returns a field with base of the kind that reads into a channel of dt. A
// timestamp field reads unix_ns.
func readField(base kafka.BaseReadField, dt telem.DataType) kafka.ReadField {
	GinkgoHelper()
	kind, ok := payload.KindOf(dt)
	Expect(ok).To(BeTrue())
	switch kind {
	case payload.KindFloat:
		return kafka.ReadField{Variant: kafka.FloatReadField{BaseReadField: base}}
	case payload.KindInteger:
		return kafka.ReadField{Variant: kafka.IntegerReadField{BaseReadField: base}}
	case payload.KindUnsigned:
		return kafka.ReadField{Variant: kafka.UnsignedReadField{BaseReadField: base}}
	case payload.KindString:
		return kafka.ReadField{Variant: kafka.StringReadField{BaseReadField: base}}
	case payload.KindBoolean:
		return kafka.ReadField{Variant: kafka.BooleanReadField{BaseReadField: base}}
	case payload.KindTimestamp:
		return timestampField(base, payload.TimeFormatUnixNs)
	case payload.KindJSON:
		return kafka.ReadField{Variant: kafka.JsonReadField{BaseReadField: base}}
	case payload.KindUUID:
		return kafka.ReadField{Variant: kafka.UuidReadField{BaseReadField: base}}
	default:
		return kafka.ReadField{Variant: kafka.BytesReadField{BaseReadField: base}}
	}
}

// timestampField returns a timestamp field with base that reads format.
func timestampField(
	base kafka.BaseReadField,
	format payload.TimeFormat,
) kafka.ReadField {
	return kafka.ReadField{Variant: kafka.TimestampReadField{
		BaseReadField: base,
		TimestampRead: payload.TimestampRead{TimeFormat: format},
	}}
}

// toJSON converts v, marshaled with opts, into the map a stored task config or device
// holds.
func toJSON(v any, opts ...json.Options) msgpack.EncodedJSON {
	GinkgoHelper()
	b := MustSucceed(json.Marshal(v, opts...))
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
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return deviceSvc.NewWriter(tx).Create(ctx, &dev)
	})).To(Succeed())
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
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return channelSvc.NewWriter(tx).Create(ctx, &index)
	})).To(Succeed())
	data = channel.Channel{
		Name:       "data_" + suffix,
		DataType:   dt,
		LocalIndex: index.LocalKey,
	}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return channelSvc.NewWriter(tx).Create(ctx, &data)
	})).To(Succeed())
	return index, data
}

// newTask stores and returns a task of typ holding cfg. It leaves out each zero struct
// field, so the factory's decode gives that field its default as it does an absent one.
// To send an explicit zero, pass cfg as a map. newTask skips task.Writer, which writes
// a status of its own, so a spec sees only the statuses the factory writes.
func newTask(ctx context.Context, typ string, cfg any) task.Task {
	GinkgoHelper()
	t := task.Task{
		Key:    uuid.New(),
		Rack:   rackSvc.EmbeddedKey,
		Name:   "test-" + typ,
		Type:   typ,
		Config: toJSON(cfg, json.OmitZeroStructFields(true)),
	}
	Expect(gorp.NewCreate[task.Key, task.Task]().Entry(&t).Exec(ctx, db)).
		To(Succeed())
	DeferCleanup(func(ctx SpecContext) {
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return taskSvc.NewWriter(tx).Delete(ctx, t.Key, true)
		})).To(Succeed())
	})
	return t
}

// taskStatus retrieves the status of t.
func taskStatus(ctx context.Context, t task.Task) task.Status {
	GinkgoHelper()
	var stat task.Status
	Expect(statusSvc.NewRetrieve[task.StatusDetails]().
		Where(status.MatchOwners[task.StatusDetails](t.ResourceID())).
		Entry(&stat).
		Exec(ctx, db)).To(Succeed())
	return stat
}

// deviceStatus retrieves the status of dev.
func deviceStatus(ctx context.Context, dev device.Device) device.Status {
	GinkgoHelper()
	var stat device.Status
	Expect(statusSvc.NewRetrieve[device.StatusDetails]().
		Where(status.MatchOwners[device.StatusDetails](dev.ResourceID())).
		Entry(&stat).
		Exec(ctx, db)).To(Succeed())
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

// newSlowCluster starts a cluster that holds topic with one record and drops every
// fetch.
func newSlowCluster(ctx context.Context, topic string) *kfake.Cluster {
	GinkgoHelper()
	slow := MustSucceed(
		kfake.NewCluster(kfake.NumBrokers(1), kfake.SeedTopics(1, topic)),
	)
	DeferCleanup(slow.Close)
	cl := newKafkaClient(kgo.SeedBrokers(slow.ListenAddrs()...))
	produceJSON(ctx, cl, topic, "", map[string]int{"v": 1})
	slow.ControlKey(int16(kmsg.Fetch), func(kmsg.Request) (kmsg.Response, error, bool) {
		slow.KeepControl()
		return nil, errors.New("fetch cut"), true
	})
	return slow
}

// newLeaderlessCluster starts a cluster that holds topic with one record and answers
// each offset listing with no leader for each partition.
func newLeaderlessCluster(ctx context.Context, topic string) *kfake.Cluster {
	GinkgoHelper()
	c := MustSucceed(
		kfake.NewCluster(kfake.NumBrokers(1), kfake.SeedTopics(1, topic)),
	)
	DeferCleanup(c.Close)
	cl := newKafkaClient(kgo.SeedBrokers(c.ListenAddrs()...))
	produceJSON(ctx, cl, topic, "", map[string]int{"v": 1})
	c.ControlKey(
		int16(kmsg.ListOffsets),
		func(kreq kmsg.Request) (kmsg.Response, error, bool) {
			c.KeepControl()
			req := kreq.(*kmsg.ListOffsetsRequest)
			res := req.ResponseKind().(*kmsg.ListOffsetsResponse)
			for _, rt := range req.Topics {
				t := kmsg.NewListOffsetsResponseTopic()
				t.Topic = rt.Topic
				for _, rp := range rt.Partitions {
					p := kmsg.NewListOffsetsResponseTopicPartition()
					p.Partition = rp.Partition
					p.ErrorCode = kerr.LeaderNotAvailable.Code
					t.Partitions = append(t.Partitions, p)
				}
				res.Topics = append(res.Topics, t)
			}
			return res, nil, true
		},
	)
	return c
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
