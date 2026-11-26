// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package impact_test

import (
	"context"
	"encoding/json/v2"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	apiimpact "github.com/synnaxlabs/synnax/pkg/api/impact"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/impact"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Running task deletes", Ordered, func() {
	var (
		layer  *service.Layer
		db     *gorp.DB
		binder *bind.Service
		svc    *apiimpact.Service
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		node := testutil.NewNode(ctx)
		db = node.DB
		layer = MustOpen(svctestutil.OpenLayer(ctx, node))
		svc = MustSucceed(apiimpact.NewService(apiimpact.ServiceConfig{
			RBAC:   layer.RBAC,
			Impact: layer.Impact,
		}))
		binder = MustSucceed(bind.NewService(bind.ServiceConfig{
			DB:   db,
			RBAC: layer.RBAC,
			Held: apiauth.HeldTables(),
		}))
	})

	createChannel := func(ctx SpecContext) channel.Key {
		GinkgoHelper()
		ch := channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.Float32T,
			Virtual:  true,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Channel.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		return ch.Key()
	}
	// createRunningTask creates a task that reads ch and reports running, on a new
	// rack,
	// because the rack monitor marks the tasks of a silent rack as not running.
	createRunningTask := func(ctx SpecContext, ch channel.Key) task.Task {
		GinkgoHelper()
		r := rack.Rack{Name: "Driver"}
		dev := device.Device{
			Key:      UniqueChannelName(),
			Rack:     layer.Rack.EmbeddedKey,
			Name:     "plc",
			Make:     "modbus",
			Model:    "modbus",
			Location: "127.0.0.1:502",
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			if err := layer.Rack.NewWriter(tx).Create(ctx, &r); err != nil {
				return err
			}
			return layer.Device.NewWriter(tx).Create(ctx, &dev)
		})).To(Succeed())
		t := task.Task{
			Name: "modbus_read",
			Type: "modbus_read",
			Rack: r.Key,
			Config: msgpack.EncodedJSON{
				"device": dev.Key,
				"channels": []any{map[string]any{
					"type":    "coil",
					"key":     "coil-1",
					"channel": ch,
				}},
			},
			Status: &task.Status{
				Variant: status.VariantSuccess,
				Message: "Task is running",
				Time:    telem.Now(),
				Details: task.StatusDetails{Running: true},
			},
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Task.NewWriter(tx).Create(ctx, &t)
		})).To(Succeed())
		return t
	}
	// reportStopped writes the status a task writes when it stops.
	reportStopped := func(ctx SpecContext, t task.Task) {
		GinkgoHelper()
		stat := task.Status{
			Owner:   new(t.ResourceID()),
			Variant: status.VariantSuccess,
			Message: "Stopped task",
			Time:    telem.Now(),
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Status.NewWriter(tx).Set(ctx, &stat)
		})).To(Succeed())
	}
	// streamCommands sends each command written to the task command channel on the
	// returned channel. It returns once the stream is live.
	streamCommands := func(ctx SpecContext) <-chan task.Command {
		GinkgoHelper()
		streamer := MustSucceed(layer.Framer.NewStreamer(ctx, framer.StreamerConfig{
			Keys:        channel.Keys{layer.Task.CommandChannelKey()},
			SendOpenAck: true,
		}))
		requests, responses := confluence.Attach(streamer, 2)
		sCtx, cancel := signal.Isolated()
		shutdown := signal.NewHardShutdown(sCtx, cancel)
		streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
		Eventually(responses.Outlet()).Should(Receive())
		commands := make(chan task.Command, 16)
		done := make(chan struct{})
		go func() {
			defer GinkgoRecover()
			defer close(done)
			for res := range responses.Outlet() {
				for series := range res.Frame.Series() {
					for sample := range series.Samples() {
						var cmd task.Command
						Expect(json.Unmarshal(sample, &cmd)).To(Succeed())
						commands <- cmd
					}
				}
			}
		}()
		DeferCleanup(func() {
			requests.Close()
			Eventually(done).Should(BeClosed())
			Expect(shutdown.Close()).To(Succeed())
		})
		return commands
	}
	taskExists := func(ctx SpecContext, k task.Key) bool {
		GinkgoHelper()
		return MustSucceed(layer.Task.NewRetrieve().
			Where(task.MatchKeys(k)).
			Exists(ctx, db))
	}
	channelExists := func(ctx SpecContext, k channel.Key) bool {
		GinkgoHelper()
		return MustSucceed(layer.Channel.NewRetrieve().
			Where(channel.MatchKeys(k)).
			Exists(ctx, db))
	}
	createUser := func(ctx SpecContext) user.User {
		GinkgoHelper()
		return MustSucceed(rbactestutil.CreateUser(
			ctx,
			db,
			layer.RBAC,
			layer.User,
			user.User{Name: uuid.New().String()},
		))
	}
	grant := func(ctx SpecContext, u user.User, a access.Action, objects ...resource.ID) {
		GinkgoHelper()
		MustSucceed(rbactestutil.Grant(
			ctx,
			db,
			layer.RBAC,
			layer.User,
			u.Key,
			[]access.Action{a},
			objects...,
		))
	}
	// grantDelete grants u what a delete of ch and of t needs.
	grantDelete := func(ctx SpecContext, u user.User, ch channel.Key, t task.Task) {
		GinkgoHelper()
		grant(ctx, u, access.ActionRetrieve, channel.ResourceID(ch), t.ResourceID())
		grant(ctx, u, access.ActionDelete, channel.ResourceID(ch), t.ResourceID())
		grant(ctx, u, access.ActionDelete, resource.ID{Type: resource.TypeStatus})
		grant(ctx, u, access.ActionUpdate, t.Rack.ResourceID())
	}
	deleteOf := func(ch channel.Key) apiimpact.Change {
		return apiimpact.Change{Variant: apiimpact.DeleteChange{
			Records: []resource.ID{channel.ResourceID(ch)},
		}}
	}
	// expectStop expects a stop command for t while t still exists, then runs then. The
	// returned channel closes once it has.
	expectStop := func(
		ctx SpecContext,
		commands <-chan task.Command,
		t task.Task,
		then func(),
	) <-chan struct{} {
		driver := make(chan struct{})
		go func() {
			defer GinkgoRecover()
			defer close(driver)
			var cmd task.Command
			Eventually(commands).Should(Receive(&cmd))
			Expect(cmd.Task).To(Equal(t.Key))
			Expect(cmd.Type).To(Equal("stop"))
			Expect(taskExists(ctx, t.Key)).To(BeTrue())
			then()
		}()
		return driver
	}

	It("Should stop a running task that a delete pick removes before its record goes",
		func(ctx SpecContext) {
			ch := createChannel(ctx)
			t := createRunningTask(ctx, ch)
			u := createUser(ctx)
			grantDelete(ctx, u, ch, t)
			driver := expectStop(ctx, streamCommands(ctx), t, func() {
				reportStopped(ctx, t)
			})
			MustSucceed(
				svc.ApplyHandler(binder)(AuthedCtx(ctx, u), apiimpact.ApplyRequest{
					Change: deleteOf(ch),
					Picks: map[string]impact.Choice{
						t.ResourceID().String(): impact.ChoiceDelete,
					},
				}),
			)
			Eventually(driver).Should(BeClosed())
			Expect(taskExists(ctx, t.Key)).To(BeFalse())
			Expect(channelExists(ctx, ch)).To(BeFalse())
		},
	)

	It("Should stop nothing for a caller that cannot delete a hidden task", func(
		ctx SpecContext,
	) {
		ch := createChannel(ctx)
		t := createRunningTask(ctx, ch)
		u := createUser(ctx)
		grant(ctx, u, access.ActionRetrieve, channel.ResourceID(ch))
		grant(ctx, u, access.ActionDelete, channel.ResourceID(ch))
		commands := streamCommands(ctx)
		Expect(svc.ApplyHandler(binder)(AuthedCtx(ctx, u), apiimpact.ApplyRequest{
			Change: deleteOf(ch),
			Hidden: new(impact.ChoiceDelete),
		})).Error().To(SatisfyAll(
			MatchError(access.ErrDenied),
			MatchError(ContainSubstring("a record that it cannot see")),
			Not(MatchError(ContainSubstring(t.Key.String()))),
		))
		Consistently(commands).WithTimeout(500 * time.Millisecond).ShouldNot(Receive())
		Expect(taskExists(ctx, t.Key)).To(BeTrue())
		Expect(channelExists(ctx, ch)).To(BeTrue())
	})

	It("Should delete nothing when the stop fails", func(ctx SpecContext) {
		ch := createChannel(ctx)
		t := createRunningTask(ctx, ch)
		u := createUser(ctx)
		grantDelete(ctx, u, ch, t)
		canceled, cancel := context.WithCancel(ctx)
		defer cancel()
		reqCtx := AuthedCtx(canceled, u)
		driver := expectStop(ctx, streamCommands(ctx), t, cancel)
		Expect(svc.ApplyHandler(binder)(reqCtx, apiimpact.ApplyRequest{
			Change: deleteOf(ch),
			Picks: map[string]impact.Choice{
				t.ResourceID().String(): impact.ChoiceDelete,
			},
		})).Error().To(MatchError(context.Canceled))
		Eventually(driver).Should(BeClosed())
		Expect(taskExists(ctx, t.Key)).To(BeTrue())
		Expect(channelExists(ctx, ch)).To(BeTrue())
	})
})
