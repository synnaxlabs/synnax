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
)

var _ = Describe("Module", func() {
	var (
		table   *resolution.Table
		thing   = resolution.TypeRef{Name: "thing.Thing"}
		rackKey = resolution.TypeRef{Name: "rack.Key"}
	)

	BeforeEach(func(ctx SpecContext) {
		loader := NewMockFileLoader()
		loader.Add(
			"schemas/rack",
			"@py output \"client/py/synnax/rack\"\n\nKey = uint32",
		)
		t, diag := Analyze(ctx, `
			import "schemas/rack"

			@py output "client/py/synnax/thing"

			Thing struct {
				rack rack.Key
			}
		`, "thing", loader)
		Expect(diag.Ok()).To(BeTrue())
		table = t
	})

	It("Should import a type of the namespace from types_gen.py", func() {
		m := types.NewModule("thing", "client/py/synnax/thing")
		Expect(m.Type(thing, table)).To(Equal("Thing"))
		Expect(m.Imports()).To(Equal("\nfrom synnax.thing.types_gen import Thing"))
	})

	It("Should import a type of another namespace through its module", func() {
		m := types.NewModule("thing", "client/py/synnax/thing")
		Expect(m.Type(rackKey, table)).To(Equal("rack.Key"))
		Expect(m.Imports()).To(Equal("\nfrom synnax import rack"))
	})

	It("Should alias a module that a top-level name of the module takes", func() {
		m := types.NewModule("thing", "client/py/synnax/thing", "rack")
		Expect(m.Type(rackKey, table)).To(Equal("rack_.Key"))
		Expect(m.Imports()).To(Equal("\nfrom synnax import rack as rack_"))
	})

	It("Should import nothing before a type is named", func() {
		m := types.NewModule("thing", "client/py/synnax/thing")
		Expect(m.Imports()).To(BeEmpty())
	})
})
