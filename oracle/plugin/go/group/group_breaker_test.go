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

var _ = Describe("Go Group Plugin breaker", func() {
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

	It(
		"Should put a member whose type is not its scope in the space of its scope",
		func() {
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
					@group namespace
				}

				@resource type "rack"
			}
		`, "rack", loader, p)
			ExpectContent(resp, "members.gen.go").
				ToContain("newSiblingIndex(resource.TypeDevice),")
			ExpectContent(resp, "siblings.gen.go").
				ToContain("group.SiblingSpace(resource.TypeDevice),")
		},
	)

	It("Should write one sibling file per package for two namespace members", func() {
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
					@group namespace
				}

				@resource type "rack"
			}

			Symbol struct {
				key   uuid {
					@key
				}
				name  string
				group group.Key? {
					@group scope schematic_symbol
					@group namespace
				}

				@resource type "schematic_symbol"
			}
		`, "rack", loader, p)
		paths := make(map[string]int)
		for _, f := range resp.Files {
			paths[f.Path]++
		}
		for path, n := range paths {
			Expect(n).To(Equal(1), "%s written %d times", path, n)
		}
	})
	It(
		"Should not import its own package for a namespace member in the group package",
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
				name  string
				group Key? {
					@group scope channel
					@group namespace
				}

				@resource type "channel"
			}
		`)
			resp := MustGenerateMulti(loader, p)
			ExpectContent(resp, "siblings.gen.go").
				ToBeValidGoSource().
				ToNotContain(`"github.com/synnaxlabs/synnax/core/pkg/service/group"`)
		},
	)
})
