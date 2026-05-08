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
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("ImEx", func() {
	Describe("Envelope", func() {
		Describe("UnmarshalJSON", func() {
			It("Should extract promoted fields and put the rest into Data", func() {
				src := []byte(`{"version":54,"type":"log","name":"n","foo":1}`)
				var env imex.Envelope
				Expect(json.Unmarshal(src, &env)).To(Succeed())
				Expect(env.Version).To(Equal(imex.Version(54)))
				Expect(env.Type).To(Equal("log"))
				Expect(env.Name).To(Equal("n"))
				Expect(env.Data).To(HaveKeyWithValue("foo", BeNumerically("==", 1)))
				Expect(env.Data).NotTo(HaveKey("version"))
				Expect(env.Data).NotTo(HaveKey("type"))
				Expect(env.Data).NotTo(HaveKey("name"))
			})

			It("Should accept a legacy N.0.0 version string", func() {
				var env imex.Envelope
				Expect(json.Unmarshal(
					[]byte(`{"version":"4.0.0","type":"log"}`), &env,
				)).To(Succeed())
				Expect(env.Version).To(Equal(imex.Version(4)))
			})

			It("Should reject a version string with non-zero minor or patch", func() {
				var env imex.Envelope
				Expect(json.Unmarshal([]byte(`{"version":"1.2.3"}`), &env)).To(
					MatchError(ContainSubstring("only N.0.0")),
				)
			})

			It("Should reject a version string with a non-numeric major component", func() {
				var env imex.Envelope
				Expect(json.Unmarshal([]byte(`{"version":"sdfsd.0.0"}`), &env)).To(
					MatchError(ContainSubstring("invalid version major")),
				)
			})

			It("Should reject a version string with more than three dot-separated parts", func() {
				var env imex.Envelope
				Expect(json.Unmarshal([]byte(`{"version":"1.0.0.0"}`), &env)).To(
					MatchError(ContainSubstring("expected N.0.0")),
				)
			})

			It("Should error when version is neither number nor string", func() {
				var env imex.Envelope
				Expect(json.Unmarshal([]byte(`{"version":[1,2,3]}`), &env)).To(
					MatchError(ContainSubstring("must be a number or semver string")),
				)
			})

			It("Should error when type is not a string", func() {
				var env imex.Envelope
				Expect(json.Unmarshal([]byte(`{"type":5}`), &env)).To(
					MatchError(ContainSubstring("string")),
				)
			})

			It("Should error when name is not a string", func() {
				var env imex.Envelope
				Expect(json.Unmarshal([]byte(`{"name":[]}`), &env)).To(
					MatchError(ContainSubstring("string")),
				)
			})

			It("Should leave Data nil when only promoted fields are present", func() {
				var env imex.Envelope
				Expect(json.Unmarshal(
					[]byte(`{"version":1,"type":"log","name":"n"}`), &env,
				)).To(Succeed())
				Expect(env.Version).To(Equal(imex.Version(1)))
				Expect(env.Type).To(Equal("log"))
				Expect(env.Name).To(Equal("n"))
				Expect(env.Data).To(BeNil())
			})

			It("Should yield a zero envelope on null", func() {
				var env imex.Envelope
				Expect(json.Unmarshal([]byte(`null`), &env)).To(Succeed())
				Expect(env).To(Equal(imex.Envelope{}))
			})

			It("Should error when the input is a bare JSON number", func() {
				var env imex.Envelope
				Expect(json.Unmarshal([]byte(`34`), &env)).To(
					MatchError(ContainSubstring("cannot unmarshal number")),
				)
			})
		})

		Describe("MarshalJSON", func() {
			It("Should merge the promoted fields onto Data, with promoted winning on conflict", func() {
				env := imex.Envelope{
					Version: 54,
					Type:    "log",
					Name:    "n",
					Data: map[string]any{
						"version":  999,
						"type":     "ignored",
						"channels": []any{1.0, 2.0},
					},
				}
				b := MustSucceed(json.Marshal(env))
				var round map[string]any
				Expect(json.Unmarshal(b, &round)).To(Succeed())
				Expect(round["version"]).To(BeNumerically("==", 54))
				Expect(round["type"]).To(Equal("log"))
				Expect(round["name"]).To(Equal("n"))
				Expect(round["channels"]).To(HaveLen(2))
				Expect(round).NotTo(HaveKey("key"))
			})
		})

		Describe("Round-trip", func() {
			It("Should preserve nested content across Marshal/Unmarshal", func() {
				src := imex.Envelope{
					Version: 7,
					Type:    "log",
					Name:    "n",
					Data: map[string]any{
						"channels":       []any{float64(1), float64(2), float64(3)},
						"remote_created": true,
					},
				}
				b := MustSucceed(json.Marshal(src))
				var dst imex.Envelope
				Expect(json.Unmarshal(b, &dst)).To(Succeed())
				Expect(src).To(Equal(dst))
			})
		})
	})
})
