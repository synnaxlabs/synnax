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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	fgorp "github.com/synnaxlabs/freighter/gorp"
	"github.com/synnaxlabs/gorp"
	apiimex "github.com/synnaxlabs/synnax/pkg/api/imex"
	apisymbol "github.com/synnaxlabs/synnax/pkg/api/schematic/symbol"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/encoding/zip"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

// symbolType is the type-wide symbol object Create enforces on.
var symbolType = resource.ID{Type: resource.TypeSchematicSymbol}

// inGroup returns an unsaved symbol in g.
func inGroup(name string, g group.Group) symbol.Symbol {
	sym := newSymbol(name)
	sym.Group = &g.Key
	return sym
}

var _ = Describe("Create", func() {
	It("Should create the symbols in their group when both are granted", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		g := createGroup(ctx, "creatable")
		grantCreateOn(ctx, u.ResourceID(), symbolType)
		grantUpdateOn(ctx, u.ResourceID(), g.ResourceID())
		res := MustSucceed(fgorp.CreateWriteUnaryHandler(db, apiSvc.Create)(
			AuthedCtx(ctx, u), apisymbol.CreateRequest{
				Symbols: []symbol.Symbol{inGroup("Inlet", g), inGroup("Outlet", g)},
			},
		))
		Expect(res.Symbols).To(HaveLen(2))
		Expect(symbolSvc.RetrieveGroupSymbols(ctx, db, g.Key)).To(ConsistOf(
			symbol.ResourceID(res.Symbols[0].Key),
			symbol.ResourceID(res.Symbols[1].Key),
		))
	})
	It("Should reject the request when create is not granted on the symbol type", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		g := createGroup(ctx, "uncreatable")
		grantUpdateOn(ctx, u.ResourceID(), g.ResourceID())
		Expect(
			fgorp.CreateWriteUnaryHandler(
				db,
				apiSvc.Create,
			)(
				AuthedCtx(ctx, u),
				apisymbol.CreateRequest{
					Symbols: []symbol.Symbol{inGroup("Inlet", g)},
				},
			),
		).Error().
			To(MatchError(access.ErrDenied))
	})
	It("Should reject the request when update is not granted on the group", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		g := createGroup(ctx, "ungranted-group")
		grantCreateOn(ctx, u.ResourceID(), symbolType)
		Expect(
			fgorp.CreateWriteUnaryHandler(
				db,
				apiSvc.Create,
			)(
				AuthedCtx(ctx, u),
				apisymbol.CreateRequest{
					Symbols: []symbol.Symbol{inGroup("Inlet", g)},
				},
			),
		).Error().
			To(MatchError(access.ErrDenied))
	})
})

