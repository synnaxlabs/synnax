// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package panel_test

import (
	"encoding/json"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	. "github.com/synnaxlabs/synnax/pkg/service/imex/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/synnax/pkg/service/panel/versions"
	"github.com/synnaxlabs/x/spatial"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// encodedBody unpacks env's wire form into the generic map assertions inspect.
func encodedBody(env imex.Envelope) map[string]any {
	GinkgoHelper()
	raw := MustSucceed(json.Marshal(env))
	var m map[string]any
	Expect(json.Unmarshal(raw, &m)).To(Succeed())
	return m
}

// rootOf returns the panel tree from an encoded envelope body.
func rootOf(env imex.Envelope) map[string]any {
	GinkgoHelper()
	root, ok := encodedBody(env)["root"].(map[string]any)
	Expect(ok).To(BeTrue())
	return root
}

var _ = Describe("EncodeBundle", func() {
	It("Should stamp the panel envelope headers", func() {
		p := panel.Panel{Name: "Controls", Root: leafNode()}
		env := MustSucceed(panel.EncodeBundle(p, nil))
		Expect(env.Version).To(Equal(versions.Latest))
		Expect(env.Type).To(Equal("panel"))
		Expect(env.Name).To(Equal("Controls"))
		body := encodedBody(env)
		Expect(body).To(HaveKeyWithValue("type", "panel"))
		Expect(body).To(HaveKeyWithValue("name", "Controls"))
		Expect(body).To(HaveKeyWithValue(
			"version", BeNumerically("==", versions.Latest),
		))
	})

	It("Should rewrite resource references to bundle paths", func() {
		first, second := tab(uuid.New()), tab(uuid.New())
		refs := map[ontology.ID]string{
			mustResource(first):  "chamber_pressure.json",
			mustResource(second): "propulsion/pressurization.json",
		}
		p := panel.Panel{
			Name: "Controls",
			Root: splitNode(
				spatial.DirectionX, 0.5, leafNode(first), leafNode(second),
			),
		}
		root := rootOf(MustSucceed(panel.EncodeBundle(p, refs)))
		Expect(root).To(HaveKeyWithValue("variant", "split"))
		firstLeaf := root["first"].(map[string]any)
		Expect(firstLeaf["tabs"]).To(ConsistOf(SatisfyAll(
			HaveKeyWithValue("variant", "resource"),
			HaveKeyWithValue("resource", "chamber_pressure.json"),
			HaveKeyWithValue("key", first.Key().String()),
		)))
		lastLeaf := root["last"].(map[string]any)
		Expect(lastLeaf["tabs"]).To(ConsistOf(
			HaveKeyWithValue("resource", "propulsion/pressurization.json"),
		))
	})

	It("Should pass view tabs through unchanged", func() {
		view := viewTab(uuid.New(), "docs")
		p := panel.Panel{Name: "Controls", Root: leafNode(view)}
		root := rootOf(MustSucceed(panel.EncodeBundle(p, nil)))
		Expect(root["tabs"]).To(ConsistOf(SatisfyAll(
			HaveKeyWithValue("variant", "view"),
			HaveKeyWithValue("type", "docs"),
			HaveKeyWithValue("key", view.Key().String()),
		)))
	})

	It("Should strip resource tabs whose target is not a member", func() {
		member, stranger := tab(uuid.New()), tab(uuid.New())
		refs := map[ontology.ID]string{mustResource(member): "kept.json"}
		p := panel.Panel{Name: "Controls", Root: leafNode(member, stranger)}
		root := rootOf(MustSucceed(panel.EncodeBundle(p, refs)))
		Expect(root["tabs"]).To(ConsistOf(
			HaveKeyWithValue("resource", "kept.json"),
		))
	})

	It("Should collapse a leaf emptied by stripping", func() {
		member, stranger := tab(uuid.New()), tab(uuid.New())
		refs := map[ontology.ID]string{mustResource(member): "kept.json"}
		p := panel.Panel{
			Name: "Controls",
			Root: splitNode(
				spatial.DirectionY, 0.5, leafNode(member), leafNode(stranger),
			),
		}
		root := rootOf(MustSucceed(panel.EncodeBundle(p, refs)))
		Expect(root).To(HaveKeyWithValue("variant", "leaf"))
		Expect(root["tabs"]).To(ConsistOf(
			HaveKeyWithValue("resource", "kept.json"),
		))
	})

	It("Should collapse a split whose both sides were emptied", func() {
		first, second := tab(uuid.New()), tab(uuid.New())
		p := panel.Panel{
			Name: "Controls",
			Root: splitNode(
				spatial.DirectionX, 0.5, leafNode(first), leafNode(second),
			),
		}
		root := rootOf(MustSucceed(panel.EncodeBundle(p, nil)))
		Expect(root).To(HaveKeyWithValue("variant", "leaf"))
		Expect(root["tabs"]).To(BeEmpty())
	})

	It("Should collapse nested splits emptied by stripping", func() {
		member, s1, s2 := tab(uuid.New()), tab(uuid.New()), tab(uuid.New())
		refs := map[ontology.ID]string{mustResource(member): "kept.json"}
		p := panel.Panel{
			Name: "Controls",
			Root: splitNode(
				spatial.DirectionY,
				0.5,
				splitNode(spatial.DirectionX, 0.5, leafNode(s1), leafNode(s2)),
				leafNode(member),
			),
		}
		root := rootOf(MustSucceed(panel.EncodeBundle(p, refs)))
		Expect(root).To(HaveKeyWithValue("variant", "leaf"))
		Expect(root["tabs"]).To(ConsistOf(
			HaveKeyWithValue("resource", "kept.json"),
		))
	})

	It("Should reject a panel without a name", func() {
		Expect(panel.EncodeBundle(panel.Panel{Root: leafNode()}, nil)).Error().
			To(MatchError(ContainSubstring("name must be a non-empty string")))
	})
})

var _ = Describe("DecodeBundle", func() {
	It("Should resolve resource paths back to ontology IDs", func(ctx SpecContext) {
		first, second := tab(uuid.New()), tab(uuid.New())
		encodeRefs := map[ontology.ID]string{
			mustResource(first):  "chamber_pressure.json",
			mustResource(second): "propulsion/pressurization.json",
		}
		p := panel.Panel{
			Name: "Controls",
			Root: splitNode(
				spatial.DirectionX, 0.5, leafNode(first), leafNode(second),
			),
		}
		env := MustSucceed(panel.EncodeBundle(p, encodeRefs))
		minted := map[string]ontology.ID{
			"chamber_pressure.json": {
				Type: ontology.ResourceTypeLineplot, Key: uuid.NewString(),
			},
			"propulsion/pressurization.json": {
				Type: ontology.ResourceTypeSchematic, Key: uuid.NewString(),
			},
		}
		decoded := MustSucceed(panel.DecodeBundle(ctx, WireRoundTrip(env), minted))
		Expect(decoded.Name).To(Equal("Controls"))
		split, ok := decoded.Root.Variant.(panel.SplitNode)
		Expect(ok).To(BeTrue())
		firstLeaf, ok := split.First.Variant.(panel.LeafNode)
		Expect(ok).To(BeTrue())
		Expect(firstLeaf.Tabs).To(HaveLen(1))
		Expect(firstLeaf.Tabs[0].Key()).To(Equal(first.Key()))
		Expect(mustResource(firstLeaf.Tabs[0])).
			To(Equal(minted["chamber_pressure.json"]))
		lastLeaf, ok := split.Last.Variant.(panel.LeafNode)
		Expect(ok).To(BeTrue())
		Expect(mustResource(lastLeaf.Tabs[0])).
			To(Equal(minted["propulsion/pressurization.json"]))
	})

	It("Should pass view tabs through unchanged", func(ctx SpecContext) {
		view := viewTab(uuid.New(), "docs")
		p := panel.Panel{Name: "Controls", Root: leafNode(view)}
		env := MustSucceed(panel.EncodeBundle(p, nil))
		decoded := MustSucceed(panel.DecodeBundle(ctx, WireRoundTrip(env), nil))
		leaf, ok := decoded.Root.Variant.(panel.LeafNode)
		Expect(ok).To(BeTrue())
		Expect(leaf.Tabs).To(HaveLen(1))
		v, ok := leaf.Tabs[0].Variant.(panel.ViewTab)
		Expect(ok).To(BeTrue())
		Expect(v.Type).To(Equal("docs"))
	})

	It("Should reject a path the reference table does not hold", func(
		ctx SpecContext,
	) {
		t := tab(uuid.New())
		refs := map[ontology.ID]string{mustResource(t): "missing.json"}
		env := MustSucceed(panel.EncodeBundle(
			panel.Panel{Name: "Controls", Root: leafNode(t)}, refs,
		))
		Expect(panel.DecodeBundle(ctx, WireRoundTrip(env), nil)).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(`"missing.json"`)),
		))
	})

	It("Should reject a version newer than the panel schema", func(
		ctx SpecContext,
	) {
		env := imex.Envelope{
			Version: versions.Latest + 1, Type: "panel", Name: "Controls",
		}
		Expect(imex.Encode(&env, map[string]any{
			"root": map[string]any{"variant": "leaf", "tabs": []any{}},
		})).To(Succeed())
		Expect(panel.DecodeBundle(ctx, WireRoundTrip(env), nil)).Error().To(SatisfyAll(
			MatchError(ContainSubstring("panel version 1")),
			MatchError(ContainSubstring("newer than this Core supports")),
		))
	})

	It("Should reject a resource tab without a path", func(ctx SpecContext) {
		env := imex.Envelope{Version: 0, Type: "panel", Name: "Controls"}
		Expect(imex.Encode(&env, map[string]any{
			"root": map[string]any{
				"variant": "leaf",
				"tabs": []any{map[string]any{
					"key":      uuid.NewString(),
					"variant":  "resource",
					"resource": 42,
				}},
			},
		})).To(Succeed())
		Expect(panel.DecodeBundle(ctx, WireRoundTrip(env), nil)).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("holds no member path")),
		))
	})
})

