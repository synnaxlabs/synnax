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
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	v0 "github.com/synnaxlabs/synnax/pkg/service/log/migrations/v0"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Data", func() {
	Describe("Parse", func() {
		It("Should parse a valid v0 payload", func() {
			data := map[string]any{
				"channels":      []channel.Key{1, 2, 3},
				"remoteCreated": true,
			}
			d := MustSucceed(v0.Parse(data))
			Expect(d.Channels).To(Equal([]channel.Key{1, 2, 3}))
			Expect(d.RemoteCreated).To(BeTrue())
		})

		It("Should accept an empty channels array", func() {
			d := MustSucceed(v0.Parse(map[string]any{"channels": []any{}}))
			Expect(d.Channels).To(HaveLen(0))
		})

		It("Should default a missing remoteCreated to false", func() {
			d := MustSucceed(v0.Parse(map[string]any{"channels": []any{1}}))
			Expect(d.RemoteCreated).To(BeFalse())
		})

		It("Should coerce whole-number channel keys", func() {
			d := MustSucceed(v0.Parse(map[string]any{"channels": []any{1.0, 2.0}}))
			Expect(d.Channels).To(Equal([]channel.Key{1, 2}))
		})

		It("Should coerce json.Number channel keys", func() {
			data := map[string]any{
				"channels": []any{json.Number("1"), json.Number("2")},
			}
			d := MustSucceed(v0.Parse(data))
			Expect(d.Channels).To(Equal([]channel.Key{1, 2}))
		})

		It("Should reject a non-array channels field", func() {
			Expect(v0.Parse(map[string]any{"channels": "not-an-array"})).Error().
				To(MatchError(ContainSubstring("channels")))
		})

		It("Should reject a fractional channel key", func() {
			Expect(v0.Parse(map[string]any{"channels": []any{1.5}})).Error().
				To(MatchError(ContainSubstring("channels")))
		})
	})

	Describe("Validate", func() {
		It("Should accept a payload whose channel keys are all non-zero", func() {
			d := v0.Data{Channels: []channel.Key{1, 2, 3}}
			Expect(d.Validate()).To(Succeed())
		})

		It("Should reject a zero channel key", func() {
			d := v0.Data{Channels: []channel.Key{1, 0}}
			Expect(d.Validate()).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("channels[1]")),
			))
		})
		It("Should accept an empty channels array", func() {
			d := v0.Data{Channels: []channel.Key{}}
			Expect(d.Validate()).To(Succeed())
		})
	})
})
