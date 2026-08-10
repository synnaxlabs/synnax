// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("ExportGroup", func() {
	// Writes commit immediately so the api enforcers, which read committed state, can
	// observe the new ontology resources.
	createGroup := func(ctx SpecContext, name string) group.Group {
		GinkgoHelper()
		return MustSucceed(groupSvc.NewWriter(nil).Create(ctx, name, ontology.RootID))
	}
	createSymbol := func(ctx SpecContext, g group.Group, name string) symbol.Symbol {
		GinkgoHelper()
		sym := symbol.Symbol{
			Name: name,
			Data: symbol.Spec{SVG: "<svg/>", Variant: "valve"},
		}
		Expect(symbolSvc.NewWriter(nil).Create(ctx, &sym, g.OntologyID())).To(Succeed())
		return sym
	}

	It("Should export the bundle when retrieve is granted on the group and its members",
		func(ctx SpecContext) {
			g := createGroup(ctx, "granted")
			sym := createSymbol(ctx, g, "Inlet")
			grantRetrieveOn(
				ctx, author.OntologyID(), g.OntologyID(), symbol.OntologyID(sym.Key),
			)
			bundle := MustSucceed(apiSvc.ExportGroup(
				authedCtx(ctx, author), ExportGroupRequest{Key: g.Key},
			))
			Expect(bundle.Files).To(HaveKey("manifest.json"))
			Expect(bundle.Files).To(HaveKey("Inlet.json"))
		},
	)
	It("Should reject the request when retrieve is not granted on the group", func(
		ctx SpecContext,
	) {
		g := createGroup(ctx, "ungranted-group")
		sym := createSymbol(ctx, g, "Outlet")
		grantRetrieveOn(ctx, author.OntologyID(), symbol.OntologyID(sym.Key))
		Expect(apiSvc.ExportGroup(
			authedCtx(ctx, author), ExportGroupRequest{Key: g.Key},
		)).Error().To(MatchError(access.ErrDenied))
	})
	It("Should reject the request when retrieve is not granted on a member", func(
		ctx SpecContext,
	) {
		g := createGroup(ctx, "ungranted-member")
		createSymbol(ctx, g, "Vent")
		grantRetrieveOn(ctx, author.OntologyID(), g.OntologyID())
		Expect(apiSvc.ExportGroup(
			authedCtx(ctx, author), ExportGroupRequest{Key: g.Key},
		)).Error().To(MatchError(access.ErrDenied))
	})
})

var _ = Describe("DeleteGroup", func() {
	createGroup := func(ctx SpecContext, name string) group.Group {
		GinkgoHelper()
		return MustSucceed(groupSvc.NewWriter(nil).Create(ctx, name, ontology.RootID))
	}
	createSymbol := func(ctx SpecContext, g group.Group, name string) symbol.Symbol {
		GinkgoHelper()
		sym := symbol.Symbol{
			Name: name,
			Data: symbol.Spec{SVG: "<svg/>", Variant: "valve"},
		}
		Expect(symbolSvc.NewWriter(nil).Create(ctx, &sym, g.OntologyID())).To(Succeed())
		return sym
	}

	It("Should delete the group and its symbols when both are granted", func(
		ctx SpecContext,
	) {
		g := createGroup(ctx, "deletable")
		sym := createSymbol(ctx, g, "Inlet")
		grantDeleteOn(
			ctx, author.OntologyID(), g.OntologyID(), symbol.OntologyID(sym.Key),
		)
		Expect(apiSvc.DeleteGroup(
			authedCtx(ctx, author), nil, DeleteGroupRequest{Key: g.Key},
		)).Error().ToNot(HaveOccurred())
		Expect(symbolSvc.NewRetrieve().
			Where(symbol.MatchKeys(sym.Key)).
			Entry(&symbol.Symbol{}).
			Exec(ctx, nil)).To(MatchError(query.ErrNotFound))
	})
	It("Should reject the request when the group is not granted", func(
		ctx SpecContext,
	) {
		g := createGroup(ctx, "ungranted-group")
		sym := createSymbol(ctx, g, "Outlet")
		grantDeleteOn(ctx, author.OntologyID(), symbol.OntologyID(sym.Key))
		Expect(apiSvc.DeleteGroup(
			authedCtx(ctx, author), nil, DeleteGroupRequest{Key: g.Key},
		)).Error().To(MatchError(access.ErrDenied))
	})
	It("Should reject the request when a member is not granted", func(ctx SpecContext) {
		g := createGroup(ctx, "ungranted-member")
		createSymbol(ctx, g, "Vent")
		grantDeleteOn(ctx, author.OntologyID(), g.OntologyID())
		Expect(apiSvc.DeleteGroup(
			authedCtx(ctx, author), nil, DeleteGroupRequest{Key: g.Key},
		)).Error().To(MatchError(access.ErrDenied))
	})
})
