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
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	apiimex "github.com/synnaxlabs/synnax/pkg/api/imex"
	apisymbol "github.com/synnaxlabs/synnax/pkg/api/schematic/symbol"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/defaults"
	"github.com/synnaxlabs/x/encoding/zip"
	"github.com/synnaxlabs/x/filter"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// symbolType is the type-wide symbol object.
var symbolType = resource.ID{Type: resource.TypeSchematicSymbol}

// inGroup returns an unsaved symbol in g.
func inGroup(name string, g group.Group) symbol.Symbol {
	sym := newSymbol(name)
	sym.Group = &g.Key
	return sym
}

// denied matches an error that denies the subject the action on object.
func denied(
	subject resource.ID,
	action access.Action,
	object resource.ID,
) types.GomegaMatcher {
	return SatisfyAll(
		MatchError(access.ErrDenied),
		MatchError(ContainSubstring("%s cannot %s %s", subject, action, object)),
	)
}

// deniedOnType matches an error that denies the subject the action on a record of type
// t.
func deniedOnType(
	subject resource.ID,
	action access.Action,
	t resource.Type,
) types.GomegaMatcher {
	return SatisfyAll(
		MatchError(access.ErrDenied),
		MatchError(ContainSubstring("%s cannot %s %s:", subject, action, t)),
	)
}

// exists returns whether a symbol has key.
func exists(ctx SpecContext, key symbol.Key) bool {
	GinkgoHelper()
	return MustSucceed(symbolSvc.NewRetrieve().
		Where(symbol.MatchKeys(key)).
		Exists(ctx, db))
}

var _ = Describe("Create", func() {
	It("Should create the symbols in their group when both are granted", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		g := createGroup(ctx, "creatable")
		grantCreateOn(ctx, u.ResourceID(), symbolType)
		grantUpdateOn(ctx, u.ResourceID(), g.ResourceID())
		res := MustSucceed(create(AuthedCtx(ctx, u), apisymbol.CreateRequest{
			Symbols: defaults.Fill(
				[]symbol.Symbol{inGroup("Inlet", g), inGroup("Outlet", g)},
			),
		}))
		Expect(res.Symbols).To(HaveLen(2))
		Expect(groupSymbols(ctx, g.Key)).To(ConsistOf(
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
		Expect(create(AuthedCtx(ctx, u), apisymbol.CreateRequest{
			Symbols: defaults.Fill([]symbol.Symbol{inGroup("Inlet", g)}),
		})).Error().To(deniedOnType(
			u.ResourceID(),
			access.ActionCreate,
			resource.TypeSchematicSymbol,
		))
		Expect(groupSymbols(ctx, g.Key)).To(BeEmpty())
	})
	It("Should reject the request when update is not granted on the group", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		g := createGroup(ctx, "ungranted-group")
		grantCreateOn(ctx, u.ResourceID(), symbolType)
		Expect(create(AuthedCtx(ctx, u), apisymbol.CreateRequest{
			Symbols: defaults.Fill([]symbol.Symbol{inGroup("Inlet", g)}),
		})).Error().To(denied(u.ResourceID(), access.ActionUpdate, g.ResourceID()))
		Expect(groupSymbols(ctx, g.Key)).To(BeEmpty())
	})
	It("Should refuse a symbol whose key exists", func(ctx SpecContext) {
		u := createUser(ctx)
		g := createGroup(ctx, "refused")
		stored := createSymbol(ctx, g, "Inlet")
		grantCreateOn(ctx, u.ResourceID(), symbolType)
		grantUpdateOn(ctx, u.ResourceID(), g.ResourceID(), stored.ResourceID())
		again := stored
		again.Name = "Outlet"
		Expect(create(AuthedCtx(ctx, u), apisymbol.CreateRequest{
			Symbols: []symbol.Symbol{again},
		})).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(
				"schematic_symbol %s already exists",
				stored.Key,
			)),
		))
		var res symbol.Symbol
		Expect(symbolSvc.NewRetrieve().
			Where(symbol.MatchKeys(stored.Key)).
			Entry(&res).
			Exec(ctx, db)).To(Succeed())
		Expect(res.Name).To(Equal("Inlet"))
	})
})

