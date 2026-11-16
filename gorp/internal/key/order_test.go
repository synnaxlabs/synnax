// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package key_test

import (
	"bytes"
	"encoding/json"
	"os"
	"slices"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp/internal/key"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/types"
)

var _ = Describe("Order", func() {
	It("Should order UUIDs as Gorp stores them", func() {
		keys := make([]uuid.UUID, 64)
		for i := range keys {
			keys[i] = uuid.New()
		}
		byOrder := slices.Clone(keys)
		slices.SortFunc(byOrder, key.Order[uuid.UUID]())
		codec := key.NewCodec[uuid.UUID](nil)
		byStorage := slices.Clone(keys)
		slices.SortFunc(byStorage, func(a, b uuid.UUID) int {
			return bytes.Compare(bytes.Clone(codec.Encode(a)), codec.Encode(b))
		})
		Expect(byOrder).To(Equal(byStorage))
	})

	It("Should order UUIDs as the shared vectors do", func() {
		var vectors struct {
			Ordered []uuid.UUID `json:"ordered"`
		}
		Expect(json.Unmarshal(
			MustSucceed(os.ReadFile("../../../x/testdata/compare/uuid.json")),
			&vectors,
		)).To(Succeed())
		shuffled := slices.Clone(vectors.Ordered)
		slices.Reverse(shuffled)
		slices.SortFunc(shuffled, key.Order[uuid.UUID]())
		Expect(shuffled).To(Equal(vectors.Ordered))
	})

	DescribeTable(
		"Should order keys as Gorp stores them",
		func(order, encoded func() int) {
			Expect(order()).To(Equal(encoded()))
		},
		Entry("signed integers", orderOf[int32](-5, 3), encodedOrderOf[int32](-5, 3)),
		Entry("floats", orderOf(-1.5, 0.25), encodedOrderOf(-1.5, 0.25)),
		Entry(
			"unsigned integers",
			orderOf[uint32](256, 1),
			encodedOrderOf[uint32](256, 1),
		),
		Entry("booleans", orderOf(false, true), encodedOrderOf(false, true)),
		Entry("equal booleans", orderOf(true, true), encodedOrderOf(true, true)),
		Entry("strings", orderOf("B", "a"), encodedOrderOf("B", "a")),
		Entry(
			"named strings",
			orderOf[name]("ab", "b"),
			encodedOrderOf[name]("ab", "b"),
		),
	)
})

func orderOf[K primitive](a, b K) func() int {
	return func() int { return key.Order[K]()(a, b) }
}

func encodedOrderOf[K primitive](a, b K) func() int {
	return func() int {
		codec := key.NewCodec[K](nil)
		return bytes.Compare(bytes.Clone(codec.Encode(a)), codec.Encode(b))
	}
}

type primitive interface {
	types.Primitive
	comparable
}
