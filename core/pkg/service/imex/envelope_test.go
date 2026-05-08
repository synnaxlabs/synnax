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

var _ = Describe("Envelope", func() {
	Describe("UnmarshalJSON", func() {
		It("Should extract promoted fields and put the rest into Data", func() {
			src := []byte(`{"version":54,"type":"log","name":"n","foo":1}`)
			var env imex.Envelope
			Expect(json.Unmarshal(src, &env)).To(Succeed())
			Expect(env.Version).To(Equal(54))
			Expect(env.Type).To(Equal("log"))
			Expect(env.Name).To(Equal("n"))
			Expect(env.Data).To(HaveKeyWithValue("foo", BeNumerically("==", 1)))
			Expect(env.Data).NotTo(HaveKey("version"))
			Expect(env.Data).NotTo(HaveKey("type"))
			Expect(env.Data).NotTo(HaveKey("name"))
		})

		It("Should silently drop a top-level key field from older payloads", func() {
			src := []byte(`{"version":1,"type":"log","key":"ignored","channels":[]}`)
			var env imex.Envelope
			Expect(json.Unmarshal(src, &env)).To(Succeed())
			Expect(env.Data).NotTo(HaveKey("key"))
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
				[]byte(`{"version":"1.2.3","type":"log","channels":[]}`), &env,
			)).To(Succeed())
			Expect(env.Version).To(Equal(1))
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
					"channels": []any{float64(1), float64(2)},
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

		It("Should emit promoted fields even when Data is empty", func() {
			env := imex.Envelope{Version: 3, Type: "log"}
			b := MustSucceed(json.Marshal(env))
			var round map[string]any
			Expect(json.Unmarshal(b, &round)).To(Succeed())
			Expect(round["version"]).To(BeNumerically("==", 3))
			Expect(round["type"]).To(Equal("log"))
			Expect(round).NotTo(HaveKey("name"))
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
			Expect(dst.Version).To(Equal(7))
			Expect(dst.Type).To(Equal("log"))
			Expect(dst.Name).To(Equal("n"))
			Expect(dst.Data).To(HaveKey("channels"))
			Expect(dst.Data["channels"]).To(HaveLen(3))
			Expect(dst.Data["remote_created"]).To(Equal(true))
		})
	})
})
