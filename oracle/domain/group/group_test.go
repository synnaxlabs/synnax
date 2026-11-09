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
	"github.com/synnaxlabs/oracle/domain/group"
	. "github.com/synnaxlabs/oracle/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Extract", func() {
	var loader *MockFileLoader
	BeforeEach(func() {
		loader = NewMockFileLoader()
		loader.Add("schemas/resource", `
			Type enum {
				channel = "channel"
				device  = "device"
				group   = "group"
				label   = "label"
				rack    = "rack"
			}
		`)
		loader.Add("schemas/group", `
			import "schemas/resource"

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

	extract := func(ctx SpecContext, records string) (*group.Data, error) {
		GinkgoHelper()
		table, diag := Analyze(ctx, `
			import "schemas/group"
		`+records, "records", loader)
		Expect(diag.Ok()).To(BeTrue(), diag.String)
		return group.Extract(table)
	}

	rack := func(domains string) string {
		return `
			Rack struct {
				key   uint32 {
					@key
				}
				group group.Key? {
					` + domains + `
				}

				@resource type "rack"
			}
		`
	}

	It("Should return nil when no field carries @group", func(ctx SpecContext) {
		Expect(extract(ctx, rack(""))).To(BeNil())
	})

	It("Should return the members ordered by resource type", func(ctx SpecContext) {
		data := MustSucceed(extract(ctx, rack(`
			@group scope device
			@group pinned
		`)+`
			Channel struct {
				key   uint32 {
					@key
				}
				group group.Key? {
					@group scope channel
				}

				@resource type "channel"
			}
		`))
		Expect(data.Record.Name).To(Equal("Group"))
		Expect(data.Enum.Name).To(Equal("Type"))
		Expect(data.Members).To(HaveLen(2))
		values := func(m group.Member) []any {
			return []any{m.Type.StringValue(), m.Scope.StringValue(), m.Pinned}
		}
		Expect(values(data.Members[0])).To(Equal([]any{"channel", "channel", false}))
		Expect(values(data.Members[1])).To(Equal([]any{"rack", "device", true}))
	})

	DescribeTable(
		"Should reject a malformed @group",
		func(ctx SpecContext, domains, message string) {
			Expect(extract(ctx, rack(domains))).Error().
				To(MatchError("records.Rack.group: " + message))
		},
		Entry("with no scope", "@group pinned", "@group needs a scope"),
		Entry(
			"with an unknown scope",
			"@group scope shelf",
			"@group has scope shelf, which resource.Type does not hold",
		),
		Entry(
			"with two scopes",
			"@group scope device rack",
			"@group needs one value for scope, not 2",
		),
		Entry(
			"with a value on pinned",
			"@group scope device\n@group pinned true",
			"@group needs no value for pinned",
		),
		Entry(
			"with an unknown expression",
			"@group scope device\n@group nested",
			"@group has unknown expression nested",
		),
	)

	It("Should skip a type that inherits a marked field", func(ctx SpecContext) {
		data := MustSucceed(extract(ctx, rack("@group scope device")+`
			New struct extends Rack {}
		`))
		Expect(data.Members).To(HaveLen(1))
	})

	It("Should reject a type with no resource type", func(ctx SpecContext) {
		Expect(extract(ctx, `
			Shelf struct {
				key   uint32 {
					@key
				}
				group group.Key? {
					@group scope device
				}
			}
		`)).Error().To(MatchError(
			"records.Shelf.group: @group needs a type with a @resource type",
		))
	})

	It("Should reject a resource type the enum does not hold", func(ctx SpecContext) {
		Expect(extract(ctx, `
			Shelf struct {
				key   uint32 {
					@key
				}
				group group.Key? {
					@group scope device
				}

				@resource type "shelf"
			}
		`)).Error().To(MatchError(
			"records.Shelf.group: @group has resource type shelf, which " +
				"resource.Type does not hold",
		))
	})

	It("Should reject a resource type marked twice", func(ctx SpecContext) {
		Expect(extract(ctx, rack("@group scope device")+`
			Payload struct {
				key   uint32 {
					@key
				}
				group group.Key? {
					@group scope device
				}

				@resource type "rack"
			}
		`)).Error().To(MatchError(
			"records.Payload.group: @group marks resource type rack a second time",
		))
	})

	It("Should reject a field that holds no record's key", func(ctx SpecContext) {
		Expect(extract(ctx, `
			Rack struct {
				key   uint32 {
					@key
				}
				group  uuid? {
					@group scope device
				}
				parent group.Key?

				@resource type "rack"
			}
		`)).Error().To(MatchError(
			"records.Rack.group: @group holds uuid, which is no record's key",
		))
	})

	It("Should reject a group record with no scope enum", func(ctx SpecContext) {
		loader.Add("schemas/group", `
			Key = uuid

			Group struct {
				key Key {
					@key
				}

				@resource type "group"
			}
		`)
		Expect(extract(ctx, rack("@group scope device"))).Error().To(MatchError(
			"records.Rack.group: @group needs group.Group to have a scope field",
		))
	})

	It("Should reject a group scope that is not an enum", func(ctx SpecContext) {
		loader.Add("schemas/group", `
			Key = uuid

			Group struct {
				key   Key {
					@key
				}
				scope string

				@resource type "group"
			}
		`)
		Expect(extract(ctx, rack("@group scope device"))).Error().To(MatchError(
			"records.Rack.group: @group needs the scope of group.Group to be an enum",
		))
	})

	It("Should reject a field holding another record's key", func(ctx SpecContext) {
		Expect(extract(ctx, rack("@group scope device")+`
			ShelfKey = uuid

			Shelf struct {
				key   ShelfKey {
					@key
				}

				@resource type "label"
			}

			Channel struct {
				key   uint32 {
					@key
				}
				group ShelfKey? {
					@group scope channel
				}

				@resource type "channel"
			}
		`)).Error().To(MatchError(
			"records.Channel.group: @group holds records.ShelfKey, not the key of " +
				"group.Group",
		))
	})

	It("Should skip a type in a version file", func(ctx SpecContext) {
		table, diag := Analyze(ctx, `
			import "schemas/group"
		`+rack("@group scope device"), "schemas/records/versions/rack/v1", loader)
		Expect(diag.Ok()).To(BeTrue(), diag.String)
		Expect(group.Extract(table)).To(BeNil())
	})
})
