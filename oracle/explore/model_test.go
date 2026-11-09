// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package explore_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/explore"
	"github.com/synnaxlabs/oracle/testutil"
	"github.com/synnaxlabs/oracle/versions"
	. "github.com/synnaxlabs/x/testutil"
)

const schema = `
import "resource"

@go output "core/pkg/service/widget"

Key = uuid {
    @doc value "is a unique identifier for a rack."
}

Base struct {
    key Key {
        @key
    }
}

APIRack struct extends Base {
    @resource type "rack"
}

Rack struct extends Base {
    name string {
        @doc value """
            is a human-readable
            name for the rack.
        """
        @validate required
    }

    @doc value "is a container for tasks."
    @go marshal
    @resource type "rack"
}

ModuleKey = uuid

ModuleBase struct {
    key ModuleKey {
        @key
    }
}

Module union on kind extends ModuleBase {
    text { source string }

    @resource type "module"
}

Mode enum {
    fast = "fast" {
        @doc value "runs at full speed."
    }
    slow = "slow"
}

Task struct {
    rack   Key? {
        @ref kind place
        @ref on_delete cascade
    }
    module ModuleKey? {
        @ref kind use
        @ref on_delete restrict
    }
    owner  resource.ID? {
        @ref kind place
        @ref on_delete detach
        @ref types rack module
    }
    mode   Mode = fast
}

Shape union on kind {
    circle { radius float64 }
    square Task
}
`

const resourceSchema = `
@go output "core/pkg/resource"

ID struct {
    key string
}
`

func build(ctx context.Context) explore.Model {
	GinkgoHelper()
	loader := testutil.NewMockFileLoader().
		Add("resource", resourceSchema)
	table, diag := testutil.Analyze(ctx, schema, "widget", loader)
	Expect(diag.Ok()).To(BeTrue(), diag.String())
	return explore.Build(table, map[string]versions.Chain{
		"widget": {Resource: "widget", Numbers: []int{0, 2}},
	})
}

func typ(m explore.Model, id string) explore.Type {
	GinkgoHelper()
	return MustBeOk(lo.Find(m.Types, func(t explore.Type) bool { return t.ID == id }))
}

func field(t explore.Type, name string) explore.Field {
	GinkgoHelper()
	return MustBeOk(
		lo.Find(t.Fields, func(f explore.Field) bool { return f.Name == name }),
	)
}

var _ = Describe("Build", func() {
	It("Should list declared types sorted by qualified name", func(ctx SpecContext) {
		ids := lo.Map(
			build(ctx).Types,
			func(t explore.Type, _ int) string { return t.ID },
		)
		Expect(ids).To(Equal([]string{
			"resource.ID",
			"widget.APIRack",
			"widget.Base",
			"widget.Key",
			"widget.Mode",
			"widget.Module",
			"widget.ModuleBase",
			"widget.ModuleKey",
			"widget.Rack",
			"widget.Shape",
			"widget.Task",
		}))
	})

	It("Should describe each namespace's file and outputs", func(ctx SpecContext) {
		Expect(build(ctx).Namespaces).To(Equal([]explore.Namespace{
			{
				Name:    "resource",
				File:    "resource.oracle",
				Outputs: map[string][]string{"go": {"core/pkg/resource"}},
			},
			{
				Name:     "widget",
				File:     "widget.oracle",
				Outputs:  map[string][]string{"go": {"core/pkg/service/widget"}},
				Versions: []int{0, 2},
			},
		}))
	})

	It("Should join each doc paragraph onto one line", func(ctx SpecContext) {
		rack := typ(build(ctx), "widget.Rack")
		Expect(
			field(rack, "name").Doc,
		).To(Equal("is a human-readable name for the rack."))
		Expect(rack.Doc).To(Equal("is a container for tasks."))
	})

	It("Should mark resources, persisted types, and keys", func(ctx SpecContext) {
		rack := typ(build(ctx), "widget.Rack")
		Expect(rack.Resource).To(Equal("rack"))
		Expect(rack.Persisted).To(BeTrue())
		Expect(rack.Key).To(Equal("widget.Key"))
		Expect(rack.Extends).To(Equal([]explore.Ref{{Name: "widget.Base"}}))
		Expect(rack.Annotations).To(ConsistOf("@go marshal", `@resource type "rack"`))
	})

	It("Should give a key the persisted resource it identifies", func(ctx SpecContext) {
		key := typ(build(ctx), "widget.Key")
		Expect(key.Owner).To(Equal("widget.Rack"))
		Expect(key.Kind).To(Equal("alias"))
	})

	It("Should give a union resource the key of its base", func(ctx SpecContext) {
		m := build(ctx)
		Expect(typ(m, "widget.Module").Key).To(Equal("widget.ModuleKey"))
		Expect(typ(m, "widget.ModuleKey").Owner).To(Equal("widget.Module"))
	})

	It("Should resolve a key reference to its resource", func(ctx SpecContext) {
		task := typ(build(ctx), "widget.Task")
		Expect(field(task, "rack")).To(Equal(explore.Field{
			Name:     "rack",
			Type:     &explore.Ref{Name: "widget.Key"},
			Optional: true,
			Link: &explore.Link{
				Targets:  []string{"widget.Rack"},
				Kind:     "place",
				OnDelete: "cascade",
			},
			Annotations: []string{"@ref kind place", "@ref on_delete cascade"},
		}))
		Expect(field(task, "module").Link.Targets).To(Equal([]string{"widget.Module"}))
		Expect(field(task, "mode").Default).To(Equal("fast"))
	})

	It(
		"Should resolve a resource ID reference to each named type",
		func(ctx SpecContext) {
			Expect(
				field(typ(build(ctx), "widget.Task"), "owner").Link,
			).To(Equal(&explore.Link{
				Targets:  []string{"widget.Rack", "widget.Module"},
				Kind:     "place",
				OnDelete: "detach",
			}))
		},
	)

	It("Should list enum members with their docs", func(ctx SpecContext) {
		Expect(typ(build(ctx), "widget.Mode").Values).To(Equal([]explore.Value{
			{Name: "fast", Value: "fast", Doc: "runs at full speed."},
			{Name: "slow", Value: "slow"},
		}))
	})

	It("Should flatten the fields of inline union variants", func(ctx SpecContext) {
		shape := typ(build(ctx), "widget.Shape")
		Expect(shape.Discriminator).To(Equal("kind"))
		Expect(shape.Variants).To(HaveLen(2))
		Expect(shape.Variants[0].Fields).To(Equal([]explore.Field{
			{Name: "radius", Type: &explore.Ref{Name: "float64"}},
		}))
		Expect(shape.Variants[1].Type).To(Equal(explore.Ref{Name: "widget.Task"}))
		Expect(shape.Variants[1].Fields).To(BeEmpty())
	})
})
