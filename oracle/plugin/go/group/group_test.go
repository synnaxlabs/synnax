// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package group_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/go/group"
	. "github.com/synnaxlabs/oracle/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

func TestGoGroup(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plugin Go Group Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var _ = Describe("Go Group Plugin", func() {
	var (
		loader *MockFileLoader
		p      *group.Plugin
	)

	BeforeEach(func() {
		loader = NewMockFileLoader()
		p = group.New(group.DefaultOptions())
		loader.Add("schemas/resource", `
			@go output "core/pkg/service/resource"

			Type enum {
				channel          = "channel"
				device           = "device"
				group            = "group"
				rack             = "rack"
				schematic_symbol = "schematic_symbol"
			}
		`)
		loader.Add("schemas/group", `
			import "schemas/resource"

			@go output "core/pkg/service/group"

			Key = uuid

			Group struct {
				key   Key {
					@key
				}
				scope resource.Type

				@resource type "group"
			}
		`)
	})

	It("Should have the name, domains, and requirements of a Go plugin", func() {
		Expect(p.Name()).To(Equal("go/group"))
		Expect(p.Domains()).To(Equal([]string{"go"}))
		Expect(p.Requires()).To(Equal([]string{"go/types"}))
	})

	It("Should emit no files when no field carries @group", func(ctx SpecContext) {
		resp := MustGenerate(ctx, `
			import "schemas/group"

			@go output "core/pkg/service/rack"

			Rack struct {
				key   uint32 {
					@key
				}
				group group.Key?

				@resource type "rack"
			}
		`, "rack", loader, p)
		Expect(resp.Files).To(BeEmpty())
	})

	It("Should emit the rule of each member next to the group", func(ctx SpecContext) {
		resp := MustGenerate(ctx, `
			import "schemas/group"

			@go output "core/pkg/service/rack"

			Rack struct {
				key   uint32 {
					@key
				}
				group group.Key? {
					@group scope device
					@group pinned
				}

				@resource type "rack"
			}

			Symbol struct {
				key   uuid {
					@key
				}
				group group.Key? {
					@group scope schematic_symbol
				}

				@resource type "schematic_symbol"
			}
		`, "rack", loader, p)
		Expect(resp.Files).To(HaveLen(1))
		Expect(resp.Files[0].Path).To(Equal("core/pkg/service/group/members.gen.go"))
		ExpectContent(resp, "members.gen.go").
			ToBeValidGoSource().
			ToContain(
				"package group",
				`"github.com/synnaxlabs/synnax/core/pkg/service/resource"`,
				"scope resource.Type",
				"var memberRules = map[resource.Type]memberRule{",
				"resource.TypeRack: {scope: resource.TypeDevice, pinned: true},",
				"resource.TypeSchematicSymbol: {scope: resource.TypeSchematicSymbol},",
			).
			ToPreserveOrder("resource.TypeRack:", "resource.TypeSchematicSymbol:")
	})

	It(
		"Should name the resource types unqualified in their own package",
		func(ctx SpecContext) {
			loader.Add("schemas/group", `
			@go output "core/pkg/service/group"

			Key = uuid

			Type enum {
				channel = "channel"
			}

			Group struct {
				key   Key {
					@key
				}
				scope Type
			}

			Channel struct {
				key   uint32 {
					@key
				}
				group Key? {
					@group scope channel
				}

				@resource type "channel"
			}
		`)
			resp := MustGenerateMulti(ctx, loader, p)
			ExpectContent(resp, "members.gen.go").
				ToBeValidGoSource().
				ToContain("TypeChannel: {scope: TypeChannel},").
				ToNotContain("import")
		},
	)

	It("Should return the error of a malformed @group", func(ctx SpecContext) {
		req := MustGenerateRequest(ctx, `
			import "schemas/group"

			Rack struct {
				key   uint32 {
					@key
				}
				group group.Key? {
					@group pinned
				}

				@resource type "rack"
			}
		`, "rack", loader)
		Expect(p.Generate(req)).Error().
			To(MatchError("rack.Rack.group: @group needs a scope"))
	})
})
