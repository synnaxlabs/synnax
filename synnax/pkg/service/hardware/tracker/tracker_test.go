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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gleak"
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
	"time"
)

var _ = Describe("Tracker", Ordered, func() {
	var (
		cfg tracker.Config
		tr  *tracker.Tracker
	)
	BeforeAll(func() {
		rackSvc := MustSucceed(rack.OpenService(ctx, rack.Config{
			DB:           dist.Storage.Gorpify(),
			Ontology:     dist.Ontology,
			Group:        dist.Group,
			HostProvider: dist.Cluster,
			Signals:      dist.Signals,
		}))
		taskSvc := MustSucceed(task.OpenService(ctx, task.Config{
			DB:           dist.Storage.Gorpify(),
			Ontology:     dist.Ontology,
			Group:        dist.Group,
			Rack:         rackSvc,
			HostProvider: dist.Cluster,
			Channel:      dist.Channel,
			Signals:      dist.Signals,
		}))
		cfg = tracker.Config{
			DB:           dist.Storage.Gorpify(),
			Rack:         rackSvc,
			Task:         taskSvc,
			Signals:      dist.Signals,
			Channels:     dist.Channel,
			HostProvider: dist.Cluster,
			Framer:       dist.Framer,
		}
	})
	var grs []gleak.Goroutine
	JustBeforeEach(func() {
		grs = gleak.Goroutines()
		tr = MustSucceed(tracker.Open(ctx, cfg))
	})
	JustAfterEach(func() {
		Expect(tr.Close()).To(Succeed())
		Expect(gleak.Goroutines()).ShouldNot(gleak.HaveLeaked(grs))
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
			rack := &rack.Rack{Key: rack.NewKey(dist.Cluster.HostKey(), 1), Name: "rack1"}
			Expect(cfg.Rack.NewWriter(nil).Create(ctx, rack)).To(Succeed())
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
		It("Should update the rack heartbeat when received", func() {
			rck := &rack.Rack{Key: rack.NewKey(dist.Cluster.HostKey(), 1), Name: "rack1"}
			Expect(cfg.Rack.NewWriter(nil).Create(ctx, rck)).To(Succeed())
			var heartbeatCh channel.Channel
			Expect(dist.Channel.NewRetrieve().WhereNames("sy_rack_heartbeat").Entry(&heartbeatCh).Exec(ctx, nil)).To(Succeed())
			w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
				Start: telem.Now(),
				Keys:  []channel.Key{heartbeatCh.Key()},
			}))
			key := rack.NewHeartbeat(rck.Key, 1)
			Expect(w.Write(framer.Frame{
				Keys:   []channel.Key{heartbeatCh.Key()},
				Series: []telem.Series{telem.NewSeriesV(uint64(key))},
			})).To(BeTrue())
			Expect(w.Close()).To(Succeed())
			Eventually(func(g Gomega) {
				r, ok := tr.GetRack(ctx, rck.Key)
				g.Expect(ok).To(BeTrue())
				g.Expect(r.Heartbeat).To(Equal(key))
			}).Should(Succeed())
		})
	})

	Describe("Tracking Task State", func() {
		It("Should correctly update the state of a task", func() {
			rack := &rack.Rack{Key: rack.NewKey(dist.Cluster.HostKey(), 1), Name: "rack1"}
			Expect(cfg.Rack.NewWriter(nil).Create(ctx, rack)).To(Succeed())
			taskKey := task.NewKey(rack.Key, 1)
			tsk := &task.Task{Key: taskKey, Name: "task1"}
			Expect(cfg.Task.NewWriter(nil).Create(ctx, tsk)).To(Succeed())
			var taskStateCh channel.Channel
			Expect(dist.Channel.NewRetrieve().WhereNames("sy_task_state").Entry(&taskStateCh).Exec(ctx, nil)).To(Succeed())
			w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
				Start: telem.Now(),
				Keys:  []channel.Key{taskStateCh.Key()},
			}))
			b := MustSucceed((&binary.JSONCodec{}).Encode(ctx, task.State{
				Variant: task.ErrorStateVariant,
				Task:    taskKey,
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
				t, ok := tr.GetTask(ctx, taskKey)
				g.Expect(ok).To(BeTrue())
				g.Expect(t.Variant).To(Equal(task.ErrorStateVariant))
			}).Should(Succeed())
		})
	})

	Describe("Persisting state across restarts", func() {
		It("Should persist the state of tasks even if the tracker service is closed and reopened", func() {
			rack := &rack.Rack{Key: rack.NewKey(dist.Cluster.HostKey(), 1), Name: "rack1"}
			Expect(cfg.Rack.NewWriter(nil).Create(ctx, rack)).To(Succeed())
			taskKey := task.NewKey(rack.Key, 1)
			tsk := &task.Task{Key: taskKey, Name: "task1"}
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
					Task:    taskKey,
				})},
			})).To(BeTrue())
			Expect(w.Close()).To(Succeed())
			Eventually(func(g Gomega) {
				t, ok := tr.GetTask(ctx, taskKey)
				g.Expect(ok).To(BeTrue())
				g.Expect(t.Variant).To(Equal(task.ErrorStateVariant))
			}).Should(Succeed())
			Expect(tr.Close()).To(Succeed())
			tr = MustSucceed(tracker.Open(ctx, cfg))
			state, ok := tr.GetTask(ctx, taskKey)
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
		It("Should not update the state of tasks when a rack is alive", func() {
			rck := &rack.Rack{Key: rack.NewKey(dist.Cluster.HostKey(), 1), Name: "rack1"}
			Expect(cfg.Rack.NewWriter(nil).Create(ctx, rck)).To(Succeed())
			taskKey := task.NewKey(rck.Key, 1)
			var heartbeatCh channel.Channel
			Expect(dist.Channel.NewRetrieve().WhereNames("sy_rack_heartbeat").Entry(&heartbeatCh).Exec(ctx, nil)).To(Succeed())
			w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
				Start: telem.Now(),
				Keys:  []channel.Key{heartbeatCh.Key()},
			}))
			key := rack.NewHeartbeat(rck.Key, 1)
			Expect(w.Write(framer.Frame{
				Keys:   []channel.Key{heartbeatCh.Key()},
				Series: []telem.Series{telem.NewSeriesV(uint64(key))},
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
})
