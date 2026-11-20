// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rename_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/resource/rename"
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Rename", func() {
	var db *gorp.DB
	BeforeEach(func() { db = DeferClose(gorp.Wrap(memkv.New())) })
	channel := func(key string) resource.ID {
		return resource.ID{Type: resource.TypeChannel, Key: key}
	}

	It("Should return what Stage records, in resource order", func(ctx SpecContext) {
		b := rename.Rename{Resource: channel("2"), From: "a.b", To: "a_b"}
		a := rename.Rename{Resource: channel("1"), From: "c.d", To: "c_d"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return rename.Stage(ctx, tx, b, a)
		})).To(Succeed())
		Expect(rename.Staged(db)).To(Equal([]rename.Rename{a, b}))
	})

	It("Should keep the last rename of one resource", func(ctx SpecContext) {
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return rename.Stage(
				ctx,
				tx,
				rename.Rename{Resource: channel("1"), From: "a", To: "b"},
				rename.Rename{Resource: channel("1"), From: "a", To: "c"},
			)
		})).To(Succeed())
		Expect(rename.Staged(db)).To(Equal([]rename.Rename{
			{Resource: channel("1"), From: "a", To: "c"},
		}))
	})

	It("Should clear every staged rename", func(ctx SpecContext) {
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return rename.Stage(
				ctx,
				tx,
				rename.Rename{Resource: channel("1"), From: "a", To: "b"},
			)
		})).To(Succeed())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return rename.Clear(ctx, tx)
		})).To(Succeed())
		Expect(rename.Staged(db)).To(BeEmpty())
	})

	DescribeTable("Free",
		func(base string, taken []string, expected string) {
			Expect(rename.Free(base, set.New(taken...).Contains)).To(Equal(expected))
		},
		Entry("a free base", "pt", []string{"pt_2"}, "pt"),
		Entry("a taken base", "pt", []string{"pt"}, "pt_2"),
		Entry("a taken first suffix", "pt", []string{"pt", "pt_2"}, "pt_3"),
	)
})
