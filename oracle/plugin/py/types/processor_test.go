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
	"github.com/synnaxlabs/oracle/plugin/py/types"
	"github.com/synnaxlabs/oracle/resolution"
	. "github.com/synnaxlabs/oracle/testutil"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("FieldProcessor", func() {
	var loader *MockFileLoader

	BeforeEach(func() { loader = NewMockFileLoader() })

	processorFor := func(
		ctx SpecContext,
		source string,
	) (*types.FieldProcessor, resolution.Type) {
		GinkgoHelper()
		req := MustGenerateRequest(ctx, source, "test", loader)
		fp := types.NewFieldProcessor("test", "client/py/synnax/test", req.Resolutions)
		return fp, req.Resolutions.MustGet("test.Item")
	}

	field := func(item resolution.Type, name string) resolution.Field {
		GinkgoHelper()
		return MustBeOk(item.Form.(resolution.StructForm).Field(name))
	}

	It("Should declare a field with its type and default", func(ctx SpecContext) {
		fp, item := processorFor(ctx, `
			@py output "client/py/synnax/test"

			Item struct {
				path string[] { @doc value "is the path." }
			}
		`)
		fd := fp.ProcessField(field(item, "path"))
		Expect(fd.Name).To(Equal("path"))
		Expect(fd.PyType).To(Equal("list[str]"))
		Expect(fd.Default).To(Equal(" = Field(default_factory=list)"))
		Expect(fd.Doc).To(Equal("is the path."))
	})

	It("Should import a type of the namespace from the types module", func(
		ctx SpecContext,
	) {
		fp, item := processorFor(ctx, `
			@py output "client/py/synnax/test"

			Tile struct {
				key string
			}

			Item struct {
				tiles Tile[]
			}
		`)
		Expect(fp.ProcessField(field(item, "tiles")).PyType).To(Equal("list[Tile]"))
		Expect(fp.Imports()).To(
			ContainSubstring("from synnax.test.types_gen import Tile"),
		)
	})

	It("Should alias a module import that a reserved name shadows", func(
		ctx SpecContext,
	) {
		loader.Add("schemas/color", `
			@py output "x/py/x/color"

			Color struct {
				hex string
			}
		`)
		fp, item := processorFor(ctx, `
			import "schemas/color"

			@py output "client/py/synnax/test"

			Item struct {
				color color.Color
			}
		`)
		fp.Reserve("color")
		Expect(fp.ProcessField(field(item, "color")).PyType).To(Equal("color_.Color"))
		Expect(fp.Imports()).To(ContainSubstring("from x import color as color_"))
	})

	It("Should alias a module import that a reserved field shadows", func(
		ctx SpecContext,
	) {
		loader.Add("schemas/color", `
			@py output "x/py/x/color"

			Color struct {
				hex string
			}
		`)
		fp, item := processorFor(ctx, `
			import "schemas/color"

			@py output "client/py/synnax/test"

			Item struct {
				color color.Color
			}
		`)
		fp.ReserveField(field(item, "color"))
		Expect(fp.ProcessField(field(item, "color")).PyType).To(Equal("color_.Color"))
	})

	It("Should import the typing and pydantic names it is given", func(
		ctx SpecContext,
	) {
		fp, _ := processorFor(ctx, `
			@py output "client/py/synnax/test"

			Item struct {
				key string
			}
		`)
		fp.AddTyping("Literal")
		fp.AddPydantic("BaseModel")
		Expect(fp.Imports()).To(SatisfyAll(
			ContainSubstring("from typing import Literal"),
			ContainSubstring("from pydantic import BaseModel"),
		))
	})

	It("Should fail on a namespace type whose name the module binds", func(
		ctx SpecContext,
	) {
		fp, item := processorFor(ctx, `
			@py output "client/py/synnax/test"

			Tile struct {
				key string
			}

			Item struct {
				tile Tile
			}
		`)
		fp.Reserve("Tile")
		fp.ProcessField(field(item, "tile"))
		Expect(fp.Imports()).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("test.Tile")),
		))
	})

	It("Should fail on a hand-written type with no include module", func(
		ctx SpecContext,
	) {
		fp, item := processorFor(ctx, `
			@py output "client/py/synnax/test"

			Tile struct {
				key string

				@py hand
			}

			Item struct {
				tile Tile
			}
		`)
		fp.ProcessField(field(item, "tile"))
		Expect(fp.Imports()).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("need a @py include module: test.Tile")),
		))
	})
})
