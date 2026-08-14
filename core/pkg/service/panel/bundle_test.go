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
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/x/spatial"
	. "github.com/synnaxlabs/x/testutil"
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
		Expect(env.Version).To(Equal(imex.Version(0)))
		Expect(env.Type).To(Equal("panel"))
		Expect(env.Name).To(Equal("Controls"))
		body := encodedBody(env)
		Expect(body).To(HaveKeyWithValue("type", "panel"))
		Expect(body).To(HaveKeyWithValue("name", "Controls"))
		Expect(body).To(HaveKeyWithValue("version", BeNumerically("==", 0)))
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

// mustResource returns the ontology ID behind a resource tab fixture.
func mustResource(t panel.Tab) ontology.ID {
	GinkgoHelper()
	r, ok := t.Variant.(panel.TabResource)
	Expect(ok).To(BeTrue())
	return r.Resource
}
