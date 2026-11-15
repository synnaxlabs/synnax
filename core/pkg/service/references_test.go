// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package service_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("References", Ordered, func() {
	var (
		layer *service.Layer
		db    *gorp.DB
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		node := testutil.NewNode(ctx)
		db = node.DB
		layer = MustOpen(svctestutil.OpenLayer(ctx, node))
	})

	createTask := func(ctx SpecContext, typ string, cfg msgpack.EncodedJSON) task.Task {
		GinkgoHelper()
		t := task.Task{Name: typ, Type: typ, Config: cfg}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Task.NewWriter(tx).Create(ctx, &t)
		})).To(Succeed())
		return t
	}

	Describe("Task configs", func() {
		var (
			ch  channel.Channel
			dev device.Device
			tsk task.Task
		)
		BeforeEach(func(ctx SpecContext) {
			ch = channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.Float32T,
				Virtual:  true,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Channel.NewWriter(tx).Create(ctx, &ch)
			})).To(Succeed())
			dev = device.Device{
				Key:      device.Key(UniqueChannelName()),
				Rack:     layer.Rack.EmbeddedKey,
				Name:     "plc",
				Make:     "modbus",
				Model:    "modbus",
				Location: "127.0.0.1:502",
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Device.NewWriter(tx).Create(ctx, &dev)
			})).To(Succeed())
			tsk = createTask(ctx, "modbus_read", msgpack.EncodedJSON{
				"device": string(dev.Key),
				"channels": []any{map[string]any{
					"type":    "coil",
					"key":     "coil-1",
					"channel": ch.Key(),
				}},
			})
		})

		It("Should refuse deleting a channel that a task config names", func(
			ctx SpecContext,
		) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Channel.NewWriter(tx).Delete(ctx, ch.Key(), false)
			})).To(
				MatchError(gorp.RestrictError{
					Target:    gorp.NewTarget("channel", ch.Key()),
					Holder:    gorp.NewTarget("modbus_read_config", tsk.Key),
					Reference: "channels.channel",
				}),
			)
		})

		It("Should refuse deleting a device that a task config names", func(
			ctx SpecContext,
		) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Device.NewWriter(tx).Delete(ctx, dev.Key)
			})).To(
				MatchError(gorp.RestrictError{
					Target:    gorp.NewTarget("device", dev.Key),
					Holder:    gorp.NewTarget("modbus_read_config", tsk.Key),
					Reference: "device",
				}),
			)
		})

		It("Should delete the channel and device once the task is gone", func(
			ctx SpecContext,
		) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Task.NewWriter(tx).Delete(ctx, tsk.Key, false)
			})).
				To(Succeed())
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Channel.NewWriter(tx).Delete(ctx, ch.Key(), false)
			})).
				To(Succeed())
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Device.NewWriter(tx).Delete(ctx, dev.Key)
			})).To(Succeed())
		})
	})

	It(
		"Should clear the status of a PagerDuty alert and keep the alert",
		func(ctx SpecContext) {
			st := status.Status[any]{
				Key:     status.Key(UniqueChannelName()),
				Name:    "pressure",
				Variant: status.VariantWarning,
				Time:    telem.Now(),
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Status.NewWriter(tx).Set(ctx, &st)
			})).To(Succeed())
			tsk := createTask(ctx, pagerduty.AlertTaskType, msgpack.EncodedJSON{
				"alerts": []any{map[string]any{
					"key":    "alert-1",
					"status": string(st.Key),
				}},
			})
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Status.NewWriter(tx).Delete(ctx, st.Key)
			})).To(Succeed())
			var res task.Task
			Expect(layer.Task.NewRetrieve().
				Where(task.MatchKeys(tsk.Key)).
				Entry(&res).
				Exec(ctx, db)).To(Succeed())
			Expect(res.Config["alerts"]).To(HaveExactElements(SatisfyAll(
				HaveKeyWithValue("key", "alert-1"),
				HaveKeyWithValue("status", ""),
			)))
		},
	)
})