var _ = Describe("ResourceRefs", func() {
	taskID := func() ontology.ID {
		return ontology.ID{Type: ontology.ResourceTypeTask, Key: uuid.NewString()}
	}
	taskTab := func(id ontology.ID) panel.Tab {
		return panel.Tab{Variant: panel.ResourceTab{
			TabBase:  panel.TabBase{Key: uuid.New()},
			Resource: id,
		}}
	}

	It("Should collect every resource the tree's resource tabs reference", func() {
		task, plot := taskID(), uuid.New()
		root := splitNode(
			spatial.DirectionX,
			0.5,
			leafNode(taskTab(task), tab(plot)),
			leafNode(viewTab(uuid.New(), "docs")),
		)
		Expect(panel.ResourceRefs(root)).To(ConsistOf(task, tabResource(plot)))
	})

	It("Should return a resource referenced by two tabs once", func() {
		id := taskID()
		root := splitNode(
			spatial.DirectionX,
			0.5,
			leafNode(taskTab(id)),
			leafNode(taskTab(id)),
		)
		Expect(panel.ResourceRefs(root)).To(ConsistOf(id))
	})

	It("Should return nothing for a tree of view tabs", func() {
		Expect(panel.ResourceRefs(leafNode(viewTab(uuid.New(), "docs")))).To(BeEmpty())
	})
})

// mustResource returns the ontology ID behind a resource tab fixture.
func mustResource(t panel.Tab) ontology.ID {
	GinkgoHelper()
	r, ok := t.Variant.(panel.ResourceTab)
	Expect(ok).To(BeTrue())
	return r.Resource
}