var _ = Describe("SetGroup", func() {
	// denied matches an error that denies the subject the action on object.
	denied := func(
		subject resource.ID,
		action access.Action,
		object resource.ID,
	) types.GomegaMatcher {
		return SatisfyAll(
			MatchError(access.ErrDenied),
			MatchError(ContainSubstring("%s cannot %s %s", subject, action, object)),
		)
	}
	move := func(
		ctx SpecContext,
		u user.User,
		g group.Group,
		sym symbol.Symbol,
	) (struct{}, error) {
		return setGroup(AuthedCtx(ctx, u), apisymbol.SetGroupRequest{
			Group: &g.Key,
			Keys:  []symbol.Key{sym.Key},
		})
	}
	It(
		"Should refuse the move without update on the destination group",
		func(ctx SpecContext) {
			u := createUser(ctx)
			from := createGroup(ctx, "set-group-denied-from")
			to := createGroup(ctx, "set-group-denied-to")
			sym := createSymbol(ctx, from, "Inlet")
			grantUpdateOn(
				ctx,
				u.ResourceID(),
				symbol.ResourceID(sym.Key),
				from.ResourceID(),
			)
			Expect(move(ctx, u, to, sym)).
				Error().To(denied(u.ResourceID(), access.ActionUpdate, to.ResourceID()))
			Expect(symbolSvc.RetrieveGroupSymbols(ctx, db, from.Key)).To(
				ConsistOf(symbol.ResourceID(sym.Key)),
			)
		},
	)
	It(
		"Should refuse the move without update on the source group",
		func(ctx SpecContext) {
			u := createUser(ctx)
			from := createGroup(ctx, "set-group-source-denied-from")
			to := createGroup(ctx, "set-group-source-denied-to")
			sym := createSymbol(ctx, from, "Inlet")
			grantUpdateOn(
				ctx,
				u.ResourceID(),
				symbol.ResourceID(sym.Key),
				to.ResourceID(),
			)
			Expect(move(ctx, u, to, sym)).
				Error().
				To(denied(u.ResourceID(), access.ActionUpdate, from.ResourceID()))
			Expect(symbolSvc.RetrieveGroupSymbols(ctx, db, from.Key)).To(
				ConsistOf(symbol.ResourceID(sym.Key)),
			)
		},
	)
	It("Should refuse the move without update on the symbol", func(ctx SpecContext) {
		u := createUser(ctx)
		from := createGroup(ctx, "set-group-symbol-denied-from")
		to := createGroup(ctx, "set-group-symbol-denied-to")
		sym := createSymbol(ctx, from, "Inlet")
		grantUpdateOn(ctx, u.ResourceID(), from.ResourceID(), to.ResourceID())
		Expect(move(ctx, u, to, sym)).Error().To(denied(
			u.ResourceID(),
			access.ActionUpdate,
			symbol.ResourceID(sym.Key),
		))
		Expect(symbolSvc.RetrieveGroupSymbols(ctx, db, from.Key)).To(
			ConsistOf(symbol.ResourceID(sym.Key)),
		)
	})
	It(
		"Should move the symbol with update on the symbol and both groups",
		func(ctx SpecContext) {
			u := createUser(ctx)
			from := createGroup(ctx, "set-group-granted-from")
			to := createGroup(ctx, "set-group-granted-to")
			sym := createSymbol(ctx, from, "Inlet")
			grantUpdateOn(
				ctx,
				u.ResourceID(),
				symbol.ResourceID(sym.Key),
				from.ResourceID(),
				to.ResourceID(),
			)
			MustSucceed(move(ctx, u, to, sym))
			Expect(symbolSvc.RetrieveGroupSymbols(ctx, db, to.Key)).To(
				ConsistOf(symbol.ResourceID(sym.Key)),
			)
		},
	)
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
			author.ResourceID(),
			symbol.ResourceID(first.Key),
			symbol.ResourceID(second.Key),
		)
		res := MustSucceed(retrieve(
			AuthedCtx(ctx, author), apisymbol.RetrieveRequest{
				Where: &symbol.Where{
					Key: &filter.Set[symbol.Key]{
						In: []symbol.Key{first.Key, second.Key},
					},
				},
			},
		))
		Expect(res.Symbols).To(ConsistOf(
			HaveField("Key", first.Key), HaveField("Key", second.Key),
		))
	})
	It("Should return the symbols the search term matches", func(ctx SpecContext) {
		sym := createSymbol(ctx, createGroup(ctx, "searchable"), "Pyrometer")
		grantRetrieveOn(ctx, author.ResourceID(), symbol.ResourceID(sym.Key))
		Eventually(retrieve).WithArguments(
			AuthedCtx(ctx, author),
			apisymbol.RetrieveRequest{SearchTerm: "Pyrometer"},
		).Should(HaveField("Symbols", ConsistOf(HaveField("Key", sym.Key))))
	})
	It("Should return no symbols when a key names no symbol", func(ctx SpecContext) {
		missing := uuid.New()
		grantRetrieveOn(ctx, author.ResourceID(), symbol.ResourceID(missing))
		res := MustSucceed(
			retrieve(AuthedCtx(ctx, author), apisymbol.RetrieveRequest{
				Where: &symbol.Where{Key: &filter.Set[symbol.Key]{Eq: &missing}},
			}),
		)
		Expect(res.Symbols).To(BeEmpty())
	})
	It("Should reject the request when retrieve is not granted on a match", func(
		ctx SpecContext,
	) {
		g := createGroup(ctx, "unretrievable")
		sym := createSymbol(ctx, g, "Inlet")
		Expect(retrieve(AuthedCtx(ctx, author), apisymbol.RetrieveRequest{
			Where: &symbol.Where{Key: &filter.Set[symbol.Key]{Eq: &sym.Key}},
		})).Error().To(MatchError(access.ErrDenied))
	})
})

