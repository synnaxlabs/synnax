// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package imex_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
)

type sampleDest struct {
	Name     string   `json:"name"`
	Channels []int    `json:"channels"`
	Tags     []string `json:"tags"`
}

var _ = Describe("Decode", func() {
	It("Should decode a valid payload into the destination", func() {
		var d sampleDest
		raw := json.RawMessage(`{"name":"n","channels":[1,2],"tags":["a"]}`)
		Expect(imex.Decode(raw, &d)).To(Succeed())
		Expect(d.Name).To(Equal("n"))
		Expect(d.Channels).To(Equal([]int{1, 2}))
		Expect(d.Tags).To(Equal([]string{"a"}))
	})

	It("Should surface the field name and types on a type mismatch", func() {
		var d sampleDest
		raw := json.RawMessage(`{"channels":"not an array"}`)
		err := imex.Decode(raw, &d)
		Expect(err).To(SatisfyAll(
			MatchError(ContainSubstring("channels")),
			MatchError(ContainSubstring("expected")),
			MatchError(ContainSubstring("string")),
		))
	})

	It("Should include an offset on a syntax error", func() {
		var d sampleDest
		raw := json.RawMessage(`{"name":"n",`)
		err := imex.Decode(raw, &d)
		Expect(err).To(MatchError(ContainSubstring("malformed JSON")))
	})

	It("Should give a friendly message for a truncated payload", func() {
		var d sampleDest
		raw := json.RawMessage(`{"channels":[1,2`)
		err := imex.Decode(raw, &d)
		Expect(err).To(MatchError(SatisfyAny(
			ContainSubstring("truncated JSON"),
			ContainSubstring("malformed JSON"),
		)))
	})

	It("Should pass through unknown fields without error", func() {
		var d sampleDest
		raw := json.RawMessage(`{"name":"n","channels":[],"unknown":"ignored"}`)
		Expect(imex.Decode(raw, &d)).To(Succeed())
		Expect(d.Name).To(Equal("n"))
	})
})
