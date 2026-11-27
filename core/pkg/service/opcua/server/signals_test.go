// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package server_test

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
	"github.com/synnaxlabs/synnax/pkg/service/certificate"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channeltestutil "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/opcua/server"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	secrettestutil "github.com/synnaxlabs/synnax/pkg/service/secret/testutil"
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
		svc        *server.Service
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
		secrets := secrettestutil.OpenService(ctx, node.DB)
		svc = MustOpen(server.OpenService(ctx, server.ServiceConfig{
			DB:     node.DB,
			Status: stat,
			Secret: secrets,
			Certificate: MustOpen(
				certificate.OpenService(ctx, certificate.ServiceConfig{
					DB:      node.DB,
					Secret:  secrets,
					Status:  stat,
					Rack:    rackSvc.EmbeddedKey,
					HostKey: node.Identity.HostKey,
				}),
			),
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

	It("Should publish server sets and deletes", func(ctx SpecContext) {
		sets := openStreamer(ctx, "sy_opcua_server_set")
		deletes := openStreamer(ctx, "sy_opcua_server_delete")
		s := server.Server{
			Key:        uuid.New(),
			Rack:       rackSvc.EmbeddedKey,
			Name:       "Boiler PLC",
			Connection: newConnection("opc.tcp://boiler:4840"),
		}
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return gorp.NewCreate[server.Key, server.Server]().Entry(&s).Exec(ctx, tx)
		})).To(Succeed())
		var res framer.StreamerResponse
		Eventually(sets.Outlet(), 5*time.Second).Should(Receive(&res))
		var written struct {
			Key uuid.UUID `json:"key"`
		}
		for sample := range res.Frame.SeriesAt(0).Samples() {
			Expect(json.Unmarshal(sample, &written)).To(Succeed())
		}
		Expect(written.Key).To(Equal(s.Key))
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return gorp.NewDelete[server.Key, server.Server]().
				Where(gorp.MatchKeys[server.Key, server.Server](s.Key)).
				Exec(ctx, tx)
		})).To(Succeed())
		Eventually(deletes.Outlet(), 5*time.Second).Should(Receive(&res))
		var removed []uuid.UUID
		for sample := range res.Frame.SeriesAt(0).Samples() {
			removed = append(removed, uuid.UUID(sample))
		}
		Expect(removed).To(ContainElement(s.Key))
	})

	It("Should broadcast the create action and each dispatch", func(ctx SpecContext) {
		acts := openStreamer(ctx, "sy_opcua_server_action_set")
		s := server.Server{
			Rack:       rackSvc.EmbeddedKey,
			Name:       "Boiler PLC",
			Connection: newConnection("opc.tcp://boiler:4840"),
		}
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &s)
		})).To(Succeed())
		next := func() actions.Scoped[server.Key, server.Action] {
			GinkgoHelper()
			var res framer.StreamerResponse
			Eventually(acts.Outlet(), 5*time.Second).Should(Receive(&res))
			var scoped actions.Scoped[server.Key, server.Action]
			for sample := range res.Frame.SeriesAt(0).Samples() {
				Expect(json.Unmarshal(sample, &scoped)).To(Succeed())
			}
			return scoped
		}
		created := next()
		Expect(created.Key).To(Equal(s.Key))
		Expect(created.Actions).To(HaveLen(1))
		Expect(created.Actions[0].Create.Server.Name).To(Equal("Boiler PLC"))
		cmd := MustSucceed(
			channeltestutil.CreateVirtual(ctx, node.DB, channelSvc, 1),
		)[0]
		writes := server.NewSetAction(server.SetPayload{
			Path:  []string{"writes"},
			Value: []map[string]any{{"node": "NSU=urn:plc;s=Valve", "channel": cmd}},
		})
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			_, err := svc.NewWriter(tx).WithDispatchKey("dk").Apply(ctx, s.Key, writes)
			return err
		})).To(Succeed())
		dispatched := next()
		Expect(dispatched.Key).To(Equal(s.Key))
		Expect(dispatched.DispatchKey).To(Equal("dk"))
		Expect(dispatched.Actions).To(HaveLen(2))
		Expect(dispatched.Actions[1].Set.Path).To(Equal([]string{"writes"}))
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).ApplyUpdate(ctx, s.Key, server.Update{
				Variant: server.ConfigurationUpdate{},
			})
		})).To(Succeed())
		updated := next()
		Expect(len(updated.Actions)).To(BeNumerically(">=", 2))
		Expect(updated.Actions[0].Set.Path).To(Equal([]string{"reads"}))
		Expect(updated.Actions[1].Set.Path).To(Equal([]string{"writes"}))
	})
})
