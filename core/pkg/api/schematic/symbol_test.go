// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package schematic_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/synnaxlabs/gorp"
	apisymbol "github.com/synnaxlabs/synnax/pkg/api/schematic/symbol"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/defaults"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Symbol save", func() {
	var (
		u   user.User
		sym symbol.Symbol
		sc  schematic.Schematic
	)
	BeforeEach(func(ctx SpecContext) {
		u = newUser(ctx)
		sym = defaults.Fill(symbol.Symbol{
			Name: "Valve",
			Data: symbol.Spec{
				Drawing: symbol.Drawing{Variant: symbol.SvgDrawing{Markup: "<svg/>"}},
				Commands: []symbol.Command{{
					Key:  "vent",
					Name: "Vent",
					Value: symbol.CommandValue{
						Variant: symbol.FixedCommandValue{Value: 1},
					},
				}},
			},
		})
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return schematicSvc.Symbol.NewWriter(tx).Create(ctx, &sym)
		})).To(Succeed())
		sc = schematic.Schematic{
			Project: &proj.Key,
			Name:    "Plant",
			Nodes: []schematic.Node{{
				Key: "valve",
				Config: defaults.Fill(schematic.NodeConfig{
					Variant: schematic.SymbolNodeConfig{
						Spec: symbol.Ref{Variant: symbol.CustomRef{Key: sym.Key}},
						Commands: []schematic.CommandBinding{
							{Key: "vent", Channel: 1},
						},
					},
				}),
			}},
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return schematicSvc.NewWriter(tx).Create(ctx, &sc)
		})).To(Succeed())
		grantUpdateOn(ctx, u.ResourceID(), sym.ResourceID())
	})
	// save sets the field of the symbol at path to value as u.
	save := func(ctx SpecContext, path []string, value any) (struct{}, error) {
		return dispatchSymbol(AuthedCtx(ctx, u), apisymbol.DispatchRequest{
			Key: sym.Key,
			Actions: []symbol.Action{symbol.NewSetAction(symbol.SetPayload{
				Path:  path,
				Value: value,
			})},
		})
	}
	stored := func(ctx SpecContext) symbol.Symbol {
		GinkgoHelper()
		var res symbol.Symbol
		Expect(schematicSvc.Symbol.NewRetrieve().
			Where(symbol.MatchKeys(sym.Key)).
			Entry(&res).
			Exec(ctx, db)).To(Succeed())
		return res
	}
	// bound matches the stored schematic when its node binds commands that match m.
	bound := func(m types.GomegaMatcher) types.GomegaMatcher {
		return HaveField("Nodes", HaveExactElements(
			HaveField("Config.Variant", HaveField("Commands", m)),
		))
	}

	It("Should refuse a save that changes a schematic the subject cannot update", func(
		ctx SpecContext,
	) {
		Expect(save(ctx, []string{"data", "commands"}, []any{})).Error().
			To(denied(u, access.ActionUpdate, sc.ResourceID()))
		Expect(stored(ctx).Data.Commands).To(HaveLen(1))
		Expect(retrieveSchematic(ctx, sc.Key)).To(bound(HaveLen(1)))
	})

	It("Should save a change that changes no schematic", func(ctx SpecContext) {
		MustSucceed(save(ctx, []string{"name"}, "Vent valve"))
		Expect(stored(ctx).Name).To(Equal("Vent valve"))
		Expect(retrieveSchematic(ctx, sc.Key)).To(bound(HaveLen(1)))
	})

	It("Should remove the orphans of a save with update on the schematic", func(
		ctx SpecContext,
	) {
		grantUpdateOn(ctx, u.ResourceID(), sc.ResourceID())
		MustSucceed(save(ctx, []string{"data", "commands"}, []any{}))
		Expect(retrieveSchematic(ctx, sc.Key)).To(bound(BeEmpty()))
	})
})
