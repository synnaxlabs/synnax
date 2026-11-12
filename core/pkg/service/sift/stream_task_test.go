// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package sift_test

import (
	"context"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/sift"
	"github.com/synnaxlabs/synnax/pkg/service/sift/fake"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
)

var _ = Describe("Stream task", func() {
	var (
		f      driver.Factory
		server *fake.Server
		dev    device.Device
		idx    channel.Channel
		ch     channel.Channel
		t      task.Task
		dt     driver.Task
	)

	BeforeEach(func(ctx SpecContext) {
		f = MustSucceed(sift.NewFactory(factoryConfig()))
		server = MustOpen(fake.Open(fake.Config{APIKey: apiKey}))
		dev = createDevice(ctx, sift.Properties{
			URI:    server.Address(),
			APIKey: createAPIKey(ctx, apiKey),
		})
		idx = createIndex(ctx)
		ch = createData(ctx, idx)
		t = newTask(ctx, sift.StreamTaskType, nil)
	})

	streamConfig := func(labels ...label.Key) map[string]any {
		return map[string]any{
			"device":     dev.Key,
			"asset":      "stand",
			"channels":   []channel.Key{ch.Key()},
			"labels":     labels,
			"auto_start": true,
		}
	}

	// start configures t with cfg and waits for it to start.
	start := func(ctx context.Context, cfg map[string]any) {
		GinkgoHelper()
		t.Config = toJSON(cfg)
		dt = MustSucceed(f.ConfigureTask(ctx, t, driver.NoCommand))
		DeferCleanup(func() { Expect(dt.Stop(false)).To(Succeed()) })
		Eventually(func() task.Status { return taskStatus(ctx, t) }).
			Should(HaveField("Message", "Started task"))
	}

	// send writes one sample of ch at the current time.
	send := func(ctx context.Context, v float64) {
		GinkgoHelper()
		write(ctx, idx, ch, []telem.TimeStamp{telem.Now()}, telem.NewSeriesV(v))
	}

	values := func() []any {
		points := server.Points("stand", ch.Name)
		vs := make([]any, len(points))
		for i, p := range points {
			vs[i] = p.Value
		}
		return vs
	}

	progressOf := func(ctx context.Context) (sift.ProgressStatus, error) {
		var s sift.ProgressStatus
		err := statusSvc.NewRetrieve[sift.StreamDetails]().
			Where(status.MatchKeys[sift.StreamDetails](sift.ProgressKey(t.Key))).
			Entry(&s).
			Exec(ctx, db)
		return s, err
	}

	It("Should send live data to Sift and save its progress", func(ctx SpecContext) {
		start(ctx, streamConfig())
		send(ctx, 1)
		Eventually(values).Should(HaveExactElements(1.0))
		Eventually(progressOf).WithArguments(ctx).Should(SatisfyAll(
			HaveField("Message", "Sent to Sift"),
			HaveField(
				"Description",
				"Sent 1 sample to asset stand since the task started.",
			),
			HaveField("Details.Positions", HaveExactElements(
				HaveField("Index", idx.Key()),
			)),
		))
	})

	It("Should send the data written while it was stopped", func(ctx SpecContext) {
		start(ctx, streamConfig())
		send(ctx, 1)
		Eventually(values).Should(HaveExactElements(1.0))
		Expect(dt.Stop(false)).To(Succeed())
		send(ctx, 2)
		start(ctx, streamConfig())
		Eventually(values).Should(HaveExactElements(1.0, 2.0))
	})

	It("Should start again at the current time when its progress is deleted",
		func(ctx SpecContext) {
			start(ctx, streamConfig())
			send(ctx, 1)
			Eventually(values).Should(HaveExactElements(1.0))
			Expect(dt.Stop(false)).To(Succeed())
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return statusSvc.NewWriter(tx).Delete(ctx, sift.ProgressKey(t.Key))
			})).To(Succeed())
			send(ctx, 2)
			start(ctx, streamConfig())
			send(ctx, 3)
			Eventually(values).Should(HaveExactElements(1.0, 3.0))
		})

	It("Should save where it starts before it sends anything", func(ctx SpecContext) {
		before := telem.Now()
		start(ctx, streamConfig())
		Expect(progressOf(ctx)).To(HaveField("Details.Positions", HaveExactElements(
			SatisfyAll(
				HaveField("Index", idx.Key()),
				HaveField("Time", BeNumerically(">=", before)),
			),
		)))
	})

	It("Should start at the current time without progress", func(ctx SpecContext) {
		write(ctx, idx, ch,
			[]telem.TimeStamp{telem.Now().Add(-10 * telem.Second)},
			telem.NewSeriesV(1.0),
		)
		start(ctx, streamConfig())
		send(ctx, 2)
		Eventually(values).Should(HaveExactElements(2.0))
		Consistently(values).WithTimeout(200 * time.Millisecond).
			Should(HaveExactElements(2.0))
	})

	Describe("Runs", func() {
		var l label.Label

		BeforeEach(func(ctx SpecContext) {
			l = label.Label{Name: "hotfire"}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return labelSvc.NewWriter(tx).Create(ctx, &l)
			})).To(Succeed())
		})

		// endedRange creates a range that ends before the next sample, and sends that
		// sample.
		endedRange := func(ctx context.Context, labels ...label.Key) ranger.Range {
			GinkgoHelper()
			now := telem.Now()
			send(ctx, 1)
			rng := createRange(ctx, telem.TimeRange{Start: now, End: now.Add(1)})
			if len(labels) > 0 {
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return rangerSvc.NewWriter(tx).AddLabels(ctx, rng.Key, labels)
				})).To(Succeed())
			}
			send(ctx, 2)
			return rng
		}

		It("Should make a labeled range a run once Sift has its data",
			func(ctx SpecContext) {
				start(ctx, streamConfig(l.Key))
				rng := endedRange(ctx, l.Key)
				Eventually(server.Runs).Should(HaveExactElements(SatisfyAll(
					HaveField("Name", rng.Name),
					HaveField("ClientKey", HaveValue(Equal(rng.Key.String()))),
					HaveField("Tags", ConsistOf("hotfire")),
				)))
			})

		It("Should make a run of a range that ends after the last sample",
			func(ctx SpecContext) {
				start(ctx, streamConfig(l.Key))
				now := telem.Now()
				send(ctx, 1)
				rng := createRange(ctx, telem.TimeRange{
					Start: now,
					End:   now.Add(100 * telem.Millisecond),
				})
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return rangerSvc.NewWriter(tx).
						AddLabels(ctx, rng.Key, []label.Key{l.Key})
				})).To(Succeed())
				Eventually(server.Runs).Should(HaveExactElements(
					HaveField("Name", rng.Name),
				))
			})

		It("Should make a run when one index of the task gets no samples",
			func(ctx SpecContext) {
				idleIdx := createIndex(ctx)
				idle := createData(ctx, idleIdx)
				cfg := streamConfig(l.Key)
				cfg["channels"] = []channel.Key{ch.Key(), idle.Key()}
				start(ctx, cfg)
				rng := endedRange(ctx, l.Key)
				Eventually(server.Runs).Should(HaveExactElements(
					HaveField("Name", rng.Name),
				))
			})

		It("Should update the run of a range that changes", func(ctx SpecContext) {
			start(ctx, streamConfig(l.Key))
			rng := endedRange(ctx, l.Key)
			Eventually(server.Runs).Should(HaveLen(1))
			rename := ranger.NewSetAction(ranger.SetPayload{
				Path:  []string{"name"},
				Value: "renamed",
			})
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return rangerSvc.NewWriter(tx).Apply(ctx, rng.Key, rename)
			})).To(Succeed())
			Eventually(server.Runs).Should(HaveExactElements(
				HaveField("Name", "renamed"),
			))
		})

		It("Should update the run of a range that changed while the task was stopped",
			func(ctx SpecContext) {
				start(ctx, streamConfig(l.Key))
				rng := endedRange(ctx, l.Key)
				Eventually(server.Runs).Should(HaveLen(1))
				Expect(dt.Stop(false)).To(Succeed())
				rename := ranger.NewSetAction(ranger.SetPayload{
					Path:  []string{"name"},
					Value: "renamed",
				})
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return rangerSvc.NewWriter(tx).Apply(ctx, rng.Key, rename)
				})).To(Succeed())
				start(ctx, streamConfig(l.Key))
				Eventually(server.Runs).Should(HaveExactElements(
					HaveField("Name", "renamed"),
				))
			})

		It("Should make a run of a range that ended while the task was stopped",
			func(ctx SpecContext) {
				start(ctx, streamConfig(l.Key))
				send(ctx, 1)
				Eventually(values).Should(ConsistOf(1.0))
				Expect(dt.Stop(false)).To(Succeed())
				rng := endedRange(ctx, l.Key)
				Expect(server.Runs()).To(BeEmpty())
				start(ctx, streamConfig(l.Key))
				Eventually(server.Runs).Should(HaveExactElements(
					HaveField("Name", rng.Name),
				))
			})

		It("Should update the tags of a run when its range gets a label",
			func(ctx SpecContext) {
				cold := label.Label{Name: "cold"}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return labelSvc.NewWriter(tx).Create(ctx, &cold)
				})).To(Succeed())
				start(ctx, streamConfig(l.Key))
				rng := endedRange(ctx, l.Key)
				Eventually(server.Runs).Should(HaveLen(1))
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return rangerSvc.NewWriter(tx).
						AddLabels(ctx, rng.Key, []label.Key{cold.Key})
				})).To(Succeed())
				Eventually(server.Runs).Should(HaveExactElements(
					HaveField("Tags", ConsistOf("hotfire", "cold")),
				))
			})

		It("Should not make a run of a range without a task label",
			func(ctx SpecContext) {
				start(ctx, streamConfig(l.Key))
				endedRange(ctx)
				Eventually(values).Should(HaveLen(2))
				Consistently(server.Runs).WithTimeout(200 * time.Millisecond).
					Should(BeEmpty())
			})
	})

	It("Should warn while Sift is down and clear the warning once it recovers",
		func(ctx SpecContext) {
			start(ctx, streamConfig())
			send(ctx, 1)
			Eventually(values).Should(ConsistOf(1.0))
			server.Fail(grpcstatus.Error(codes.Unavailable, "Sift is down"))
			send(ctx, 2)
			Eventually(
				func() task.Status { return taskStatus(ctx, t) },
			).Should(SatisfyAll(
				HaveField("Variant", status.VariantWarning),
				HaveField("Message", "Failed to send data to Sift"),
				HaveField("Description", ContainSubstring("Sift is down")),
			))
			Eventually(func() task.Status { return taskStatus(ctx, t) }).
				WithTimeout(5 * time.Second).
				Should(HaveField("Variant", status.VariantSuccess))
			Expect(values()).To(ConsistOf(1.0, 2.0))
		})

	It("Should fail when Sift rejects the API key", func(ctx SpecContext) {
		dev = createDevice(ctx, sift.Properties{
			URI:    server.Address(),
			APIKey: createAPIKey(ctx, "wrong"),
		})
		t.Config = toJSON(streamConfig())
		dt = MustSucceed(f.ConfigureTask(ctx, t, driver.NoCommand))
		DeferCleanup(func() { Expect(dt.Stop(false)).To(Succeed()) })
		Eventually(func() task.Status { return taskStatus(ctx, t) }).Should(SatisfyAll(
			HaveField("Variant", status.VariantError),
			HaveField("Message", "Failed to run task"),
			HaveField("Description", ContainSubstring("Sift rejected the API key")),
		))
	})

	It("Should fail when the device belongs to another rack", func(ctx SpecContext) {
		other := rack.Rack{Name: "other"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return rackSvc.NewWriter(tx).Create(ctx, &other)
		})).To(Succeed())
		dev.Key = uuid.New().String()
		dev.Rack = other.Key
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return deviceSvc.NewWriter(tx).Create(ctx, &dev)
		})).To(Succeed())
		t.Config = toJSON(streamConfig())
		msg := "device " + dev.Name + " belongs to another rack"
		Expect(f.ConfigureTask(ctx, t, driver.NoCommand)).Error().To(MatchError(msg))
		Expect(taskStatus(ctx, t)).To(SatisfyAll(
			HaveField("Variant", status.VariantError),
			HaveField("Description", msg),
		))
	})

	DescribeTable("Should reject an invalid config",
		func(ctx SpecContext, field, msg string) {
			cfg := streamConfig()
			delete(cfg, field)
			t.Config = toJSON(cfg)
			Expect(f.ConfigureTask(ctx, t, driver.NoCommand)).Error().
				To(MatchError(ContainSubstring(msg)))
		},
		Entry("no device", "device", "device: required"),
		Entry("no asset", "asset", "asset: required"),
		Entry("no channels", "channels", "channels"),
	)
})
