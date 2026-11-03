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
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func newFactory(queueSize int) driver.Factory {
	GinkgoHelper()
	return MustOpen(mqtt.OpenFactory(mqtt.FactoryConfig{
		Device:  deviceSvc,
		Channel: channelSvc,
		Framer:  framerSvc,
		DB:      db, Status: statusSvc,
		QueueSize: queueSize,
	}))
}

// createRack creates a rack for the devices and tasks of one spec.
func createRack(ctx context.Context) rack.Key {
	GinkgoHelper()
	r := rack.Rack{Name: "mqtt-rack"}
	Expect(rackSvc.NewWriter(nil).Create(ctx, &r)).To(Succeed())
	return r.Key
}

// createBrokerDevice creates a broker device that points at port on the local host.
// The properties of the device name no port when port is zero.
func createBrokerDevice(
	ctx context.Context,
	rackKey rack.Key,
	port int,
	properties msgpack.EncodedJSON,
) device.Device {
	GinkgoHelper()
	if properties == nil {
		properties = msgpack.EncodedJSON{}
	}
	if port != 0 {
		properties["port"] = port
	}
	dev := device.Device{
		Key:        uuid.New().String(),
		Rack:       rackKey,
		Location:   "127.0.0.1",
		Name:       "Test Broker",
		Make:       mqtt.Make,
		Model:      "broker",
		Properties: properties,
	}
	writeDevice(ctx, &dev)
	return dev
}

// writeDevice creates or updates dev in one transaction, as the API does, so the
// device and its status change together.
func writeDevice(ctx context.Context, dev *device.Device) {
	GinkgoHelper()
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return deviceSvc.NewWriter(tx).Create(ctx, dev)
	})).To(Succeed())
}

// editDevice replaces the properties of dev.
func editDevice(ctx context.Context, dev device.Device, props msgpack.EncodedJSON) {
	GinkgoHelper()
	dev.Properties = props
	writeDevice(ctx, &dev)
}

// createIndexed creates an index channel and one data channel of each data type.
func createIndexed(
	ctx context.Context,
	dataTypes ...telem.DataType,
) (channel.Channel, []channel.Channel) {
	GinkgoHelper()
	return createIndexedOn(ctx, host, dataTypes...)
}

// createIndexedOn is createIndexed on the Core that leaseholder names.
func createIndexedOn(
	ctx context.Context,
	leaseholder node.Key,
	dataTypes ...telem.DataType,
) (channel.Channel, []channel.Channel) {
	GinkgoHelper()
	w := channelSvc.NewWriter(nil)
	idx := channel.Channel{
		Name:        UniqueChannelName(),
		DataType:    telem.TimestampT,
		IsIndex:     true,
		Leaseholder: leaseholder,
	}
	Expect(w.Create(ctx, &idx)).To(Succeed())
	data := make([]channel.Channel, len(dataTypes))
	for i, dt := range dataTypes {
		data[i] = channel.Channel{
			Name:        UniqueChannelName(),
			DataType:    dt,
			Leaseholder: leaseholder,
			LocalIndex:  idx.LocalKey,
		}
		Expect(w.Create(ctx, &data[i])).To(Succeed())
	}
	// A peer's channels reach the host by gossip.
	keys := append(channel.Keys{idx.Key()}, channel.KeysFromChannels(data)...)
	Eventually(func(g Gomega) {
		var found []channel.Channel
		g.Expect(channelSvc.NewRetrieve().
			Where(channel.MatchKeys(keys...)).
			Entries(&found).
			Exec(ctx, nil)).To(Succeed())
		g.Expect(found).To(HaveLen(len(keys)))
	}).Should(Succeed())
	return idx, data
}

func createVirtual(ctx context.Context, dt telem.DataType) channel.Channel {
	GinkgoHelper()
	ch := channel.Channel{Name: UniqueChannelName(), DataType: dt, Virtual: true}
	Expect(channelSvc.NewWriter(nil).Create(ctx, &ch)).To(Succeed())
	return ch
}

func newTask(
	rackKey rack.Key,
	taskType string,
	cfg msgpack.EncodedJSON,
) task.Task {
	return task.Task{
		Key:    uuid.New(),
		Name:   "mqtt-task",
		Type:   taskType,
		Rack:   rackKey,
		Config: cfg,
	}
}

// configure configures t with a start command key and stops the task at the end of
// the spec.
func configure(ctx context.Context, f driver.Factory, t task.Task) driver.Task {
	GinkgoHelper()
	configured := MustSucceed(f.ConfigureTask(ctx, t, "configure"))
	DeferCleanup(func() { Expect(configured.Stop(false)).To(Succeed()) })
	return configured
}