var _ = Describe("Dispatch", func() {
	var (
		u        user.User
		from, to group.Group
		sym      symbol.Symbol
	)
	BeforeEach(func(ctx SpecContext) {
		u = createUser(ctx)
		from = createGroup(ctx, "dispatch-from")
		to = createGroup(ctx, "dispatch-to")
		sym = createSymbol(ctx, from, "Inlet")
	})
	move := func(ctx SpecContext) (struct{}, error) {
		return dispatch(AuthedCtx(ctx, u), apisymbol.DispatchRequest{
			Key: sym.Key,
			Actions: []symbol.Action{symbol.NewSetAction(symbol.SetPayload{
				Path:  []string{"group"},
				Value: to.Key,
			})},
		})
	}
	DescribeTable(
		"Should refuse a group change without update on a record it changes",
		func(ctx SpecContext, refused func() resource.ID) {
			changed := []resource.ID{
				symbol.ResourceID(sym.Key),
				from.ResourceID(),
				to.ResourceID(),
			}
			grantUpdateOn(ctx, u.ResourceID(), lo.Without(changed, refused())...)
			Expect(move(ctx)).Error().
				To(denied(u.ResourceID(), access.ActionUpdate, refused()))
			Expect(groupSymbols(ctx, from.Key)).
				To(ConsistOf(symbol.ResourceID(sym.Key)))
		},
		Entry("the symbol", func() resource.ID { return symbol.ResourceID(sym.Key) }),
		Entry("the source group", func() resource.ID { return from.ResourceID() }),
		Entry("the destination group", func() resource.ID { return to.ResourceID() }),
	)
	It("Should move the symbol with update on it and both groups", func(
		ctx SpecContext,
	) {
		grantUpdateOn(
			ctx,
			u.ResourceID(),
			symbol.ResourceID(sym.Key),
			from.ResourceID(),
			to.ResourceID(),
		)
		MustSucceed(move(ctx))
		Expect(groupSymbols(ctx, to.Key)).To(ConsistOf(symbol.ResourceID(sym.Key)))
	})
})

