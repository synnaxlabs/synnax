// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/synnax/pkg/service/resource/versions/v0"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Text sites", func() {
	var (
		calc    = v0.ID{Type: v0.TypeChannel, Key: "1048578"}
		earlier = v0.TextSite{
			Holder:        calc,
			Name:          "pt",
			NewName:       "pt_2",
			Line:          1,
			Column:        7,
			BesideCapture: true,
		}
		later = v0.TextSite{
			Holder:   calc,
			Function: "f",
			Name:     "pt",
			NewName:  "pt_2",
			Line:     2,
			Column:   0,
			Captured: true,
		}
	)

	It("Should return each staged site in key order", func(ctx SpecContext) {
		kv := DeferClose(memkv.New())
		Expect(v0.StageTextSite(ctx, kv, later)).To(Succeed())
		Expect(v0.StageTextSite(ctx, kv, earlier)).To(Succeed())
		Expect(v0.StagedTextSites(kv)).To(Equal([]v0.TextSite{earlier, later}))
	})

	It("Should read no staged site as a staged rename", func(ctx SpecContext) {
		kv := DeferClose(memkv.New())
		Expect(v0.StageTextSite(ctx, kv, earlier)).To(Succeed())
		Expect(v0.StagedRenames(kv)).To(BeEmpty())
	})

	It("Should delete each staged site", func(ctx SpecContext) {
		kv := DeferClose(memkv.New())
		Expect(v0.StageTextSite(ctx, kv, earlier)).To(Succeed())
		Expect(v0.StageTextSite(ctx, kv, later)).To(Succeed())
		Expect(v0.ClearTextSites(ctx, kv)).To(Succeed())
		Expect(v0.StagedTextSites(kv)).To(BeEmpty())
	})

	It("Should fail on a staged site whose value does not decode", func(
		ctx SpecContext,
	) {
		kv := DeferClose(memkv.New())
		Expect(kv.Set(ctx, []byte("sy_v63_staged_text_site/x"), []byte("{"))).
			To(Succeed())
		Expect(v0.StagedTextSites(kv)).Error().
			To(MatchError(ContainSubstring(`staged text site "x"`)))
	})
})