var _ = Describe("Rename", func() {
	It("Should rename the symbol when update is granted", func(ctx SpecContext) {
		g := createGroup(ctx, "renamable")
		sym := createSymbol(ctx, g, "Inlet")
		grantUpdateOn(ctx, author.ResourceID(), symbol.ResourceID(sym.Key))
		Expect(fgorp.CreateWriteUnaryHandler(db, apiSvc.Rename)(
			AuthedCtx(ctx, author),
			apisymbol.RenameRequest{Key: sym.Key, Name: "Outlet"},
		)).Error().ToNot(HaveOccurred())
		var renamed symbol.Symbol
		Expect(symbolSvc.NewRetrieve().
			Where(symbol.MatchKeys(sym.Key)).
			Entry(&renamed).
			Exec(ctx, db)).To(Succeed())
		Expect(renamed.Name).To(Equal("Outlet"))
	})
	It("Should reject the request when update is not granted", func(ctx SpecContext) {
		g := createGroup(ctx, "unrenamable")
		sym := createSymbol(ctx, g, "Inlet")
		Expect(fgorp.CreateWriteUnaryHandler(db, apiSvc.Rename)(
			AuthedCtx(ctx, author),
			apisymbol.RenameRequest{Key: sym.Key, Name: "Outlet"},
		)).Error().To(MatchError(access.ErrDenied))
	})
})

var _ = Describe("Delete", func() {
	It("Should delete the symbols when delete is granted", func(ctx SpecContext) {
		g := createGroup(ctx, "deletable-symbols")
		sym := createSymbol(ctx, g, "Inlet")
		grantDeleteOn(ctx, author.ResourceID(), symbol.ResourceID(sym.Key))
		Expect(fgorp.CreateWriteUnaryHandler(db, apiSvc.Delete)(
			AuthedCtx(ctx, author),
			apisymbol.DeleteRequest{Keys: []symbol.Key{sym.Key}},
		)).Error().ToNot(HaveOccurred())
		Expect(symbolSvc.NewRetrieve().
			Where(symbol.MatchKeys(sym.Key)).
			Entry(&symbol.Symbol{}).
			Exec(ctx, db)).To(MatchError(query.ErrNotFound))
	})
	It("Should reject the request when delete is not granted", func(ctx SpecContext) {
		g := createGroup(ctx, "undeletable-symbols")
		sym := createSymbol(ctx, g, "Inlet")
		Expect(fgorp.CreateWriteUnaryHandler(db, apiSvc.Delete)(
			AuthedCtx(ctx, author),
			apisymbol.DeleteRequest{Keys: []symbol.Key{sym.Key}},
		)).Error().To(MatchError(access.ErrDenied))
	})
})