var _ = Describe("Retrieve", func() {
	It("Should leave out the symbols that the subject cannot retrieve", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		g := createGroup(ctx, "leave-out")
		granted := createSymbol(ctx, g, "leave-out-granted")
		createSymbol(ctx, g, "leave-out-denied")
		grantRetrieveOn(ctx, u.ResourceID(), granted.ResourceID())
		res := MustSucceed(retrieve(AuthedCtx(ctx, u), apisymbol.RetrieveRequest{}))
		Expect(res.Symbols).To(HaveExactElements(HaveField("Key", granted.Key)))
	})

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

var _ = Describe("Delete", func() {
	remove := func(ctx SpecContext, u user.User, sym symbol.Symbol) (struct{}, error) {
		return del(
			AuthedCtx(ctx, u),
			apisymbol.DeleteRequest{Keys: []symbol.Key{sym.Key}},
		)
	}
	It(
		"Should delete the symbol with delete on it and update on its group",
		func(ctx SpecContext) {
			u := createUser(ctx)
			g := createGroup(ctx, "deletable-symbols")
			sym := createSymbol(ctx, g, "Inlet")
			grantDeleteOn(ctx, u.ResourceID(), symbol.ResourceID(sym.Key))
			grantUpdateOn(ctx, u.ResourceID(), g.ResourceID())
			MustSucceed(remove(ctx, u, sym))
			Expect(exists(ctx, sym.Key)).To(BeFalse())
		},
	)
	It("Should reject the request when delete is not granted", func(ctx SpecContext) {
		u := createUser(ctx)
		g := createGroup(ctx, "undeletable-symbols")
		sym := createSymbol(ctx, g, "Inlet")
		grantUpdateOn(ctx, u.ResourceID(), g.ResourceID())
		Expect(remove(ctx, u, sym)).Error().To(denied(
			u.ResourceID(),
			access.ActionDelete,
			symbol.ResourceID(sym.Key),
		))
		Expect(exists(ctx, sym.Key)).To(BeTrue())
	})
	It(
		"Should reject the request when update is not granted on the group",
		func(ctx SpecContext) {
			u := createUser(ctx)
			g := createGroup(ctx, "unupdatable-symbol-group")
			sym := createSymbol(ctx, g, "Inlet")
			grantDeleteOn(ctx, u.ResourceID(), symbol.ResourceID(sym.Key))
			Expect(remove(ctx, u, sym)).Error().
				To(denied(u.ResourceID(), access.ActionUpdate, g.ResourceID()))
			Expect(exists(ctx, sym.Key)).To(BeTrue())
		},
	)
	It("Should need no access to delete a key that matches no symbol", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		MustSucceed(del(
			AuthedCtx(ctx, u),
			apisymbol.DeleteRequest{Keys: []symbol.Key{uuid.New()}},
		))
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
	removeGroup := func(ctx SpecContext, u user.User, g group.Group) (struct{}, error) {
		return deleteGroup(AuthedCtx(ctx, u), apisymbol.DeleteGroupRequest{Key: g.Key})
	}
	It(
		"Should delete the group and its symbols with delete on both alone",
		func(ctx SpecContext) {
			u := createUser(ctx)
			g := createGroup(ctx, "deletable")
			sym := createSymbol(ctx, g, "Inlet")
			grantDeleteOn(
				ctx, u.ResourceID(), g.ResourceID(), symbol.ResourceID(sym.Key),
			)
			MustSucceed(removeGroup(ctx, u, g))
			Expect(exists(ctx, sym.Key)).To(BeFalse())
		},
	)
	It("Should reject the request when delete is not granted on the group", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		g := createGroup(ctx, "ungranted-group")
		sym := createSymbol(ctx, g, "Outlet")
		grantDeleteOn(ctx, u.ResourceID(), symbol.ResourceID(sym.Key))
		Expect(removeGroup(ctx, u, g)).Error().
			To(denied(u.ResourceID(), access.ActionDelete, g.ResourceID()))
		Expect(exists(ctx, sym.Key)).To(BeTrue())
	})
	It("Should reject the request when a member is not granted", func(ctx SpecContext) {
		u := createUser(ctx)
		g := createGroup(ctx, "ungranted-member")
		sym := createSymbol(ctx, g, "Vent")
		grantDeleteOn(ctx, u.ResourceID(), g.ResourceID())
		Expect(removeGroup(ctx, u, g)).Error().To(denied(
			u.ResourceID(),
			access.ActionDelete,
			symbol.ResourceID(sym.Key),
		))
		Expect(exists(ctx, sym.Key)).To(BeTrue())
	})
	It(
		"Should refuse a group that holds another group before any access check",
		func(ctx SpecContext) {
			g := createGroup(ctx, "ungranted-with-nested")
			nested := group.Group{Name: "Nested", Parent: new(g.ResourceID())}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return groupSvc.NewWriter(tx).Create(ctx, &nested)
			})).To(Succeed())
			Expect(removeGroup(ctx, createUser(ctx), g)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("not empty")),
			))
		},
	)
})

// groupType is the type-wide group object.
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
	It("Should import the bundle with create on the types alone", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		grantCreateOn(ctx, u.ResourceID(), groupType, symbolType)
		res := MustSucceed(importGroup(AuthedCtx(ctx, u), newBundle("Valves")))
		Expect(res.Group.Name).To(Equal("Valves"))
		Expect(groupSymbols(ctx, res.Group.Key)).To(HaveLen(1))
	})
	It("Should reject the request when create is not granted on the types", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		Expect(importGroup(AuthedCtx(ctx, u), newBundle("Valves"))).Error().
			To(deniedOnType(u.ResourceID(), access.ActionCreate, resource.TypeGroup))
	})
	It("Should reject the request when create is granted on one type alone", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		grantCreateOn(ctx, u.ResourceID(), groupType)
		Expect(importGroup(AuthedCtx(ctx, u), newBundle("Valves"))).Error().To(
			deniedOnType(
				u.ResourceID(),
				access.ActionCreate,
				resource.TypeSchematicSymbol,
			),
		)
	})
})
