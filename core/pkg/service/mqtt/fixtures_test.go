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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/device"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func newFactory(queueSize int) driver.Factory {
	GinkgoHelper()
	return MustSucceed(mqtt.NewFactory(mqtt.FactoryConfig{
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
	properties["port"] = port
	dev := device.Device{
		Key:        uuid.New().String(),
		Rack:       rackKey,
		Location:   "127.0.0.1",
		Name:       "Test Broker",
		Make:       mqtt.Make,
		Model:      "broker",
		Properties: properties,
	}
	Expect(deviceSvc.NewWriter(nil).Create(ctx, &dev)).To(Succeed())
	return dev
}

// createIndexed creates an index channel and one data channel of each data type.
func createIndexed(
	ctx context.Context,
	dataTypes ...telem.DataType,
) (channel.Channel, []channel.Channel) {
	GinkgoHelper()
	w := channelSvc.NewWriter(nil)
	idx := channel.Channel{
		Name:     UniqueChannelName(),
		DataType: telem.TimestampT,
		IsIndex:  true,
	}
	Expect(w.Create(ctx, &idx)).To(Succeed())
	data := make([]channel.Channel, len(dataTypes))
	for i, dt := range dataTypes {
		data[i] = channel.Channel{
			Name:       UniqueChannelName(),
			DataType:   dt,
			LocalIndex: idx.LocalKey,
		}
		Expect(w.Create(ctx, &data[i])).To(Succeed())
	}
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
	err := statusSvc.NewRetrieve[task.StatusDetails]().
		Where(status.MatchKeys[task.StatusDetails](t.OntologyID().String())).
		Entry(&stat).Exec(ctx, nil)
	if !errors.Is(err, query.ErrNotFound) {
		Expect(err).ToNot(HaveOccurred())
	}
	return stat
}

// deviceStatus returns the status of dev, or the zero status when it has none.
func deviceStatus(ctx context.Context, dev device.Device) device.Status {
	GinkgoHelper()
	var stat device.Status
	err := statusSvc.NewRetrieve[device.StatusDetails]().
		Where(status.MatchKeys[device.StatusDetails](dev.OntologyID().String())).
		Entry(&stat).Exec(ctx, nil)
	if !errors.Is(err, query.ErrNotFound) {
		Expect(err).ToNot(HaveOccurred())
	}
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
