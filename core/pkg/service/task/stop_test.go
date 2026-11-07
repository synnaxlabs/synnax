// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package task_test

import (
	"context"
	"fmt"
	"strings"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Stopper", func() {
	Describe("Config", func() {
		full := func() task.StopperConfig {
			return task.StopperConfig{
				Task:    &task.Service{},
				Status:  &status.Service{},
				Framer:  &framer.Service{},
				Timeout: time.Second,
			}
		}
		It("Should validate a full config", func() {
			Expect(full().Validate()).To(Succeed())
		})
		It("Should take the fields that the other config sets", func() {
			other := full()
			merged := task.StopperConfig{
				Task:    &task.Service{},
				Status:  &status.Service{},
				Framer:  &framer.Service{},
				Timeout: time.Minute,
			}.Override(other)
			Expect(merged.Task).To(BeIdenticalTo(other.Task))
			Expect(merged.Status).To(BeIdenticalTo(other.Status))
			Expect(merged.Framer).To(BeIdenticalTo(other.Framer))
			Expect(merged.Timeout).To(Equal(other.Timeout))
		})
		It("Should keep its fields when the other config is empty", func() {
			cfg := full()
			merged := cfg.Override(task.StopperConfig{})
			Expect(merged.Task).To(BeIdenticalTo(cfg.Task))
			Expect(merged.Status).To(BeIdenticalTo(cfg.Status))
			Expect(merged.Framer).To(BeIdenticalTo(cfg.Framer))
			Expect(merged.Timeout).To(Equal(cfg.Timeout))
		})
		It("Should default the timeout", func() {
			cfg := full()
			cfg.Timeout = 0
			Expect(task.NewStopper(cfg)).ToNot(BeNil())
		})
		DescribeTable(
			"Should fail when a field is missing",
			func(clear func(*task.StopperConfig), msg string) {
				cfg := full()
				clear(&cfg)
				Expect(cfg.Validate()).To(MatchError(msg))
				Expect(task.NewStopper(cfg)).Error().To(MatchError(msg))
			},
			Entry(
				"task",
				func(c *task.StopperConfig) { c.Task = nil },
				"task: must be non-nil",
			),
			Entry(
				"status",
				func(c *task.StopperConfig) { c.Status = nil },
				"status: must be non-nil",
			),
			Entry(
				"framer",
				func(c *task.StopperConfig) { c.Framer = nil },
				"framer: must be non-nil",
			),
		)
		It("Should reject a negative timeout", func() {
			cfg := full()
			cfg.Timeout = -time.Second
			Expect(cfg.Validate()).To(MatchError("timeout: must be positive"))
		})
	})

	Describe("Stop", func() {
		var svc *service.Layer
		BeforeEach(func(ctx SpecContext) {
			svc = MustOpen(svctestutil.OpenLayer(ctx, testutil.NewNode(ctx)))
		})
		withTimeout := func(timeout time.Duration) *task.Stopper {
			GinkgoHelper()
			return MustSucceed(task.NewStopper(task.StopperConfig{
				Task:    svc.Task,
				Status:  svc.Status,
				Framer:  svc.Framer,
				Timeout: timeout,
			}))
		}
		createTask := func(ctx SpecContext, running bool) task.Task {
			GinkgoHelper()
			t := task.Task{
				Type: pagerduty.AlertTaskType,
				Rack: svc.Rack.EmbeddedKey,
				Name: "Stopper",
				Status: &task.Status{
					Variant: status.VariantSuccess,
					Message: "Task status",
					Time:    telem.Now(),
					Details: task.StatusDetails{Running: running},
				},
			}
			Expect(svc.Task.NewWriter(nil).Create(ctx, &t)).To(Succeed())
			return t
		}
		createChannel := func(ctx SpecContext, ch channel.Channel) channel.Channel {
			GinkgoHelper()
			ch.Name = "stop_" + strings.ReplaceAll(uuid.New().String(), "-", "_")
			ch.Leaseholder = 1
			Expect(svc.Channel.NewWriter(nil).Create(ctx, &ch)).To(Succeed())
			return ch
		}
		retrieveRunning := func(ctx SpecContext, t task.Task) (bool, error) {
			var stat task.Status
			err := svc.Status.NewRetrieve[task.StatusDetails]().
				Where(status.MatchKeys[task.StatusDetails](t.OntologyID().String())).
				Entry(&stat).
				Exec(ctx, nil)
			return stat.Details.Running, err
		}
		// startArc starts an Arc task on the runtime in the Core.
		startArc := func(ctx SpecContext) task.Task {
			GinkgoHelper()
			index := createChannel(
				ctx, channel.Channel{DataType: telem.TimestampT, IsIndex: true},
			)
			data := createChannel(ctx, channel.Channel{
				DataType:   telem.Uint8T,
				LocalIndex: index.LocalKey,
			})
			a := arc.Arc{Variant: arc.AutomationArc{Base: arc.Base{
				Name: "stopper-" + uuid.New().String(),
				Source: arc.Source{Variant: arc.TextSource{Text: text.Text{
					Raw: fmt.Sprintf("interval(period=10ms) -> %s", data.Name),
				}}},
			}}}
			Expect(svc.Arc.NewWriter(nil).Create(ctx, &a)).To(Succeed())
			var coreRack rack.Rack
			Expect(svc.Rack.NewRetrieve().
				Where(rack.MatchNames("Node 1")).
				Entry(&coreRack).
				Exec(ctx, nil)).To(Succeed())
			t := task.Task{
				Rack:   coreRack.Key,
				Name:   a.Base().Name,
				Type:   arc.TaskType,
				Config: msgpack.EncodedJSON{"arc_key": a.Base().Key.String()},
			}
			Expect(svc.Task.NewWriter(nil).Create(ctx, &t)).To(Succeed())
			cmdKey := svc.Task.CommandChannelKey()
			w := MustSucceed(svc.Framer.OpenWriter(ctx, framer.WriterConfig{
				Keys:  channel.Keys{cmdKey},
				Start: telem.Now(),
			}))
			Expect(w.Write(frame.NewUnary(
				cmdKey,
				MustSucceed(telem.NewJSONSeriesV(task.Command{
					Task: t.Key,
					Type: "start",
					Key:  uuid.New().String(),
					Args: msgpack.EncodedJSON{},
				})),
			))).To(BeTrue())
			Expect(w.Close()).To(Succeed())
			Eventually(func() (bool, error) {
				return retrieveRunning(ctx, t)
			}).Should(BeTrue())
			return t
		}
		It("Should stop a running task and return on its ack", func(ctx SpecContext) {
			const timeout = 10 * time.Second
			t := startArc(ctx)
			start := time.Now()
			Expect(withTimeout(timeout).Stop(ctx, []task.Key{t.Key})).To(Succeed())
			Expect(time.Since(start)).To(BeNumerically("<", timeout))
			Expect(retrieveRunning(ctx, t)).To(BeFalse())
		})
		It("Should not wait for no keys", func(ctx SpecContext) {
			start := time.Now()
			Expect(withTimeout(10*time.Second).Stop(ctx, nil)).To(Succeed())
			Expect(time.Since(start)).To(BeNumerically("<", time.Second))
		})
		It("Should not wait for a task that is not running", func(ctx SpecContext) {
			t := createTask(ctx, false)
			start := time.Now()
			Expect(
				withTimeout(10*time.Second).Stop(ctx, []task.Key{t.Key}),
			).To(Succeed())
			Expect(time.Since(start)).To(BeNumerically("<", time.Second))
		})
		It("Should return at the timeout with no error", func(ctx SpecContext) {
			const timeout = 200 * time.Millisecond
			t := createTask(ctx, true)
			start := time.Now()
			Expect(withTimeout(timeout).Stop(ctx, []task.Key{t.Key})).To(Succeed())
			Expect(time.Since(start)).To(And(
				BeNumerically(">=", timeout),
				BeNumerically("<", 2*timeout),
			))
		})
		It("Should return the error of a canceled caller", func(ctx SpecContext) {
			t := createTask(ctx, true)
			reqCtx, cancel := context.WithCancel(ctx)
			defer time.AfterFunc(50*time.Millisecond, cancel).Stop()
			Expect(withTimeout(10*time.Second).Stop(reqCtx, []task.Key{t.Key})).
				To(MatchError(context.Canceled))
		})
	})
})
