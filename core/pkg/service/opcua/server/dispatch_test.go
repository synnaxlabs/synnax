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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/opcua/server"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Dispatch", func() {
	var (
		e env
		s server.Server
	)
	set := func(path []string, value any) server.Action {
		return server.NewSetAction(server.SetPayload{Path: path, Value: value})
	}
	dispatch := func(ctx context.Context, key server.Key, acts ...server.Action) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			return e.svc.NewWriter(tx).WithDispatchKey("dk").Apply(ctx, key, acts...)
		})
	}
	retrieve := func(ctx context.Context) server.Server {
		GinkgoHelper()
		var got server.Server
		Expect(e.svc.NewRetrieve().
			Where(server.MatchKeys(s.Key)).
			Entry(&got).
			Exec(ctx, db)).To(Succeed())
		return got
	}
	// Dispatch commits on its own, so the server commits too and is deleted after each
	// spec.
	BeforeEach(func(ctx SpecContext) {
		e = openEnv(ctx, db, nil)
		s = server.Server{
			Rack:       e.racks.EmbeddedKey,
			Name:       "Boiler PLC",
			Connection: newConnection("opc.tcp://boiler:4840"),
			Reads: []server.Read{
				{Node: "nsu=urn:plc;s=Temp", Index: 1, Channel: 2},
			},
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return e.svc.NewWriter(tx).Create(ctx, &s)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return e.svc.NewWriter(tx).Delete(ctx, s.Key)
			})).To(Succeed())
		})
	})

	It("Should rename the status when a dispatch renames the server", func(
		ctx SpecContext,
	) {
		Expect(dispatch(ctx, s.Key, set([]string{"name"}, "Mixer PLC"))).To(Succeed())
		var st server.Status
		Expect(e.status.NewRetrieve[server.StatusDetails]().
			Where(status.MatchOwners[server.StatusDetails](s.ResourceID())).
			Entry(&st).
			Exec(ctx, db)).To(Succeed())
		Expect(st.Name).To(Equal("Mixer PLC"))
	})

	It("Should keep edits to different fields from separate dispatches", func(
		ctx SpecContext,
	) {
		Expect(dispatch(ctx, s.Key, set([]string{"name"}, "Mixer PLC"))).To(Succeed())
		Expect(dispatch(
			ctx,
			s.Key,
			set([]string{"connection", "session_timeout"}, 30*telem.Minute),
		)).To(Succeed())
		got := retrieve(ctx)
		Expect(got.Name).To(Equal("Mixer PLC"))
		Expect(got.Connection.SessionTimeout).To(Equal(30 * telem.Minute))
		Expect(got.Connection.Endpoint).To(Equal(s.Connection.Endpoint))
		Expect(got.Reads).To(Equal(s.Reads))
	})

	It("Should store dispatched node IDs in the form the Driver writes", func(
		ctx SpecContext,
	) {
		Expect(dispatch(ctx, s.Key, set(
			[]string{"writes"},
			[]map[string]any{{"node": "NSU=urn:plc;s=Valve", "channel": 4}},
		))).To(Succeed())
		Expect(retrieve(ctx).Writes).To(Equal([]server.Write{
			{Node: "nsu=urn:plc;s=Valve", Channel: 4},
		}))
	})

	It("Should apply none of the actions when one fails", func(ctx SpecContext) {
		Expect(dispatch(
			ctx,
			s.Key,
			set([]string{"name"}, "Mixer PLC"),
			set([]string{"connection", "endpoint"}, "http://boiler"),
		)).To(MatchError(validate.ErrValidation))
		Expect(retrieve(ctx).Name).To(Equal("Boiler PLC"))
	})

	It("Should refuse a create action", func(ctx SpecContext) {
		Expect(dispatch(
			ctx,
			s.Key,
			server.NewCreateAction(server.CreatePayload{Server: s}),
		)).To(MatchError(validate.ErrValidation))
	})

	It("Should refuse a server that does not exist", func(ctx SpecContext) {
		Expect(dispatch(ctx, uuid.New(), set([]string{"name"}, "Renamed"))).
			To(MatchError(query.ErrNotFound))
	})
})