var _ = Describe("ExportGroup", func() {
	It("Should export the bundle when retrieve is granted on the group and its members",
		func(ctx SpecContext) {
			g := createGroup(ctx, "granted")
			sym := createSymbol(ctx, g, "Inlet")
			grantRetrieveOn(
				ctx, author.ResourceID(), g.ResourceID(), symbol.ResourceID(sym.Key),
			)
			Expect(MustSucceed(exportGroup(
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
		grantRetrieveOn(ctx, author.ResourceID(), g.ResourceID())
		Expect(exportGroup(
			AuthedCtx(ctx, author),
			apisymbol.ExportGroupRequest{Key: g.Key, Encoding: "YAML"},
		)).Error().To(MatchError(ContainSubstring("unsupported encoding")))
	})
	It("Should reject the request when retrieve is not granted on the group", func(
		ctx SpecContext,
	) {
		g := createGroup(ctx, "ungranted-group")
		sym := createSymbol(ctx, g, "Outlet")
		grantRetrieveOn(ctx, author.ResourceID(), symbol.ResourceID(sym.Key))
		Expect(exportGroup(
			AuthedCtx(ctx, author),
			apisymbol.ExportGroupRequest{Key: g.Key, Encoding: apiimex.EncodingJSON},
		)).Error().To(MatchError(access.ErrDenied))
	})
	It("Should refuse the group before it names the child that is not a symbol", func(
		ctx SpecContext,
	) {
		g := createGroup(ctx, "ungranted-with-nested")
		nested := group.Group{Name: "Nested", Parent: new(g.ResourceID())}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return groupSvc.NewWriter(tx).Create(ctx, &nested)
		})).To(Succeed())
		Expect(exportGroup(
			AuthedCtx(ctx, createUser(ctx)),
			apisymbol.ExportGroupRequest{Key: g.Key, Encoding: apiimex.EncodingJSON},
		)).Error().To(MatchError(access.ErrDenied))
	})
	It("Should reject the request when retrieve is not granted on a member", func(
		ctx SpecContext,
	) {
		g := createGroup(ctx, "ungranted-member")
		createSymbol(ctx, g, "Vent")
		grantRetrieveOn(ctx, author.ResourceID(), g.ResourceID())
		Expect(exportGroup(
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
			ctx, author.ResourceID(), g.ResourceID(), symbol.ResourceID(sym.Key),
		)
		Expect(fgorp.CreateWriteUnaryHandler(db, apiSvc.DeleteGroup)(
			AuthedCtx(ctx, author), apisymbol.DeleteGroupRequest{Key: g.Key},
		)).Error().ToNot(HaveOccurred())
		Expect(symbolSvc.NewRetrieve().
			Where(symbol.MatchKeys(sym.Key)).
			Entry(&symbol.Symbol{}).
			Exec(ctx, db)).To(MatchError(query.ErrNotFound))
	})
	It("Should reject the request when the group is not granted", func(
		ctx SpecContext,
	) {
		g := createGroup(ctx, "ungranted-group")
		sym := createSymbol(ctx, g, "Outlet")
		grantDeleteOn(ctx, author.ResourceID(), symbol.ResourceID(sym.Key))
		Expect(fgorp.CreateWriteUnaryHandler(db, apiSvc.DeleteGroup)(
			AuthedCtx(ctx, author), apisymbol.DeleteGroupRequest{Key: g.Key},
		)).Error().To(MatchError(access.ErrDenied))
	})
	It("Should reject the request when a member is not granted", func(ctx SpecContext) {
		g := createGroup(ctx, "ungranted-member")
		createSymbol(ctx, g, "Vent")
		grantDeleteOn(ctx, author.ResourceID(), g.ResourceID())
		Expect(fgorp.CreateWriteUnaryHandler(db, apiSvc.DeleteGroup)(
			AuthedCtx(ctx, author), apisymbol.DeleteGroupRequest{Key: g.Key},
		)).Error().To(MatchError(access.ErrDenied))
	})
	It("Should refuse the group before it names the child that is not a symbol", func(
		ctx SpecContext,
	) {
		g := createGroup(ctx, "ungranted-with-nested")
		nested := group.Group{Name: "Nested", Parent: new(g.ResourceID())}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return groupSvc.NewWriter(tx).Create(ctx, &nested)
		})).To(Succeed())
		Expect(fgorp.CreateWriteUnaryHandler(db, apiSvc.DeleteGroup)(
			AuthedCtx(ctx, createUser(ctx)),
			apisymbol.DeleteGroupRequest{Key: g.Key},
		)).Error().To(MatchError(access.ErrDenied))
	})
})

// groupType is the type-wide group object ImportGroup enforces on.
var groupType = resource.ID{Type: resource.TypeGroup}

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
	It("Should import the bundle when create is granted on the types",
		func(ctx SpecContext) {
			u := createUser(ctx)
			grantCreateOn(ctx, u.ResourceID(), groupType, symbolType)
			res := MustSucceed(fgorp.CreateWriteUnaryHandler(db, apiSvc.ImportGroup)(
				AuthedCtx(ctx, u), newBundle("Valves"),
			))
			Expect(res.Group.Name).To(Equal("Valves"))
			Expect(symbolSvc.RetrieveGroupSymbols(ctx, db, res.Group.Key)).
				To(HaveLen(1))
		},
	)
	It("Should reject the request when create is not granted on the types", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		Expect(fgorp.CreateWriteUnaryHandler(db, apiSvc.ImportGroup)(
			AuthedCtx(ctx, u), newBundle("Valves"),
		)).Error().To(MatchError(access.ErrDenied))
	})
	It("Should reject the request when create is granted on one type alone", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		grantCreateOn(ctx, u.ResourceID(), symbolType)
		Expect(fgorp.CreateWriteUnaryHandler(db, apiSvc.ImportGroup)(
			AuthedCtx(ctx, u), newBundle("Valves"),
		)).Error().To(MatchError(access.ErrDenied))
	})
})
