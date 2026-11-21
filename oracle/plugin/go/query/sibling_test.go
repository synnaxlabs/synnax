// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package query_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/go/query"
	. "github.com/synnaxlabs/oracle/testutil"
)

var _ = Describe("@group namespaced", func() {
	var (
		loader *MockFileLoader
		p      *query.Plugin
	)
	BeforeEach(func() {
		loader = NewMockFileLoader()
		p = query.New(query.DefaultOptions())
		loader.Add("schemas/resource", `
			@go output "core/pkg/service/resource"

			Type enum {
				channel = "channel"
				device  = "device"
				group   = "group"
				rack    = "rack"
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
				name  string
				scope resource.Type

				@resource type "group"
			}
		`)
	})

	It("Should build the sibling index in the indexes bundle", func() {
		resp := MustGenerate(`
			import "schemas/group"

			@go output "core/pkg/service/channel"

			Key = uint32

			Channel struct {
				key   Key {
					@key
				}
				name  string {
					@index sorted
				}
				group group.Key? {
					@group scope channel
					@group namespaced
				}

				@resource type "channel"
				@retrieve
			}
		`, "channel", loader, p)
		ExpectContent(resp, "retrieve.gen.go").
			ToBeValidGoSource().
			ToContain(
				`"github.com/synnaxlabs/synnax/core/pkg/service/group"`,
				"sibling *gorp.UniqueIndex[Key, Channel, group.Sibling]",
				"sibling: group.NewMemberIndex[Key, Channel](",
				"resource.TypeChannel,",
				"s := group.Sibling{Name: e.Name}",
				"s.Parent = *e.Group",
				"i.sibling,",
			)
	})

	It("Should take the space from the scope, not the resource type", func() {
		resp := MustGenerate(`
			import "schemas/group"

			@go output "core/pkg/service/rack"

			Rack struct {
				key   uint32 {
					@key
				}
				name  string
				group group.Key? {
					@group scope device
					@group namespaced
				}

				@resource type "rack"
				@retrieve
			}
		`, "rack", loader, p)
		ExpectContent(resp, "retrieve.gen.go").
			ToContain("resource.TypeDevice,").
			ToNotContain("resource.TypeRack,")
	})

	It("Should refuse two namespaced members in one package", func() {
		req := MustGenerateRequest(`
			import "schemas/group"

			@go output "core/pkg/service/rack"

			Rack struct {
				key   uint32 {
					@key
				}
				name  string
				group group.Key? {
					@group scope device
					@group namespaced
				}

				@resource type "rack"
				@retrieve
			}

			Device struct {
				key   string {
					@key
				}
				name  string
				group group.Key? {
					@group scope device
					@group namespaced
				}

				@resource type "device"
				@retrieve
			}
		`, "rack", loader)
		Expect(p.Generate(req)).Error().To(MatchError(ContainSubstring(
			"only one @retrieve type is allowed per output, found 2",
		)))
	})

	It("Should not import its own package for a member in the group package", func() {
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
				name  string
				scope Type
			}

			Channel struct {
				key   uint32 {
					@key
				}
				name  string
				group Key? {
					@group scope channel
					@group namespaced
				}

				@resource type "channel"
				@retrieve
			}
		`)
		resp := MustGenerateMulti(loader, p)
		ExpectContent(resp, "retrieve.gen.go").
			ToBeValidGoSource().
			ToContain("sibling: NewMemberIndex[uint32, Channel](", "TypeChannel,").
			ToNotContain(`"github.com/synnaxlabs/synnax/core/pkg/service/group"`)
	})

	It("Should build no sibling index for a member that is not namespaced", func() {
		resp := MustGenerate(`
			import "schemas/group"

			@go output "core/pkg/service/rack"

			Rack struct {
				key   uint32 {
					@key
				}
				name  string {
					@index sorted
				}
				group group.Key? {
					@group scope device
				}

				@resource type "rack"
				@retrieve
			}
		`, "rack", loader, p)
		ExpectContent(resp, "retrieve.gen.go").ToNotContain("sibling", "NewMemberIndex")
	})
})
