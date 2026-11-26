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
	"fmt"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	apiimex "github.com/synnaxlabs/synnax/pkg/api/imex"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	. "github.com/synnaxlabs/synnax/pkg/service/imex/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	v13 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v13"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/defaults"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Import of an old file with a symbol copy", func() {
	var (
		u   user.User
		g   group.Group
		sym symbol.Symbol
	)
	BeforeEach(func(ctx SpecContext) {
		u = newUser(ctx)
		g = group.Group{Name: "Library", Scope: resource.TypeSchematicSymbol}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return groupSvc.NewWriter(tx).Create(ctx, &g)
		})).To(Succeed())
		sym = defaults.Fill(symbol.Symbol{
			Name:  "Tank",
			Group: &g.Key,
			Data: symbol.Spec{
				Drawing: symbol.Drawing{Variant: symbol.SvgDrawing{Markup: "<svg/>"}},
				States: []symbol.State{
					{Variant: symbol.EnumerationState{Key: "active"}},
				},
			},
		})
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return schematicSvc.Symbol.NewWriter(tx).Create(ctx, &sym)
		})).To(Succeed())
		grantOn(ctx, u.ResourceID(), access.ActionCreate, resource.ID{
			Type: resource.TypeSchematic,
		})
		grantUpdateOn(ctx, u.ResourceID(), project.ResourceID(proj.Key))
		grantOn(ctx, u.ResourceID(), access.ActionRetrieve, sym.ResourceID())
	})
	// importPanel imports, as u, an old file whose custom actuator draws sym, which
	// does not actuate, so the import copies sym into its group.
	importPanel := func(ctx SpecContext) (apiimex.ImportResponse, error) {
		GinkgoHelper()
		var cfg v13.CustomActuatorNodeConfig
		cfg.SetDefaults()
		cfg.SpecKey = sym.Key.String()
		env := imex.Envelope{Version: v13.Version, Type: "schematic", Name: "Panel"}
		Expect(env.Encode(v13.Schematic{
			Key:   uuid.New(),
			Name:  "Panel",
			Nodes: []v13.Node{{Key: "a", Config: v13.NodeConfig{Variant: cfg}}},
		})).To(Succeed())
		fctx := AuthedCtx(ctx, u)
		fctx.Set("params", fmt.Sprintf(
			`{"file_name":"Panel.json","parent":"project:%s"}`, proj.Key,
		))
		return importFile(fctx, WireRoundTrip(env))
	}
	symbolType := resource.ID{Type: resource.TypeSchematicSymbol}

	It("Should copy the symbol with create on symbols and update on its group", func(
		ctx SpecContext,
	) {
		grantOn(ctx, u.ResourceID(), access.ActionCreate, symbolType)
		grantUpdateOn(ctx, u.ResourceID(), g.ResourceID())
		Expect(MustSucceed(importPanel(ctx)).Type).To(Equal(resource.TypeSchematic))
		Expect(schematicSvc.Symbol.NewRetrieve().
			Where(symbol.MatchGroups(g.Key)).
			Count(ctx, db)).To(Equal(2))
	})

	It("Should refuse the import without create on symbols", func(ctx SpecContext) {
		grantUpdateOn(ctx, u.ResourceID(), g.ResourceID())
		Expect(importPanel(ctx)).Error().To(SatisfyAll(
			MatchError(access.ErrDenied),
			MatchError(ContainSubstring(
				"%s cannot create %s:", u.ResourceID(), resource.TypeSchematicSymbol,
			)),
		))
	})

	It("Should refuse the import without update on the group of the symbol", func(
		ctx SpecContext,
	) {
		grantOn(ctx, u.ResourceID(), access.ActionCreate, symbolType)
		Expect(importPanel(ctx)).Error().To(SatisfyAll(
			MatchError(access.ErrDenied),
			MatchError(ContainSubstring(
				"%s cannot update %s", u.ResourceID(), g.ResourceID(),
			)),
		))
	})
})
