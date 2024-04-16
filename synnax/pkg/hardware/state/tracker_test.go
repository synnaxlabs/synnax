package state_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/hardware/rack"
	"github.com/synnaxlabs/synnax/pkg/hardware/state"
	"github.com/synnaxlabs/synnax/pkg/hardware/task"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Tracker", Ordered, func() {
	var (
		trackerCfg state.TrackerConfig
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
		trackerCfg = state.TrackerConfig{
			DB:           dist.Storage.Gorpify(),
			Rack:         rackSvc,
			Task:         taskSvc,
			Signals:      dist.Signals,
			Channels:     dist.Channel,
			HostProvider: dist.Cluster,
		}
	})
	Describe("Tracking Rack Updates", func() {
		It("Should add the rack to state when created", func() {
			tracker := MustSucceed(state.OpenTracker(ctx, trackerCfg))
			rackKey := rack.NewKey(dist.Cluster.HostKey(), 1)
			rack := &rack.Rack{Key: rackKey, Name: "rack1"}
			Expect(trackerCfg.Rack.NewWriter(nil).Create(ctx, rack)).To(Succeed())
			Eventually(func(g Gomega) {
				g.Expect(tracker.Racks).To(HaveKey(rackKey))
			}).Should(Succeed())
			Expect(tracker.Close()).To(Succeed())
		})
		It("Should remove the rack from state when deleted", func() {
			tracker := MustSucceed(state.OpenTracker(ctx, trackerCfg))
			rackKey := rack.NewKey(dist.Cluster.HostKey(), 1)
			rack := &rack.Rack{Key: rackKey, Name: "rack1"}
			Expect(trackerCfg.Rack.NewWriter(nil).Create(ctx, rack)).To(Succeed())
			Expect(trackerCfg.Rack.NewWriter(nil).Delete(ctx, rackKey)).To(Succeed())
			Eventually(func(g Gomega) {
				g.Expect(tracker.Racks).ToNot(HaveKey(rackKey))
			}).Should(Succeed())
			Eventually(func(g Gomega) {
				g.Expect(tracker.Racks).ToNot(HaveKey(rackKey))
			}).Should(Succeed())
			Expect(tracker.Close()).To(Succeed())
		})
	})
	Describe("Tracking Task Updates", func() {
		It("Should add the task to state when created", func() {
			tracker := MustSucceed(state.OpenTracker(ctx, trackerCfg))
			rack := &rack.Rack{Key: rack.NewKey(dist.Cluster.HostKey(), 1), Name: "rack1"}
			Expect(trackerCfg.Rack.NewWriter(nil).Create(ctx, rack)).To(Succeed())
			taskKey := task.NewKey(rack.Key, 1)
			task := &task.Task{Key: taskKey, Name: "task1"}
			Expect(trackerCfg.Task.NewWriter(nil).Create(ctx, task)).To(Succeed())
			Eventually(func(g Gomega) {
				g.Expect(tracker.Racks[rack.Key].Tasks).To(HaveKey(taskKey))
			}).Should(Succeed())
			Expect(tracker.Close()).To(Succeed())
		})
		It("Should remove the task from state when deleted", func() {
			tracker := MustSucceed(state.OpenTracker(ctx, trackerCfg))
			rack := &rack.Rack{Key: rack.NewKey(dist.Cluster.HostKey(), 1), Name: "rack1"}
			Expect(trackerCfg.Rack.NewWriter(nil).Create(ctx, rack)).To(Succeed())
			taskKey := task.NewKey(rack.Key, 1)
			task := &task.Task{Key: taskKey, Name: "task1"}
			Expect(trackerCfg.Task.NewWriter(nil).Create(ctx, task)).To(Succeed())
			Expect(trackerCfg.Task.NewWriter(nil).Delete(ctx, taskKey)).To(Succeed())
			Eventually(func(g Gomega) {
				g.Expect(tracker.Racks[rack.Key].Tasks).ToNot(HaveKey(taskKey))
			}).Should(Succeed())
			Expect(tracker.Close()).To(Succeed())
		})
	})
	Describe("Tracking Rack Heartbeats", func() {
		It("Should update the rack heartbeat when received", func() {
			tracker := MustSucceed(state.OpenTracker(ctx, trackerCfg))
			rack := &rack.Rack{Key: rack.NewKey(dist.Cluster.HostKey(), 1), Name: "rack1"}
			Expect(trackerCfg.Rack.NewWriter(nil).Create(ctx, rack)).To(Succeed())
			var heartbeatCh channel.Channel
			Expect(dist.Channel.NewRetrieve().WhereNames("sy_rack_heartbeat").Entry(&heartbeatCh).Exec(ctx, nil)).To(Succeed())
			w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
				Start: telem.Now(),
				Keys:  []channel.Key{heartbeatCh.Key()},
			}))
			// set the first 32 bits of the uint64 heartbeat as the rack key, second as "1"
			key := uint64(rack.Key)<<32 | uint64(1)
			Expect(w.Write(framer.Frame{
				Keys:   []channel.Key{heartbeatCh.Key()},
				Series: []telem.Series{telem.NewSeries([]uint64{key})},
			})).To(BeTrue())
			Expect(w.Close()).To(Succeed())
			Eventually(func(g Gomega) {
				g.Expect(tracker.Racks[rack.Key].Heartbeat).To(Equal(uint64(key)))
			}).Should(Succeed())
			Expect(tracker.Close()).To(Succeed())
		})
	})
	Describe("Tracking Task State", func() {
		It("Should correctly update the state of a task", func() {
			tracker := MustSucceed(state.OpenTracker(ctx, trackerCfg))
			rack := &rack.Rack{Key: rack.NewKey(dist.Cluster.HostKey(), 1), Name: "rack1"}
			Expect(trackerCfg.Rack.NewWriter(nil).Create(ctx, rack)).To(Succeed())
			taskKey := task.NewKey(rack.Key, 1)
			task := &task.Task{Key: taskKey, Name: "task1"}
			Expect(trackerCfg.Task.NewWriter(nil).Create(ctx, task)).To(Succeed())
			var taskStateCh channel.Channel
			Expect(dist.Channel.NewRetrieve().WhereNames("sy_task_state").Entry(&taskStateCh).Exec(ctx, nil)).To(Succeed())
			w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
				Start: telem.Now(),
				Keys:  []channel.Key{taskStateCh.Key()},
			}))
			b := MustSucceed((&binary.JSONEncoderDecoder{}).Encode(ctx, state.Task{
				Status: state.TaskStatusRunning,
				Key:    taskKey,
			}))
			Expect(w.Write(framer.Frame{
				Keys: []channel.Key{taskStateCh.Key()},
				Series: []telem.Series{telem.Series{
					DataType: telem.JSONT,
					Data:     append(b, '\n'),
				}},
			})).To(BeTrue())
			Expect(w.Close()).To(Succeed())
			Eventually(func(g Gomega) {
				g.Expect(tracker.Racks[rack.Key].Tasks[taskKey].Status).To(Equal(state.TaskStatusRunning))
			}).Should(Succeed())
			Expect(tracker.Close()).To(Succeed())
		})
	})
})
