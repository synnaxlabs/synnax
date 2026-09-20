// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiimex "github.com/synnaxlabs/synnax/pkg/api/imex"
	apisymbol "github.com/synnaxlabs/synnax/pkg/api/schematic/symbol"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	"github.com/synnaxlabs/x/encoding/zip"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

// symbolType is the type-wide symbol object Create enforces on.
var symbolType = ontology.ID{Type: ontology.ResourceTypeSchematicSymbol}

var _ = Describe("Create", func() {
	It("Should create the symbols under the parent when both are granted", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		g := createGroup(ctx, "creatable")
		grantCreateOn(ctx, u.OntologyID(), symbolType)
		grantUpdateOn(ctx, u.OntologyID(), g.OntologyID())
		res := MustSucceed(apiSvc.Create(
			AuthedCtx(ctx, u), nil, apisymbol.CreateRequest{
				Parent:  g.OntologyID(),
				Symbols: []symbol.Symbol{newSymbol("Inlet"), newSymbol("Outlet")},
			},
		))
		Expect(res.Symbols).To(HaveLen(2))
		Expect(symbolSvc.RetrieveGroupSymbols(ctx, nil, g.Key)).To(ConsistOf(
			symbol.OntologyID(res.Symbols[0].Key),
			symbol.OntologyID(res.Symbols[1].Key),
		))
	})
	It("Should reject the request when create is not granted on the symbol type", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		g := createGroup(ctx, "uncreatable")
		grantUpdateOn(ctx, u.OntologyID(), g.OntologyID())
		Expect(apiSvc.Create(AuthedCtx(ctx, u), nil, apisymbol.CreateRequest{
			Parent:  g.OntologyID(),
			Symbols: []symbol.Symbol{newSymbol("Inlet")},
		})).Error().To(MatchError(access.ErrDenied))
	})
	It("Should reject the request when update is not granted on the parent", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		g := createGroup(ctx, "ungranted-parent")
		grantCreateOn(ctx, u.OntologyID(), symbolType)
		Expect(apiSvc.Create(AuthedCtx(ctx, u), nil, apisymbol.CreateRequest{
			Parent:  g.OntologyID(),
			Symbols: []symbol.Symbol{newSymbol("Inlet")},
		})).Error().To(MatchError(access.ErrDenied))
	})
})

var _ = Describe("Retrieve", func() {
	It("Should return the symbols the keys name when retrieve is granted", func(
		ctx SpecContext,
	) {
		g := createGroup(ctx, "retrievable")
		first := createSymbol(ctx, g, "Inlet")
		second := createSymbol(ctx, g, "Outlet")
		createSymbol(ctx, g, "Vent")
		grantRetrieveOn(
			ctx,
			author.OntologyID(),
			symbol.OntologyID(first.Key),
			symbol.OntologyID(second.Key),
		)
		res := MustSucceed(apiSvc.Retrieve(
			AuthedCtx(ctx, author), apisymbol.RetrieveRequest{
				Keys: []symbol.Key{first.Key, second.Key},
			},
		))
		Expect(res.Symbols).To(ConsistOf(
			HaveField("Key", first.Key), HaveField("Key", second.Key),
		))
	})
	It("Should return the symbols the search term matches", func(ctx SpecContext) {
		sym := createSymbol(ctx, createGroup(ctx, "searchable"), "Pyrometer")
		grantRetrieveOn(ctx, author.OntologyID(), symbol.OntologyID(sym.Key))
		Eventually(func(g Gomega) {
			res, err := apiSvc.Retrieve(
				AuthedCtx(ctx, author), apisymbol.RetrieveRequest{
					SearchTerm: "Pyrometer",
				},
			)
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(res.Symbols).To(ConsistOf(HaveField("Key", sym.Key)))
		}).Should(Succeed())
	})
	It("Should return not found when a key names no symbol", func(ctx SpecContext) {
		Expect(apiSvc.Retrieve(AuthedCtx(ctx, author), apisymbol.RetrieveRequest{
			Keys: []symbol.Key{uuid.New()},
		})).Error().To(MatchError(query.ErrNotFound))
	})
	It("Should reject the request when retrieve is not granted on a match", func(
		ctx SpecContext,
	) {
		g := createGroup(ctx, "unretrievable")
		sym := createSymbol(ctx, g, "Inlet")
		Expect(apiSvc.Retrieve(AuthedCtx(ctx, author), apisymbol.RetrieveRequest{
			Keys: []symbol.Key{sym.Key},
		})).Error().To(MatchError(access.ErrDenied))
	})
})

