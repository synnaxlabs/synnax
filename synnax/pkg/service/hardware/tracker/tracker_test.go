// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package tracker_test

import (
	"time"

	"github.com/synnaxlabs/synnax/pkg/service/hardware/device"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/service/hardware/rack"
	"github.com/synnaxlabs/synnax/pkg/service/hardware/task"
	"github.com/synnaxlabs/synnax/pkg/service/hardware/tracker"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Tracker", Ordered, func() {
	var (
		cfg       tracker.Config
		tr        *tracker.Tracker
		rackSvc   *rack.Service
		taskSvc   *task.Service
		deviceSvc *device.Service
	)
	BeforeAll(func() {
		rackSvc = MustSucceed(rack.OpenService(ctx, rack.Config{
			DB:           dist.Storage.Gorpify(),
			Ontology:     dist.Ontology,
			Group:        dist.Group,
			HostProvider: dist.Cluster,
			Signals:      dist.Signals,
		}))
		taskSvc = MustSucceed(task.OpenService(ctx, task.Config{
			DB:           dist.Storage.Gorpify(),
			Ontology:     dist.Ontology,
			Group:        dist.Group,
			Rack:         rackSvc,
			HostProvider: dist.Cluster,
			Channel:      dist.Channel,
			Signals:      dist.Signals,
		}))
		deviceSvc = MustSucceed(device.OpenService(ctx, device.Config{
			DB:       dist.Storage.Gorpify(),
			Ontology: dist.Ontology,
			Group:    dist.Group,
			Signals:  dist.Signals,
		}))
		cfg = tracker.Config{
			DB:           dist.Storage.Gorpify(),
			Rack:         rackSvc,
			Task:         taskSvc,
			Signals:      dist.Signals,
			Channels:     dist.Channel,
			HostProvider: dist.Cluster,
			Framer:       dist.Framer,
			Device:       deviceSvc,
		}
	})
	JustBeforeEach(func() {
		tr = MustSucceed(tracker.Open(ctx, cfg))
	})
	JustAfterEach(func() {
		Expect(rackSvc.Close()).To(Succeed())
		Expect(taskSvc.Close()).To(Succeed())
		Expect(deviceSvc.Close()).To(Succeed())
		Expect(tr.Close()).To(Succeed())
	})
	Describe("Tracking Rack Updates", func() {
		It("Should add the rack to state when created", func() {
			rackKey := rack.NewKey(dist.Cluster.HostKey(), 1)
			rack := &rack.Rack{Key: rackKey, Name: "rack1"}
			Expect(cfg.Rack.NewWriter(nil).Create(ctx, rack)).To(Succeed())
			Eventually(func(g Gomega) {
				_, ok := tr.GetRack(ctx, rackKey)
				g.Expect(ok).To(BeTrue())
			}).Should(Succeed())
		})
		It("Should remove the rack from state when deleted", func() {
			rackKey := rack.NewKey(dist.Cluster.HostKey(), 1)
			rack := &rack.Rack{Key: rackKey, Name: "rack1"}
			Expect(cfg.Rack.NewWriter(nil).Create(ctx, rack)).To(Succeed())
			Eventually(func(g Gomega) {
				_, ok := tr.GetRack(ctx, rackKey)
				g.Expect(ok).To(BeTrue())
			}).Should(Succeed())
			Expect(cfg.Rack.NewWriter(nil).Delete(ctx, rackKey)).To(Succeed())
			Eventually(func(g Gomega) {
				_, ok := tr.GetRack(ctx, rackKey)
				g.Expect(ok).To(BeFalse())
			}).Should(Succeed())
		})
	})

	Describe("Tracking Task Updates", func() {
		It("Should add the task to state when created", func() {
			rack := &rack.Rack{Key: rack.NewKey(dist.Cluster.HostKey(), 1), Name: "rack1"}
			Expect(cfg.Rack.NewWriter(nil).Create(ctx, rack)).To(Succeed())
			taskKey := task.NewKey(rack.Key, 1)
			task := &task.Task{Key: taskKey, Name: "task1"}
			Expect(cfg.Task.NewWriter(nil).Create(ctx, task)).To(Succeed())
			Eventually(func(g Gomega) {
				_, ok := tr.GetTask(ctx, taskKey)
				g.Expect(ok).To(BeTrue())
			}).Should(Succeed())
		})
		It("Should remove the task from state when deleted", func() {
			rack := &rack.Rack{Name: "rack1"}
			Expect(cfg.Rack.NewWriter(nil).Create(ctx, rack)).To(Succeed())
			Eventually(func(g Gomega) {
				_, ok := tr.GetRack(ctx, rack.Key)
				g.Expect(ok).To(BeTrue())
			})
			taskKey := task.NewKey(rack.Key, 1)
			task := &task.Task{Key: taskKey, Name: "task1"}
			Expect(cfg.Task.NewWriter(nil).Create(ctx, task)).To(Succeed())
			Eventually(func(g Gomega) {
				_, ok := tr.GetTask(ctx, taskKey)
				g.Expect(ok).To(BeTrue())
			})
			Expect(cfg.Task.NewWriter(nil).Delete(ctx, taskKey, false)).To(Succeed())
			Eventually(func(g Gomega) {
				_, ok := tr.GetTask(ctx, taskKey)
				g.Expect(ok).To(BeFalse())
			}).Should(Succeed())
		})
	})

	Describe("Tracking Rack Heartbeats", func() {
		It("Should update the rack state when received", func() {
			rck := &rack.Rack{Name: "rack1"}
			Expect(cfg.Rack.NewWriter(nil).Create(ctx, rck)).To(Succeed())

			var rackStateCh channel.Channel
			Expect(dist.Channel.NewRetrieve().WhereNames("sy_rack_state").Entry(&rackStateCh).Exec(ctx, nil)).To(Succeed())

			w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
				Start: telem.Now(),
				Keys:  []channel.Key{rackStateCh.Key()},
			}))

			state := rack.State{
				Key:          rck.Key,
				Variant:      "info",
				LastReceived: telem.Now(),
				Message:      "Rack is alive",
			}

			Expect(w.Write(framer.Frame{
				Keys:   []channel.Key{rackStateCh.Key()},
				Series: []telem.Series{telem.NewStaticJSONV(state)},
			})).To(BeTrue())

			Expect(w.Close()).To(Succeed())

			Eventually(func(g Gomega) {
				r, ok := tr.GetRack(ctx, rck.Key)
				g.Expect(ok).To(BeTrue())
				g.Expect(r.State.Variant).To(Equal("info"))
				g.Expect(r.State.Message).To(Equal("Rack is alive"))
			}).Should(Succeed())
		})

		It("Should not update the state of tasks when a rack is alive", func() {
			rck := &rack.Rack{Key: rack.NewKey(dist.Cluster.HostKey(), 1), Name: "rack1"}
			Expect(cfg.Rack.NewWriter(nil).Create(ctx, rck)).To(Succeed())
			taskKey := task.NewKey(rck.Key, 1)

			var rackStateCh channel.Channel
			Expect(dist.Channel.NewRetrieve().WhereNames("sy_rack_state").Entry(&rackStateCh).Exec(ctx, nil)).To(Succeed())

			w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
				Start: telem.Now(),
				Keys:  []channel.Key{rackStateCh.Key()},
			}))

			state := rack.State{
				Key:          rck.Key,
				Variant:      "info",
				LastReceived: telem.Now(),
				Message:      "Rack is alive",
			}

			Expect(w.Write(framer.Frame{
				Keys:   []channel.Key{rackStateCh.Key()},
				Series: []telem.Series{telem.NewStaticJSONV(state)},
			})).To(BeTrue())

			Expect(w.Close()).To(Succeed())

			var taskStateCh channel.Channel
			Expect(dist.Channel.NewRetrieve().WhereNames("sy_task_state").Entry(&taskStateCh).Exec(ctx, nil)).To(Succeed())

			streamer := MustSucceed(dist.Framer.NewStreamer(ctx, framer.StreamerConfig{
				Keys: []channel.Key{taskStateCh.Key()},
			}))
			sCtx, sCancel := signal.Isolated()
			requests, responses := confluence.Attach[framer.StreamerRequest, framer.StreamerResponse](streamer)
			streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
			time.Sleep(1 * time.Millisecond)
			tsk := &task.Task{Key: taskKey, Name: "task1"}
			Expect(cfg.Task.NewWriter(nil).Create(ctx, tsk)).To(Succeed())
			Consistently(responses.Outlet()).ShouldNot(Receive())
			requests.Close()
			Eventually(responses.Outlet()).Should(BeClosed())
			sCancel()
		})
	})

	Describe("Tracking Task State", func() {
		It("Should correctly update the state of a task", func() {
			rack := &rack.Rack{Name: "rack1"}
			Expect(cfg.Rack.NewWriter(nil).Create(ctx, rack)).To(Succeed())
			tsk := &task.Task{Key: task.NewKey(rack.Key, 0), Name: "task1"}
			Expect(cfg.Task.NewWriter(nil).Create(ctx, tsk)).To(Succeed())
			var taskStateCh channel.Channel
			Expect(dist.Channel.NewRetrieve().WhereNames("sy_task_state").Entry(&taskStateCh).Exec(ctx, nil)).To(Succeed())
			w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
				Start: telem.Now(),
				Keys:  []channel.Key{taskStateCh.Key()},
			}))
			b := MustSucceed((&binary.JSONCodec{}).Encode(ctx, task.State{
				Variant: task.ErrorStateVariant,
				Task:    tsk.Key,
			}))
			Expect(w.Write(framer.Frame{
				Keys: []channel.Key{taskStateCh.Key()},
				Series: []telem.Series{{
					DataType: telem.JSONT,
					Data:     append(b, '\n'),
				}},
			})).To(BeTrue())
			Expect(w.Close()).To(Succeed())
			Eventually(func(g Gomega) {
				t, ok := tr.GetTask(ctx, tsk.Key)
				g.Expect(ok).To(BeTrue())
				g.Expect(t.Variant).To(Equal(task.ErrorStateVariant))
			}).Should(Succeed())
		})
	})

	Describe("Persisting state across restarts", func() {
		It("Should persist the state of tasks even if the tracker service is closed and reopened", func() {
			rack := &rack.Rack{Name: "rack1"}
			Expect(cfg.Rack.NewWriter(nil).Create(ctx, rack)).To(Succeed())
			tsk := &task.Task{Key: task.NewKey(rack.Key, 0), Name: "task1"}
			Expect(cfg.Task.NewWriter(nil).Create(ctx, tsk)).To(Succeed())
			var taskStateCh channel.Channel
			Expect(dist.Channel.NewRetrieve().WhereNames("sy_task_state").Entry(&taskStateCh).Exec(ctx, nil)).To(Succeed())
			w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
				Start: telem.Now(),
				Keys:  []channel.Key{taskStateCh.Key()},
			}))
			Expect(w.Write(framer.Frame{
				Keys: []channel.Key{taskStateCh.Key()},
				Series: []telem.Series{telem.NewStaticJSONV(task.State{
					Variant: task.ErrorStateVariant,
					Task:    tsk.Key,
				})},
			})).To(BeTrue())
			Expect(w.Close()).To(Succeed())
			Eventually(func(g Gomega) {
				t, ok := tr.GetTask(ctx, tsk.Key)
				g.Expect(ok).To(BeTrue())
				g.Expect(t.Variant).To(Equal(task.ErrorStateVariant))
			}).Should(Succeed())
			Expect(tr.Close()).To(Succeed())
			tr = MustSucceed(tracker.Open(ctx, cfg))
			state, ok := tr.GetTask(ctx, tsk.Key)
			Expect(ok).To(BeTrue())
			Expect(state.Variant).To(Equal(task.ErrorStateVariant))
		})
	})

	Describe("Communicating Through Task State when a Rack Has Died", func() {
		BeforeEach(func() {
			cfg.RackStateAliveThreshold = 5 * telem.Millisecond
		})
		It("Should update the state of tasks when a rack dies", func() {
			rack := &rack.Rack{Key: rack.NewKey(dist.Cluster.HostKey(), 1), Name: "rack1"}
			Expect(cfg.Rack.NewWriter(nil).Create(ctx, rack)).To(Succeed())
			taskKey := task.NewKey(rack.Key, 1)

			var taskStateCh channel.Channel
			Expect(dist.Channel.NewRetrieve().WhereNames("sy_task_state").Entry(&taskStateCh).Exec(ctx, nil)).To(Succeed())

			streamer := MustSucceed(dist.Framer.NewStreamer(ctx, framer.StreamerConfig{
				Keys: []channel.Key{taskStateCh.Key()},
			}))
			sCtx, sCancel := signal.Isolated()
			requests, responses := confluence.Attach[framer.StreamerRequest, framer.StreamerResponse](streamer)
			streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
			time.Sleep(10 * time.Millisecond)
			tsk := &task.Task{Key: taskKey, Name: "task1"}
			Expect(cfg.Task.NewWriter(nil).Create(ctx, tsk)).To(Succeed())
			Eventually(responses.Outlet()).Should(Receive())
			requests.Close()
			Eventually(responses.Outlet()).Should(BeClosed())
			sCancel()
		})
	})
	Describe("Communicating Through Task State when a Rack is Alive", func() {
		BeforeEach(func() {
			cfg.RackStateAliveThreshold = 10 * telem.Second
		})
		It("Should not update the state of tasks when a rack is alive", func() {
			rck := &rack.Rack{Key: rack.NewKey(dist.Cluster.HostKey(), 1), Name: "rack1"}
			Expect(cfg.Rack.NewWriter(nil).Create(ctx, rck)).To(Succeed())
			taskKey := task.NewKey(rck.Key, 1)
			var rackStateCh channel.Channel
			Expect(dist.Channel.NewRetrieve().WhereNames("sy_rack_state").Entry(&rackStateCh).Exec(ctx, nil)).To(Succeed())
			w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
				Start: telem.Now(),
				Keys:  []channel.Key{rackStateCh.Key()},
			}))

			state := rack.State{
				Key:          rck.Key,
				Variant:      "info",
				LastReceived: telem.Now(),
				Message:      "Rack is alive",
			}

			Expect(w.Write(framer.Frame{
				Keys:   []channel.Key{rackStateCh.Key()},
				Series: []telem.Series{telem.NewStaticJSONV(state)},
			})).To(BeTrue())

			Expect(w.Close()).To(Succeed())

			var taskStateCh channel.Channel
			Expect(dist.Channel.NewRetrieve().WhereNames("sy_task_state").Entry(&taskStateCh).Exec(ctx, nil)).To(Succeed())

			streamer := MustSucceed(dist.Framer.NewStreamer(ctx, framer.StreamerConfig{
				Keys: []channel.Key{taskStateCh.Key()},
			}))
			sCtx, sCancel := signal.Isolated()
			requests, responses := confluence.Attach[framer.StreamerRequest, framer.StreamerResponse](streamer)
			streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
			time.Sleep(1 * time.Millisecond)
			tsk := &task.Task{Key: taskKey, Name: "task1"}
			Expect(cfg.Task.NewWriter(nil).Create(ctx, tsk)).To(Succeed())
			Consistently(responses.Outlet()).ShouldNot(Receive())
			requests.Close()
			Eventually(responses.Outlet()).Should(BeClosed())
			sCancel()
		})
	})

	Describe("Tracking Device Updates", func() {
		It("Should add the device to state when created", func() {
			rck := &rack.Rack{Name: "rack1"}
			Expect(cfg.Rack.NewWriter(nil).Create(ctx, rck)).To(Succeed())

			dev := device.Device{
				Key:      "dev1",
				Rack:     rck.Key,
				Name:     "device1",
				Location: "slot1",
				Make:     "TestMake",
				Model:    "TestModel",
			}
			Expect(cfg.Device.NewWriter(nil).Create(ctx, dev)).To(Succeed())

			Eventually(func(g Gomega) {
				state, ok := tr.GetDevice(ctx, rck.Key, dev.Key)
				g.Expect(ok).To(BeTrue())
				g.Expect(state.Key).To(Equal(dev.Key))
				g.Expect(state.Rack).To(Equal(rck.Key))
				g.Expect(state.Variant).To(Equal("info"))
			}).Should(Succeed())
		})

		It("Should remove the device from state when deleted", func() {
			rck := &rack.Rack{Name: "rack1"}
			Expect(cfg.Rack.NewWriter(nil).Create(ctx, rck)).To(Succeed())

			dev := device.Device{
				Key:      "dev12345",
				Rack:     rck.Key,
				Name:     "device1",
				Location: "slot1",
			}
			Expect(cfg.Device.NewWriter(nil).Create(ctx, dev)).To(Succeed())

			Eventually(func(g Gomega) {
				_, ok := tr.GetDevice(ctx, rck.Key, dev.Key)
				g.Expect(ok).To(BeTrue())
			}).Should(Succeed())

			Expect(cfg.Device.NewWriter(nil).Delete(ctx, dev.Key)).To(Succeed())

			Eventually(func(g Gomega) {
				_, ok := tr.GetDevice(ctx, rck.Key, dev.Key)
				g.Expect(ok).To(BeFalse())
			}).Should(Succeed())
		})

		It("Should update device state when received", func() {
			rck := &rack.Rack{Name: "rack1"}
			Expect(cfg.Rack.NewWriter(nil).Create(ctx, rck)).To(Succeed())

			dev := device.Device{
				Key:      "dev1",
				Rack:     rck.Key,
				Name:     "device1",
				Location: "slot1",
			}
			Expect(cfg.Device.NewWriter(nil).Create(ctx, dev)).To(Succeed())

			var deviceStateCh channel.Channel
			Expect(dist.Channel.NewRetrieve().WhereNames("sy_device_state").Entry(&deviceStateCh).Exec(ctx, nil)).To(Succeed())

			w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
				Start: telem.Now(),
				Keys:  []channel.Key{deviceStateCh.Key()},
			}))

			state := device.State{
				Key:     dev.Key,
				Rack:    rck.Key,
				Variant: "warning",
				Details: device.NewStaticDetails(map[string]interface{}{
					"message":     "Device is warming up",
					"temperature": 45.5,
				}),
			}

			Expect(w.Write(framer.Frame{
				Keys:   []channel.Key{deviceStateCh.Key()},
				Series: []telem.Series{telem.NewStaticJSONV(state)},
			})).To(BeTrue())

			Expect(w.Close()).To(Succeed())

			Eventually(func(g Gomega) {
				devState, ok := tr.GetDevice(ctx, rck.Key, dev.Key)
				g.Expect(ok).To(BeTrue())
				g.Expect(devState.Variant).To(Equal("warning"))
				g.Expect(string(devState.Details)).To(ContainSubstring("Device is warming up"))
				g.Expect(string(devState.Details)).To(ContainSubstring("45.5"))
			}).Should(Succeed())
		})

		It("Should maintain device state across rack restarts", func() {
			rck := &rack.Rack{Name: "rack1"}
			Expect(cfg.Rack.NewWriter(nil).Create(ctx, rck)).To(Succeed())

			dev := device.Device{
				Key:      "dev1",
				Rack:     rck.Key,
				Name:     "device1",
				Location: "slot1",
			}
			Expect(cfg.Device.NewWriter(nil).Create(ctx, dev)).To(Succeed())

			var deviceStateCh channel.Channel
			Expect(dist.Channel.NewRetrieve().WhereNames("sy_device_state").Entry(&deviceStateCh).Exec(ctx, nil)).To(Succeed())

			w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
				Start: telem.Now(),
				Keys:  []channel.Key{deviceStateCh.Key()},
			}))

			state := device.State{
				Key:     dev.Key,
				Rack:    rck.Key,
				Variant: "error",
				Details: device.NewStaticDetails("Device error state"),
			}

			Expect(w.Write(framer.Frame{
				Keys:   []channel.Key{deviceStateCh.Key()},
				Series: []telem.Series{telem.NewStaticJSONV(state)},
			})).To(BeTrue())

			Expect(w.Close()).To(Succeed())

			Eventually(func(g Gomega) {
				devState, ok := tr.GetDevice(ctx, rck.Key, dev.Key)
				g.Expect(ok).To(BeTrue())
				g.Expect(devState.Variant).To(Equal("error"))
			}).Should(Succeed())

			// Close and reopen the tracker
			Expect(tr.Close()).To(Succeed())
			tr = MustSucceed(tracker.Open(ctx, cfg))

			// Verify state is maintained
			devState, ok := tr.GetDevice(ctx, rck.Key, dev.Key)
			Expect(ok).To(BeTrue())
			Expect(devState.Variant).To(Equal("error"))
		})
	})
})
