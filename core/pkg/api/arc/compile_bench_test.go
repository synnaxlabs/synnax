// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	"testing"

	"github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/gorp"
	apiarc "github.com/synnaxlabs/synnax/pkg/api/arc"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	apichannel "github.com/synnaxlabs/synnax/pkg/api/channel"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	disttestutil "github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	calctestutil "github.com/synnaxlabs/synnax/pkg/service/calculation/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// openCompiledLayer opens a service layer over a new node, closed when b ends.
func openCompiledLayer(b *testing.B) (*gorp.DB, *service.Layer) {
	b.Helper()
	gomega.RegisterTestingT(b)
	node := disttestutil.OpenNode(b.Context())
	layer := MustSucceed(svctestutil.OpenLayer(b.Context(), node))
	b.Cleanup(func() {
		gomega.Expect(layer.Close()).To(gomega.Succeed())
		gomega.Expect(node.Close()).To(gomega.Succeed())
	})
	return node.DB, layer
}

// compiledRoute returns the retrieve route that compiles each automation it returns, as
// the API layer binds it.
func compiledRoute(
	b *testing.B,
	db *gorp.DB,
	layer *service.Layer,
) freighter.UnaryHandler[apiarc.RetrieveRequest, apiarc.RetrieveResponse] {
	b.Helper()
	apiChannelSvc := MustSucceed(apichannel.NewService(apichannel.ServiceConfig{
		DB:          db,
		RBAC:        layer.RBAC,
		Channel:     layer.Channel,
		Status:      layer.Status,
		Calculation: layer.Calculation,
	}))
	binder := MustSucceed(bind.NewService(bind.ServiceConfig{
		DB:   db,
		RBAC: layer.RBAC,
		Held: apiauth.HeldTables(),
	}))
	svc := MustSucceed(apiarc.NewService(apiarc.ServiceConfig{
		DB:              db,
		Arc:             layer.Arc,
		Status:          layer.Status,
		Channel:         layer.Channel,
		Calculation:     layer.Calculation,
		RBAC:            layer.RBAC,
		DispatchChannel: bind.Write(binder, apiChannelSvc.Dispatch),
		Stopper:         layer.TaskStopper,
	}))
	return svc.Compiled(bind.Read(binder, svc.Retrieve))
}

// createReadingAutomation creates an automation that reads a calculated channel and a
// user that can retrieve it, and returns a request for its program and the user.
func createReadingAutomation(
	b *testing.B,
	db *gorp.DB,
	layer *service.Layer,
) (apiarc.RetrieveRequest, user.User) {
	b.Helper()
	ctx := b.Context()
	base := channel.Channel{Name: "bench_base", DataType: telem.Int64T, Virtual: true}
	out := channel.Channel{Name: "bench_out", DataType: telem.Int64T, Virtual: true}
	gomega.Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return layer.Channel.NewWriter(tx).
			CreateMany(ctx, &[]channel.Channel{base, out})
	})).To(gomega.Succeed())
	creator := calctestutil.Creator{
		DB:          db,
		Channel:     layer.Channel,
		Calculation: layer.Calculation,
	}
	MustSucceed(creator.Create(ctx, calctestutil.Calculation{
		Name:       "bench_calc",
		DataType:   telem.Int64T,
		Expression: "return bench_base + 1",
	}))
	a := arc.Arc{Variant: arc.AutomationArc{Base: arc.Base{
		Name: "bench_automation",
		Source: arc.Source{Variant: arc.TextSource{
			Text: text.Text{Raw: "bench_calc -> bench_out"},
		}},
	}}}
	gomega.Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return layer.Arc.NewWriter(tx).Create(ctx, &a)
	})).To(gomega.Succeed())
	u := MustSucceed(rbactestutil.CreateUser(
		ctx,
		db,
		layer.RBAC,
		layer.User,
		user.User{Name: "bench"},
	))
	actions := []access.Action{access.ActionRetrieve}
	MustSucceed(
		rbactestutil.Grant(
			ctx,
			db,
			layer.RBAC,
			layer.User,
			u.Key,
			actions,
			a.ResourceID(),
		),
	)
	key := a.Base().Key
	return apiarc.RetrieveRequest{
		Where:          &arc.Where{Key: &filter.Set[arc.Key]{Eq: &key}},
		IncludeProgram: true,
	}, u
}

// BenchmarkCompiled measures a retrieve of one automation with its program, which
// compiles the automation and the calculated channel that it reads.
func BenchmarkCompiled(b *testing.B) {
	db, layer := openCompiledLayer(b)
	retrieve := compiledRoute(b, db, layer)
	req, u := createReadingAutomation(b, db, layer)
	subject := AuthedCtx(b.Context(), u)
	b.ReportAllocs()
	for b.Loop() {
		res := MustSucceed(retrieve(subject, req))
		if len(res.Arcs) != 1 {
			b.Fatalf("expected one automation, got %d", len(res.Arcs))
		}
	}
}
