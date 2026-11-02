// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package instance_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/plugin/pb/instance"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/oracle/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

const source = `
import "schemas/x/filter"

Key uuid

Alias = uuid

Detail struct {
	name string
}

Holder struct<T> {
	set filter.Set<T>
}

Thing struct {
	key       filter.Set<Key>
	raw       filter.Set<uuid>
	aliased   filter.Set<Alias>
	detail    filter.Set<Detail>
	none      filter.Set<nil>
	anything  filter.Set<any>
	labels    filter.Set<map<string, Detail>>
	direction filter.Ordered<filter.Direction>
	name      filter.String
}
`

var _ = Describe("Of", func() {
	var table *resolution.Table
	BeforeEach(func(ctx SpecContext) {
		loader := testutil.NewMockFileLoader()
		testutil.AddFilterSchema(loader)
		table = testutil.MustGenerateRequest(ctx, source, "thing", loader).Resolutions
	})
	fieldOf := func(typeName, fieldName string) resolution.Field {
		GinkgoHelper()
		typ := MustBeOk(table.Get(typeName))
		return MustBeOk(lo.Find(
			typ.Form.(resolution.StructForm).Fields,
			func(f resolution.Field) bool { return f.Name == fieldName },
		))
	}

	It("Should name a use over a local type after the type and the struct", func() {
		inst := MustBeOk(
			instance.Of(fieldOf("thing.Thing", "key").Type, "thing", table),
		)
		Expect(inst.Name).To(Equal("KeySet"))
		Expect(inst.Generic.QualifiedName).To(Equal("filter.Set"))
	})

	It("Should substitute the type arguments into the fields", func() {
		inst := MustBeOk(
			instance.Of(fieldOf("thing.Thing", "key").Type, "thing", table),
		)
		Expect(inst.Fields).To(HaveLen(2))
		Expect(inst.Fields[0].Name).To(Equal("eq"))
		Expect(inst.Fields[0].Type.Name).To(Equal("thing.Key"))
		Expect(inst.Fields[0].Optional).To(BeTrue())
		Expect(inst.Fields[1].Type.Name).To(Equal("Array"))
		Expect(inst.Fields[1].Type.TypeArgs[0].Name).To(Equal("thing.Key"))
	})

	It("Should name a use over a primitive after the primitive", func() {
		inst := MustBeOk(
			instance.Of(fieldOf("thing.Thing", "raw").Type, "thing", table),
		)
		Expect(inst.Name).To(Equal("UuidSet"))
	})

	It("Should name a use through an alias after the alias's target", func() {
		inst := MustBeOk(
			instance.Of(fieldOf("thing.Thing", "aliased").Type, "thing", table),
		)
		Expect(inst.Name).To(Equal("UuidSet"))
		Expect(inst.Fields[0].Type.Name).To(Equal("uuid"))
	})

	It("Should qualify a type argument from another namespace", func() {
		inst := MustBeOk(instance.Of(
			fieldOf("thing.Thing", "direction").Type,
			"thing",
			table,
		))
		Expect(inst.Name).To(Equal("FilterDirectionOrdered"))
	})

	It("Should name a use over a map after the map's type arguments", func() {
		inst := MustBeOk(
			instance.Of(fieldOf("thing.Thing", "labels").Type, "thing", table),
		)
		Expect(inst.Name).To(Equal("StringDetailMapSet"))
	})

	It("Should leave a use over a struct to the generic struct's message", func() {
		_, ok := instance.Of(fieldOf("thing.Thing", "detail").Type, "thing", table)
		Expect(ok).To(BeFalse())
	})

	It("Should leave a use over nil to the generic struct's message", func() {
		_, ok := instance.Of(fieldOf("thing.Thing", "none").Type, "thing", table)
		Expect(ok).To(BeFalse())
	})

	It("Should leave a use over any to the generic struct's message", func() {
		_, ok := instance.Of(fieldOf("thing.Thing", "anything").Type, "thing", table)
		Expect(ok).To(BeFalse())
	})

	It(
		"Should leave a use over a type parameter to the generic struct's message",
		func() {
			_, ok := instance.Of(fieldOf("thing.Holder", "set").Type, "thing", table)
			Expect(ok).To(BeFalse())
		},
	)

	It("Should leave a struct that is not generic", func() {
		_, ok := instance.Of(fieldOf("thing.Thing", "name").Type, "thing", table)
		Expect(ok).To(BeFalse())
	})
})
