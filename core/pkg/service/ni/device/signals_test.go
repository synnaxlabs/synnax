// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package device_test

import (
	"context"
	"encoding/json/v2"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	calcgraph "github.com/synnaxlabs/synnax/pkg/service/calculation/graph"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ni/device"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Signals", Ordered, func() {
	var (
		node       testutil.Node
		channelSvc *channel.Service
		framerSvc  *framer.Service
		rackSvc    *rack.Service
		svc        *device.Service
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		node = testutil.NewNode(ctx)
		MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: node.DB}))
		MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: node.DB}))
		stat := MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: node.DB}))
		roleSvc := MustOpen(role.OpenService(ctx, role.ServiceConfig{DB: node.DB}))
		rackSvc = MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
			DB:      node.DB,
			HostKey: node.Identity.HostKey,
			Status:  stat,
			DriverRole: MustSucceed(
				racktestutil.CreateDriverRole(ctx, node.DB, roleSvc),
			),
		}))
		channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
			Channel: node.Channel,
			TS:      node.Storage.TS,
			DB:      node.DB,
			HostKey: node.Identity.HostKey,
			Status:  stat,
		}))
		graph := MustOpen(calcgraph.Open(ctx, calcgraph.Config{
			DB:      node.DB,
			Channel: channelSvc,
			Status:  stat,
		}))
		framerSvc = MustOpen(framer.OpenService(ctx, framer.ServiceConfig{
			DB:           node.DB,
			Framer:       node.Framer,
			Channel:      channelSvc,
			ChannelGraph: graph,
		}))
		sigs := MustSucceed(signals.New(signals.Config{
			DB:      node.DB,
			Channel: channelSvc,
			Framer:  node.Framer,
		}))
		svc = MustOpen(device.OpenService(ctx, device.ServiceConfig{
			DB:      node.DB,
			Status:  stat,
			Signals: sigs,
		}))
		MustOpen(sigs.Start(ctx))
	})

	openStreamer := func(
		ctx context.Context,
		name string,
	) confluence.Outlet[framer.StreamerResponse] {
		GinkgoHelper()
		var ch channel.Channel
		Expect(channelSvc.NewRetrieve().
			Where(channel.MatchNames(name)).
			Entry(&ch).
			Exec(ctx, node.DB)).To(Succeed())
		streamer := MustSucceed(framerSvc.NewStreamer(ctx, framer.StreamerConfig{
			Keys:        channel.Keys{ch.Key()},
			SendOpenAck: true,
		}))
		requests, responses := confluence.Attach(streamer, 2)
		sCtx, cancel := signal.Isolated()
		shutdown := signal.NewHardShutdown(sCtx, cancel)
		streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
		DeferCleanup(func() {
			requests.Close()
			confluence.Drain(responses)
			Expect(shutdown.Close()).To(Succeed())
		})
		Eventually(responses.Outlet(), 5*time.Second).Should(Receive())
		return responses
	}

	It("Should publish device sets and deletes", func(ctx SpecContext) {
		sets := openStreamer(ctx, "sy_ni_device_set")
		deletes := openStreamer(ctx, "sy_ni_device_delete")
		d := device.Device{
			Rack:         rackSvc.EmbeddedKey,
			Name:         "PXI-6255",
			ResourceName: "Mod1",
			Model:        "PXI-6255",
		}
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &d)
		})).To(Succeed())
		var res framer.StreamerResponse
		Eventually(sets.Outlet(), 5*time.Second).Should(Receive(&res))
		var written struct {
			Key uuid.UUID `json:"key"`
		}
		for sample := range res.Frame.SeriesAt(0).Samples() {
			Expect(json.Unmarshal(sample, &written)).To(Succeed())
		}
		Expect(written.Key).To(Equal(d.Key))
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Delete(ctx, d.Key)
		})).To(Succeed())
		Eventually(deletes.Outlet(), 5*time.Second).Should(Receive(&res))
		var removed []uuid.UUID
		for sample := range res.Frame.SeriesAt(0).Samples() {
			removed = append(removed, uuid.UUID(sample))
		}
		Expect(removed).To(ContainElement(d.Key))
	})

	It("Should broadcast the create action and each dispatch", func(ctx SpecContext) {
		acts := openStreamer(ctx, "sy_ni_device_action_set")
		d := device.Device{
			Rack:         rackSvc.EmbeddedKey,
			Name:         "PXI-6255",
			ResourceName: "Mod2",
			Model:        "PXI-6255",
		}
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &d)
		})).To(Succeed())
		next := func() actions.Scoped[device.Key, device.Action] {
			GinkgoHelper()
			var res framer.StreamerResponse
			Eventually(acts.Outlet(), 5*time.Second).Should(Receive(&res))
			var scoped actions.Scoped[device.Key, device.Action]
			for sample := range res.Frame.SeriesAt(0).Samples() {
				Expect(json.Unmarshal(sample, &scoped)).To(Succeed())
			}
			return scoped
		}
		created := next()
		Expect(created.Key).To(Equal(d.Key))
		Expect(created.Actions).To(HaveLen(1))
		Expect(created.Actions[0].Create.Device.Name).To(Equal("PXI-6255"))
		rename := device.NewSetAction(device.SetPayload{
			Path:  []string{"name"},
			Value: "Thermocouples",
		})
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).WithDispatchKey("dk").Apply(ctx, d.Key, rename)
		})).To(Succeed())
		dispatched := next()
		Expect(dispatched.Key).To(Equal(d.Key))
		Expect(dispatched.DispatchKey).To(Equal("dk"))
		Expect(dispatched.Actions).To(HaveLen(1))
		Expect(dispatched.Actions[0].Set.Path).To(Equal([]string{"name"}))
	})

	It("Should broadcast a group update as set actions", func(ctx SpecContext) {
		d := device.Device{
			Rack:         rackSvc.EmbeddedKey,
			Name:         "PXI-6255",
			ResourceName: "Mod3",
			Model:        "PXI-6255",
		}
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &d)
		})).To(Succeed())
		acts := openStreamer(ctx, "sy_ni_device_action_set")
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).ApplyUpdate(ctx, d.Key, device.Update{
				Variant: device.ConfigurationUpdate{
					Configuration: device.Configuration{Name: "Thermocouples"},
				},
			})
		})).To(Succeed())
		var res framer.StreamerResponse
		Eventually(acts.Outlet(), 5*time.Second).Should(Receive(&res))
		var scoped actions.Scoped[device.Key, device.Action]
		for sample := range res.Frame.SeriesAt(0).Samples() {
			Expect(json.Unmarshal(sample, &scoped)).To(Succeed())
		}
		Expect(scoped.Key).To(Equal(d.Key))
		paths := make([]string, len(scoped.Actions))
		for i, a := range scoped.Actions {
			paths[i] = a.Set.Path[0]
		}
		Expect(paths).To(ContainElements("name", "identifier", "analog_input"))
	})
})
