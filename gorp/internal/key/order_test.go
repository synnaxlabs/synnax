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

	It("Should order signed integers by value", func() {
		order := key.Order[int32]()
		Expect(order(-5, 3)).To(Equal(-1))
		Expect(order(3, -5)).To(Equal(1))
		Expect(order(7, 7)).To(Equal(0))
	})

	It("Should order unsigned integers by value", func() {
		order := key.Order[uint32]()
		Expect(order(256, 1)).To(Equal(1))
		Expect(order(1, 256)).To(Equal(-1))
	})

	It("Should order named strings byte by byte", func() {
		order := key.Order[name]()
		Expect(order("a", "b")).To(Equal(-1))
		Expect(order("B", "a")).To(Equal(-1))
		Expect(order("ab", "ab")).To(Equal(0))
	})

	It("Should order false before true", func() {
		order := key.Order[bool]()
		Expect(order(false, true)).To(Equal(-1))
		Expect(order(true, false)).To(Equal(1))
		Expect(order(true, true)).To(Equal(0))
	})

	It("Should order floats by value", func() {
		order := key.Order[float64]()
		Expect(order(-1.5, 0.25)).To(Equal(-1))
	})
})