var _ = Describe("Rename", func() {
	It("Should rename the symbol when update is granted", func(ctx SpecContext) {
		g := createGroup(ctx, "renamable")
		sym := createSymbol(ctx, g, "Inlet")
		grantUpdateOn(ctx, author.OntologyID(), symbol.OntologyID(sym.Key))
		Expect(apiSvc.Rename(
			AuthedCtx(ctx, author), nil,
			apisymbol.RenameRequest{Key: sym.Key, Name: "Outlet"},
		)).Error().ToNot(HaveOccurred())
		var renamed symbol.Symbol
		Expect(symbolSvc.NewRetrieve().
			Where(symbol.MatchKeys(sym.Key)).
			Entry(&renamed).
			Exec(ctx, nil)).To(Succeed())
		Expect(renamed.Name).To(Equal("Outlet"))
	})
	It("Should reject the request when update is not granted", func(ctx SpecContext) {
		g := createGroup(ctx, "unrenamable")
		sym := createSymbol(ctx, g, "Inlet")
		Expect(apiSvc.Rename(
			AuthedCtx(ctx, author), nil,
			apisymbol.RenameRequest{Key: sym.Key, Name: "Outlet"},
		)).Error().To(MatchError(access.ErrDenied))
	})
})

var _ = Describe("Delete", func() {
	It("Should delete the symbols when delete is granted", func(ctx SpecContext) {
		g := createGroup(ctx, "deletable-symbols")
		sym := createSymbol(ctx, g, "Inlet")
		grantDeleteOn(ctx, author.OntologyID(), symbol.OntologyID(sym.Key))
		Expect(apiSvc.Delete(
			AuthedCtx(ctx, author), nil,
			apisymbol.DeleteRequest{Keys: []symbol.Key{sym.Key}},
		)).Error().ToNot(HaveOccurred())
		Expect(symbolSvc.NewRetrieve().
			Where(symbol.MatchKeys(sym.Key)).
			Entry(&symbol.Symbol{}).
			Exec(ctx, nil)).To(MatchError(query.ErrNotFound))
	})
	It("Should reject the request when delete is not granted", func(ctx SpecContext) {
		g := createGroup(ctx, "undeletable-symbols")
		sym := createSymbol(ctx, g, "Inlet")
		Expect(apiSvc.Delete(
			AuthedCtx(ctx, author), nil,
			apisymbol.DeleteRequest{Keys: []symbol.Key{sym.Key}},
		)).Error().To(MatchError(access.ErrDenied))
	})
})

var _ = Describe("RetrieveGroup", func() {
	It("Should return the permanent symbol group when retrieve is granted", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		grantRetrieveOn(ctx, u.OntologyID(), symbolSvc.Group().OntologyID())
		Expect(MustSucceed(apiSvc.RetrieveGroup(
			AuthedCtx(ctx, u), apisymbol.RetrieveGroupRequest{},
		)).Group).To(Equal(symbolSvc.Group()))
	})
	It("Should reject the request when retrieve is not granted", func(ctx SpecContext) {
		Expect(apiSvc.RetrieveGroup(
			AuthedCtx(ctx, createUser(ctx)), apisymbol.RetrieveGroupRequest{},
		)).Error().To(MatchError(access.ErrDenied))
	})
})

var _ = Describe("ExportGroup", func() {
	It("Should export the bundle when retrieve is granted on the group and its members",
		func(ctx SpecContext) {
			g := createGroup(ctx, "granted")
			sym := createSymbol(ctx, g, "Inlet")
			grantRetrieveOn(
				ctx, author.OntologyID(), g.OntologyID(), symbol.OntologyID(sym.Key),
			)
			Expect(MustSucceed(apiSvc.ExportGroup(
				AuthedCtx(ctx, author),
				apisymbol.ExportGroupRequest{
					Key:      g.Key,
					Encoding: apiimex.EncodingJSON,
				},
			))).To(SatisfyAll(HaveKey("manifest.json"), HaveKey("Inlet.json")))
		},
	)
	It("Should reject an unsupported encoding", func(ctx SpecContext) {
		g := createGroup(ctx, "bad-encoding")
		grantRetrieveOn(ctx, author.OntologyID(), g.OntologyID())
		Expect(apiSvc.ExportGroup(
			AuthedCtx(ctx, author),
			apisymbol.ExportGroupRequest{Key: g.Key, Encoding: "YAML"},
		)).Error().To(MatchError(ContainSubstring("unsupported encoding")))
	})
	It("Should reject the request when retrieve is not granted on the group", func(
		ctx SpecContext,
	) {
		g := createGroup(ctx, "ungranted-group")
		sym := createSymbol(ctx, g, "Outlet")
		grantRetrieveOn(ctx, author.OntologyID(), symbol.OntologyID(sym.Key))
		Expect(apiSvc.ExportGroup(
			AuthedCtx(ctx, author),
			apisymbol.ExportGroupRequest{Key: g.Key, Encoding: apiimex.EncodingJSON},
		)).Error().To(MatchError(access.ErrDenied))
	})
	It("Should refuse the group before it names the child that is not a symbol", func(
		ctx SpecContext,
	) {
		g := createGroup(ctx, "ungranted-with-nested")
		MustSucceed(groupSvc.NewWriter(nil).Create(ctx, "Nested", g.OntologyID()))
		Expect(apiSvc.ExportGroup(
			AuthedCtx(ctx, createUser(ctx)),
			apisymbol.ExportGroupRequest{Key: g.Key, Encoding: apiimex.EncodingJSON},
		)).Error().To(MatchError(access.ErrDenied))
	})
	It("Should reject the request when retrieve is not granted on a member", func(
		ctx SpecContext,
	) {
		g := createGroup(ctx, "ungranted-member")
		createSymbol(ctx, g, "Vent")
		grantRetrieveOn(ctx, author.OntologyID(), g.OntologyID())
		Expect(apiSvc.ExportGroup(
			AuthedCtx(ctx, author),
			apisymbol.ExportGroupRequest{Key: g.Key, Encoding: apiimex.EncodingJSON},
		)).Error().To(MatchError(access.ErrDenied))
	})
})

