// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/ts/internal/imports"
	"github.com/synnaxlabs/oracle/plugin/ts/types"
	"github.com/synnaxlabs/oracle/resolution"
	. "github.com/synnaxlabs/oracle/testutil"
)

var _ = Describe("FieldProcessor", func() {
	var (
		loader *MockFileLoader
		mgr    *imports.Manager
	)

	BeforeEach(func() {
		loader = NewMockFileLoader()
		mgr = imports.NewManager()
	})

	processorFor := func(ctx SpecContext, source string) (*types.FieldProcessor, resolution.Type, resolution.Field) {
		req := MustGenerateRequest(ctx, source, "schema", loader)
		fp := &types.FieldProcessor{
			Imports:    mgr,
			Namespace:  "schema",
			OutputPath: "client/ts/src/schema",
			Request:    req,
		}
		structs := req.Resolutions.StructTypes()
		Expect(structs).ToNot(BeEmpty())
		typ := structs[0]
		form := typ.Form.(resolution.StructForm)
		Expect(form.Fields).ToNot(BeEmpty())
		return fp, typ, form.Fields[0]
	}

	Describe("ProcessField", func() {
		It("Should produce zod and TS representations for a primitive field", func(ctx SpecContext) {
			source := `
				@ts output "client/ts/src/schema"

				Item struct {
					name string
				}
			`
			fp, typ, field := processorFor(ctx, source)
			fd := fp.ProcessField(field, typ)
			Expect(fd.Name).To(Equal("name"))
			Expect(fd.TSName).To(Equal("name"))
			Expect(fd.ZodType).To(ContainSubstring("z.string"))
		})

		It("Should add a same-namespace zod import when a field references another schema type", func(ctx SpecContext) {
			source := `
				@ts output "client/ts/src/schema"

				Tile struct {
					key string
				}

				Board struct {
					tile Tile
				}
			`
			req := MustGenerateRequest(ctx, source, "schema", loader)
			fp := &types.FieldProcessor{
				Imports:    mgr,
				Namespace:  "schema",
				OutputPath: "client/ts/src/schema",
				Request:    req,
			}
			structs := req.Resolutions.StructTypes()
			var board resolution.Type
			for _, s := range structs {
				if s.Name == "Board" {
					board = s
					break
				}
			}
			Expect(board.Name).To(Equal("Board"))
			tileField := board.Form.(resolution.StructForm).Fields[0]
			fp.CollectTypeImports(&tileField.Type)
			internal := mgr.InternalNamedImports()
			Expect(internal).ToNot(BeEmpty())
			Expect(internal[0].Names).To(ContainElement("tileZ"))
		})
	})

	Describe("CollectTypeImports", func() {
		It("Should be a no-op for type parameters", func() {
			fp := &types.FieldProcessor{
				Imports:    mgr,
				Namespace:  "schema",
				OutputPath: "client/ts/src/schema",
			}
			tp := resolution.TypeParam{Name: "T"}
			ref := resolution.TypeRef{Name: "T", TypeParam: &tp}
			fp.CollectTypeImports(&ref)
			Expect(mgr.InternalNamedImports()).To(BeNil())
			Expect(mgr.SynnaxImports()).To(BeNil())
		})

		It("Should recurse through type args", func(ctx SpecContext) {
			source := `
				@ts output "client/ts/src/schema"

				Tile struct {
					key string
				}

				Board struct {
					tiles Tile[]
				}
			`
			req := MustGenerateRequest(ctx, source, "schema", loader)
			fp := &types.FieldProcessor{
				Imports:    mgr,
				Namespace:  "schema",
				OutputPath: "client/ts/src/schema",
				Request:    req,
			}
			var board resolution.Type
			for _, s := range req.Resolutions.StructTypes() {
				if s.Name == "Board" {
					board = s
					break
				}
			}
			tilesField := board.Form.(resolution.StructForm).Fields[0]
			fp.CollectTypeImports(&tilesField.Type)
			internal := mgr.InternalNamedImports()
			Expect(internal).ToNot(BeEmpty())
			Expect(internal[0].Names).To(ContainElement("tileZ"))
		})

		It("Should not add imports for cross-namespace references", func(ctx SpecContext) {
			source := `
				@ts output "client/ts/src/schema"

				Item struct {
					name string
				}
			`
			req := MustGenerateRequest(ctx, source, "schema", loader)
			fp := &types.FieldProcessor{
				Imports:    mgr,
				Namespace:  "other",
				OutputPath: "client/ts/src/other",
				Request:    req,
			}
			ref := resolution.TypeRef{Name: "Item"}
			fp.CollectTypeImports(&ref)
			Expect(mgr.InternalNamedImports()).To(BeNil())
		})
	})
})
