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
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/synnax/pkg/service/log/migrations/v0"
)

var _ = Describe("Data", func() {
	Describe("Schema", func() {
		It("Should parse a valid v0 payload", func() {
			data := map[string]any{
				"channels":       []any{1, 2, 3},
				"remote_created": true,
			}
			var d v0.Data
			Expect(v0.Schema.Parse(data, &d)).To(Succeed())
			Expect(d.Channels).To(Equal([]int{1, 2, 3}))
			Expect(d.RemoteCreated).To(BeTrue())
		})

		It("Should accept an empty channels array", func() {
			data := map[string]any{"channels": []any{}}
			var d v0.Data
			Expect(v0.Schema.Parse(data, &d)).To(Succeed())
			Expect(d.Channels).To(HaveLen(0))
		})

		It("Should accept a missing remote_created (Optional)", func() {
			data := map[string]any{"channels": []any{1}}
			var d v0.Data
			Expect(v0.Schema.Parse(data, &d)).To(Succeed())
			Expect(d.RemoteCreated).To(BeFalse())
		})

		It("Should coerce float channel keys to int", func() {
			data := map[string]any{"channels": []any{1.0, 2.0}}
			var d v0.Data
			Expect(v0.Schema.Parse(data, &d)).To(Succeed())
			Expect(d.Channels).To(Equal([]int{1, 2}))
		})

		It("Should coerce json.Number channel keys to int", func() {
			data := map[string]any{
				"channels": []any{json.Number("1"), json.Number("2")},
			}
			var d v0.Data
			Expect(v0.Schema.Parse(data, &d)).To(Succeed())
			Expect(d.Channels).To(Equal([]int{1, 2}))
		})

		It("Should reject a missing channels field", func() {
			data := map[string]any{"remote_created": true}
			var d v0.Data
			Expect(v0.Schema.Parse(data, &d)).To(MatchError(ContainSubstring("channels")))
		})

		It("Should reject a non-array channels field", func() {
			data := map[string]any{"channels": "not-an-array"}
			var d v0.Data
			Expect(v0.Schema.Parse(data, &d)).To(MatchError(ContainSubstring("channels")))
		})

		It("Should reject a non-bool remote_created field", func() {
			data := map[string]any{
				"channels":       []any{1},
				"remote_created": "yes",
			}
			var d v0.Data
			Expect(v0.Schema.Parse(data, &d)).To(MatchError(ContainSubstring("remote_created")))
		})

		It("Should reject a fractional channel key", func() {
			data := map[string]any{"channels": []any{1.5}}
			var d v0.Data
			Expect(v0.Schema.Parse(data, &d)).To(MatchError(ContainSubstring("channels")))
		})
	})
})
