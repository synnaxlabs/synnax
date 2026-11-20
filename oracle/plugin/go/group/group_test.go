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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/go/group"
	. "github.com/synnaxlabs/oracle/testutil"
)

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

	It("Should emit no files when no field carries @group", func() {
		resp := MustGenerate(`
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

	It("Should emit the rule of each member next to the group", func() {
		resp := MustGenerate(`
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
		func() {
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
			resp := MustGenerateMulti(loader, p)
			ExpectContent(resp, "members.gen.go").
				ToBeValidGoSource().
				ToContain("TypeChannel: {scope: TypeChannel},").
				ToNotContain("import")
		},
	)

	It("Should emit the sibling indexes of a namespace member", func() {
		resp := MustGenerate(`
			import "schemas/group"

			@go output "core/pkg/service/channel"

			Key = uint32

			Channel struct {
				key   Key {
					@key
				}
				name  string
				group group.Key? {
					@group scope channel
					@group namespace
				}

				@resource type "channel"
			}
		`, "channel", loader, p)
		Expect(resp.Files).To(HaveLen(2))
		ExpectContent(resp, "members.gen.go").
			ToBeValidGoSource().
			ToContain(
				"namespace bool",
				"resource.TypeChannel: {scope: resource.TypeChannel, namespace: true},",
				`"github.com/synnaxlabs/gorp"`,
				"func siblingIndexes() []gorp.Index[Key, Group] {",
				"newSiblingIndex(resource.TypeChannel),",
			)
		Expect(resp.Files[1].Path).To(Equal("core/pkg/service/channel/siblings.gen.go"))
		ExpectContent(resp, "siblings.gen.go").
			ToBeValidGoSource().
			ToContain(
				"package channel",
				`"github.com/synnaxlabs/synnax/core/pkg/service/group"`,
				"func newSiblingIndex() *gorp.UniqueIndex[Key, Channel, group.Sibling] {",
				"group.SiblingSpace(resource.TypeChannel),",
				"s := group.Sibling{Name: e.Name}",
				"s.Parent = *e.Group",
			)
	})

	It("Should take the key type of a computed key from @retrieve key", func() {
		resp := MustGenerate(`
			import "schemas/group"

			@go output "core/pkg/service/channel"

			Key = uint32

			Channel struct {
				name  string
				group group.Key? {
					@group scope channel
					@group namespace
				}

				@resource type "channel"
				@retrieve key  "Key"
			}
		`, "channel", loader, p)
		ExpectContent(resp, "siblings.gen.go").
			ToBeValidGoSource().
			ToContain("*gorp.UniqueIndex[Key, Channel, group.Sibling]")
	})

	It("Should refuse a namespace member with no key", func() {
		req := MustGenerateRequest(`
			import "schemas/group"

			@go output "core/pkg/service/channel"

			Channel struct {
				name  string
				group group.Key? {
					@group scope channel
					@group namespace
				}

				@resource type "channel"
			}
		`, "channel", loader)
		Expect(p.Generate(req)).Error().To(MatchError(ContainSubstring(
			"@group namespace needs channel.Channel to have a @key field or a " +
				"@retrieve key",
		)))
	})

	It("Should emit no sibling indexes when no member is a namespace", func() {
		resp := MustGenerate(`
			import "schemas/group"

			@go output "core/pkg/service/rack"

			Rack struct {
				key   uint32 {
					@key
				}
				group group.Key? {
					@group scope device
				}

				@resource type "rack"
			}
		`, "rack", loader, p)
		Expect(resp.Files).To(HaveLen(1))
		ExpectContent(resp, "members.gen.go").
			ToBeValidGoSource().
			ToNotContain("siblingIndexes", "gorp")
	})

	It("Should return the error of a malformed @group", func() {
		req := MustGenerateRequest(`
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
