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
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/service/hardware/rack"
	"github.com/synnaxlabs/synnax/pkg/service/hardware/task"
	"github.com/synnaxlabs/synnax/pkg/service/hardware/tracker"
	"github.com/synnaxlabs/x/binary"
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
			Channel:      dist.Channel,
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
		}
	})
	BeforeEach(func() {
		tr = MustSucceed(tracker.Open(ctx, cfg))
	})
	AfterEach(func() {
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
			rack := &rack.Rack{Key: rack.NewKey(dist.Cluster.HostKey(), 1), Name: "rack1"}
			Expect(cfg.Rack.NewWriter(nil).Create(ctx, rack)).To(Succeed())
			var heartbeatCh channel.Channel
			Expect(dist.Channel.NewRetrieve().WhereNames("sy_rack_heartbeat").Entry(&heartbeatCh).Exec(ctx, nil)).To(Succeed())
			time.Sleep(5 * time.Millisecond)
			w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
				Start: telem.Now(),
				Keys:  []channel.Key{heartbeatCh.Key()},
			}))
			// Set the first 32 bits of the uint64 heartbeat as the rack key, second as "1"
			key := uint64(rack.Key)<<32 | uint64(1)
			Expect(w.Write(framer.Frame{
				Keys:   []channel.Key{heartbeatCh.Key()},
				Series: []telem.Series{telem.NewSeries([]uint64{key})},
			})).To(BeTrue())
			Expect(w.Close()).To(Succeed())
			Eventually(func(g Gomega) {
				r, ok := tr.GetRack(ctx, rack.Key)
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
				Variant: task.StatusError,
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
				g.Expect(t.Variant).To(Equal(task.StatusError))
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
			b := MustSucceed((&binary.JSONCodec{}).Encode(ctx, task.State{
				Variant: task.StatusError,
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
				g.Expect(t.Variant).To(Equal(task.StatusError))
			}).Should(Succeed())
			Expect(tr.Close()).To(Succeed())
			tr = MustSucceed(tracker.Open(ctx, cfg))
			state, ok := tr.GetTask(ctx, taskKey)
			Expect(ok).To(BeTrue())
			Expect(state.Variant).To(Equal(task.StatusError))
		})
	})
})
