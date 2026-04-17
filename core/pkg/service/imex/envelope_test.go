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
	"bytes"
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Envelope", func() {
	Describe("UnmarshalJSON", func() {
		It("Should extract promoted fields and copy the raw body into Data", func() {
			src := []byte(`{"version":54,"type":"log","key":"k","name":"n","foo":1}`)
			var env imex.Envelope
			Expect(json.Unmarshal(src, &env)).To(Succeed())
			Expect(env.Version).To(Equal(54))
			Expect(env.Type).To(Equal("log"))
			Expect(env.Key).To(Equal("k"))
			Expect(env.Name).To(Equal("n"))

			var canonicalData bytes.Buffer
			Expect(json.Compact(&canonicalData, env.Data)).To(Succeed())
			var canonicalSrc bytes.Buffer
			Expect(json.Compact(&canonicalSrc, src)).To(Succeed())
			Expect(canonicalData.String()).To(Equal(canonicalSrc.String()))
		})

		It("Should accept a numeric version", func() {
			var env imex.Envelope
			Expect(json.Unmarshal(
				[]byte(`{"version":7,"type":"log","channels":[]}`), &env,
			)).To(Succeed())
			Expect(env.Version).To(Equal(7))
		})

		It("Should translate a semver version via legacyToNumeric", func() {
			var env imex.Envelope
			Expect(json.Unmarshal(
				[]byte(`{"version":"1.0.0","type":"log","channels":[]}`), &env,
			)).To(Succeed())
			// 1*5 + 0*2 + 0 = 5
			Expect(env.Version).To(Equal(5))
		})

		It("Should not share storage with the caller's buffer", func() {
			src := []byte(`{"version":1,"type":"log","channels":[]}`)
			var env imex.Envelope
			Expect(json.Unmarshal(src, &env)).To(Succeed())
			original := append(json.RawMessage(nil), env.Data...)
			for i := range src {
				src[i] = 'x'
			}
			Expect([]byte(env.Data)).To(Equal([]byte(original)))
		})
	})

	Describe("MarshalJSON", func() {
		It("Should splice the promoted fields on top of Data", func() {
			env := imex.Envelope{
				Version: 54,
				Type:    "log",
				Key:     "k",
				Name:    "n",
				Data:    json.RawMessage(`{"version":999,"type":"ignored","channels":[1,2]}`),
			}
			b := MustSucceed(json.Marshal(env))
			var round map[string]any
			Expect(json.Unmarshal(b, &round)).To(Succeed())
			Expect(round["version"]).To(BeNumerically("==", 54))
			Expect(round["type"]).To(Equal("log"))
			Expect(round["key"]).To(Equal("k"))
			Expect(round["name"]).To(Equal("n"))
			Expect(round["channels"]).To(HaveLen(2))
		})

		It("Should emit promoted fields even when Data is empty", func() {
			env := imex.Envelope{Version: 3, Type: "log"}
			b := MustSucceed(json.Marshal(env))
			var round map[string]any
			Expect(json.Unmarshal(b, &round)).To(Succeed())
			Expect(round["version"]).To(BeNumerically("==", 3))
			Expect(round["type"]).To(Equal("log"))
			Expect(round).NotTo(HaveKey("key"))
			Expect(round).NotTo(HaveKey("name"))
		})

		It("Should reject non-object Data payloads", func() {
			env := imex.Envelope{Version: 1, Data: json.RawMessage(`[1,2,3]`)}
			_, err := json.Marshal(env)
			Expect(err).To(MatchError(ContainSubstring("envelope data must be a JSON object")))
		})
	})

	Describe("Round-trip", func() {
		It("Should preserve nested content across Marshal/Unmarshal", func() {
			src := imex.Envelope{
				Version: 7,
				Type:    "log",
				Key:     "k",
				Name:    "n",
				Data:    json.RawMessage(`{"channels":[1,2,3],"remote_created":true}`),
			}
			b := MustSucceed(json.Marshal(src))
			var dst imex.Envelope
			Expect(json.Unmarshal(b, &dst)).To(Succeed())
			Expect(dst.Version).To(Equal(7))
			Expect(dst.Type).To(Equal("log"))
			Expect(dst.Key).To(Equal("k"))
			Expect(dst.Name).To(Equal("n"))

			var round map[string]any
			Expect(json.Unmarshal(dst.Data, &round)).To(Succeed())
			Expect(round["channels"]).To(HaveLen(3))
			Expect(round["remote_created"]).To(Equal(true))
		})
	})
})
