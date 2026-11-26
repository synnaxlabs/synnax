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
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Renames", func() {
	var (
		pressure = v0.ID{Type: v0.TypeChannel, Key: "1048577"}
		stand    = v0.ID{
			Type: v0.TypeGroup,
			Key:  "5b7c1f2a-9e44-4c1b-8a55-2f6b0e9d4c71",
		}
	)

	It("Should keep the first name of a resource that the upgrade renames twice",
		func(ctx SpecContext) {
			kv := DeferClose(memkv.New())
			Expect(v0.StageRename(ctx, kv, pressure, "pt.01")).To(Succeed())
			Expect(v0.StageRename(ctx, kv, stand, "Stand A")).To(Succeed())
			Expect(v0.StageRename(ctx, kv, pressure, "pt_01")).To(Succeed())
			Expect(MustSucceed(v0.StagedRenames(kv))).To(Equal([]v0.Rename{
				{ID: pressure, From: "pt.01"},
				{ID: stand, From: "Stand A"},
			}))
		})

	It("Should delete every staged rename", func(ctx SpecContext) {
		kv := DeferClose(memkv.New())
		Expect(v0.StageRename(ctx, kv, pressure, "pt.01")).To(Succeed())
		Expect(v0.StageRename(ctx, kv, stand, "Stand A")).To(Succeed())
		Expect(v0.ClearRenames(ctx, kv)).To(Succeed())
		Expect(MustSucceed(v0.StagedRenames(kv))).To(BeEmpty())
	})

	Describe("FreeName", func() {
		It("Should return the base when it is free", func() {
			Expect(v0.FreeName("pt", set.New[string]().Contains)).To(Equal("pt"))
		})

		It("Should return the first free suffix", func() {
			taken := set.New("pt", "pt_2", "pt_4")
			Expect(v0.FreeName("pt", taken.Contains)).To(Equal("pt_3"))
		})
	})
})