// taskStatus returns the status of t, or the zero status when it has none.
func taskStatus(ctx context.Context, t task.Task) task.Status {
	GinkgoHelper()
	var stat task.Status
	Expect(errors.Skip(
		statusSvc.NewRetrieve[task.StatusDetails]().
			Where(status.MatchKeys[task.StatusDetails](t.OntologyID().String())).
			Entry(&stat).
			Exec(ctx, nil),
		query.ErrNotFound,
	)).To(Succeed())
	return stat
}

// answer runs cmd on a scan task and returns the status that answers it.
func answer(
	ctx context.Context,
	configured driver.Task,
	t task.Task,
	cmd task.Command,
) task.Status {
	GinkgoHelper()
	Expect(configured.Exec(ctx, cmd)).To(Succeed())
	return answered(ctx, t, cmd.Key)
}

// answered waits for the status of t that answers the command with key cmdKey, and
// returns it.
func answered(ctx context.Context, t task.Task, cmdKey string) task.Status {
	GinkgoHelper()
	var stat task.Status
	Eventually(func(g Gomega) {
		stat = taskStatus(ctx, t)
		g.Expect(stat.Details.Cmd).To(Equal(cmdKey))
	}).WithTimeout(10 * time.Second).Should(Succeed())
	return stat
}

// testStreamer receives the frames written to a set of channels.
type testStreamer struct {
	responses <-chan framer.StreamerResponse
	// probe is the channel that tells when the streamer connects.
	probe channel.Key
}

// stream opens a streamer on keys that receives every frame written after it returns.
// The streamer closes at the end of the spec.
func stream(ctx context.Context, keys channel.Keys) testStreamer {
	GinkgoHelper()
	probe := createVirtual(ctx, telem.Int64T).Key()
	streamer := MustSucceed(framerSvc.NewStreamer(ctx, framer.StreamerConfig{
		Keys: append(keys, probe),
	}))
	var (
		requests  = confluence.NewStream[framer.StreamerRequest]()
		responses = confluence.NewStream[framer.StreamerResponse](1024)
	)
	streamer.InFrom(requests)
	streamer.OutTo(responses)
	sCtx, cancel := signal.Isolated()
	streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
	DeferCleanup(func() {
		requests.Close()
		confluence.Drain(responses)
		cancel()
		Expect(sCtx.Wait()).To(Succeed())
	})
	// A streamer misses the frames written before it connects.
	w := MustSucceed(framerSvc.OpenWriter(ctx, framer.WriterConfig{
		Keys:  channel.Keys{probe},
		Start: telem.Now(),
	}))
	Eventually(func(g Gomega) {
		g.Expect(w.Write(frame.NewUnary(probe, telem.NewSeriesV[int64](1)))).
			To(BeTrue())
		g.Expect(responses.Outlet()).To(Receive())
	}).Should(Succeed())
	Expect(w.Close()).To(Succeed())
	return testStreamer{responses: responses.Outlet(), probe: probe}
}

// receive returns the next frame written to the channels of s.
func (s testStreamer) receive() framer.Frame {
	GinkgoHelper()
	for {
		var res framer.StreamerResponse
		Eventually(s.responses).Should(Receive(&res))
		if res.Frame.Get(s.probe).Len() == 0 {
			return res.Frame
		}
	}
}

// deviceStatus returns the status of dev, or the zero status when it has none.
func deviceStatus(ctx context.Context, dev device.Device) device.Status {
	GinkgoHelper()
	var stat device.Status
	Expect(errors.Skip(
		statusSvc.NewRetrieve[device.StatusDetails]().
			Where(status.MatchKeys[device.StatusDetails](dev.OntologyID().String())).
			Entry(&stat).
			Exec(ctx, nil),
		query.ErrNotFound,
	)).To(Succeed())
	return stat
}

// stored returns every sample stored in ch.
func stored(ctx context.Context, ch channel.Channel) telem.Series {
	GinkgoHelper()
	iter := MustSucceed(framerSvc.OpenIterator(ctx, framer.IteratorConfig{
		Keys:   []channel.Key{ch.Key()},
		Bounds: telem.TimeRangeMax,
	}))
	defer func() { Expect(iter.Close()).To(Succeed()) }()
	out := telem.Series{DataType: ch.DataType}
	next := func() bool { return iter.Next(iterator.AutoSpan) }
	for ok := iter.SeekFirst() && next(); ok; ok = next() {
		for _, s := range iter.Value().Get(ch.Key()).Series {
			out.Data = append(out.Data, s.Data...)
		}
	}
	return out
}