var _ = Describe("DeleteGroup", func() {
	It("Should delete the group and its symbols when both are granted", func(
		ctx SpecContext,
	) {
		g := createGroup(ctx, "deletable")
		sym := createSymbol(ctx, g, "Inlet")
		grantDeleteOn(
			ctx, author.OntologyID(), g.OntologyID(), symbol.OntologyID(sym.Key),
		)
		Expect(apiSvc.DeleteGroup(
			AuthedCtx(ctx, author), nil, apisymbol.DeleteGroupRequest{Key: g.Key},
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
			AuthedCtx(ctx, author), nil, apisymbol.DeleteGroupRequest{Key: g.Key},
		)).Error().To(MatchError(access.ErrDenied))
	})
	It("Should reject the request when a member is not granted", func(ctx SpecContext) {
		g := createGroup(ctx, "ungranted-member")
		createSymbol(ctx, g, "Vent")
		grantDeleteOn(ctx, author.OntologyID(), g.OntologyID())
		Expect(apiSvc.DeleteGroup(
			AuthedCtx(ctx, author), nil, apisymbol.DeleteGroupRequest{Key: g.Key},
		)).Error().To(MatchError(access.ErrDenied))
	})
	It("Should refuse the group before it names the child that is not a symbol", func(
		ctx SpecContext,
	) {
		g := createGroup(ctx, "ungranted-with-nested")
		MustSucceed(groupSvc.NewWriter(nil).Create(ctx, "Nested", g.OntologyID()))
		Expect(apiSvc.DeleteGroup(
			AuthedCtx(ctx, createUser(ctx)), nil,
			apisymbol.DeleteGroupRequest{Key: g.Key},
		)).Error().To(MatchError(access.ErrDenied))
	})
})

// groupType is the type-wide group object ImportGroup enforces on.
var groupType = ontology.ID{Type: ontology.ResourceTypeGroup}

// newBundle returns a version 2 symbol group bundle holding one symbol.
func newBundle(name string) zip.Files {
	return zip.Files{
		"manifest.json": []byte(
			`{"version":2,"type":"symbol_group","name":"` + name + `"}`,
		),
		"Inlet.json": []byte(
			`{"version":2,"type":"schematic_symbol","name":"Inlet",` +
				`"data":{"svg":"<svg/>","variant":"valve"}}`,
		),
	}
}

var _ = Describe("ImportGroup", func() {
	It("Should import the bundle when the types and the permanent group are granted",
		func(ctx SpecContext) {
			u := createUser(ctx)
			grantCreateOn(ctx, u.OntologyID(), groupType, symbolType)
			grantUpdateOn(ctx, u.OntologyID(), symbolSvc.Group().OntologyID())
			res := MustSucceed(apiSvc.ImportGroup(
				AuthedCtx(ctx, u), nil, newBundle("Valves"),
			))
			Expect(res.Group.Name).To(Equal("Valves"))
			Expect(symbolSvc.RetrieveGroupSymbols(ctx, nil, res.Group.Key)).
				To(HaveLen(1))
		},
	)
	It("Should reject the request when create is not granted on the types", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		grantUpdateOn(ctx, u.OntologyID(), symbolSvc.Group().OntologyID())
		Expect(apiSvc.ImportGroup(
			AuthedCtx(ctx, u), nil, newBundle("Valves"),
		)).Error().To(MatchError(access.ErrDenied))
	})
	It("Should reject the request when create is granted on one type alone", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		grantCreateOn(ctx, u.OntologyID(), symbolType)
		grantUpdateOn(ctx, u.OntologyID(), symbolSvc.Group().OntologyID())
		Expect(apiSvc.ImportGroup(
			AuthedCtx(ctx, u), nil, newBundle("Valves"),
		)).Error().To(MatchError(access.ErrDenied))
	})
	It("Should reject the request when update is not granted on the permanent group",
		func(ctx SpecContext) {
			u := createUser(ctx)
			grantCreateOn(ctx, u.OntologyID(), groupType, symbolType)
			Expect(apiSvc.ImportGroup(
				AuthedCtx(ctx, u), nil, newBundle("Valves"),
			)).Error().To(MatchError(access.ErrDenied))
		},
	)
})
