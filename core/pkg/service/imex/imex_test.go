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
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	. "github.com/synnaxlabs/x/testutil"
)

// wirePayload is the shape used by the envelope-level tests to exercise Decode and
// Encode through the flat wire format. Tag-driven Encode keys the body in lowercase;
// Decode reads the same keys back via the json tags.
type wirePayload struct {
	Name string `json:"name"`
	Foo  int    `json:"foo"`
	Bar  string `json:"bar,omitempty"`
}

var _ = Describe("ImEx", func() {
	Describe("Envelope", func() {
		Describe("UnmarshalJSON", func() {
			It("Should extract promoted headers and retain the body for typed decode", func(ctx SpecContext) {
				src := []byte(`{"version":54,"type":"log","name":"n","foo":1}`)
				var env imex.Envelope
				Expect(json.Unmarshal(src, &env)).To(Succeed())
				Expect(env.Version).To(Equal(imex.Version(54)))
				Expect(env.Type).To(Equal("log"))
				Expect(env.Name).To(Equal("n"))
				p := MustSucceed(imex.Decode[wirePayload](ctx, env))
				Expect(p.Foo).To(Equal(1))
				Expect(p.Name).To(Equal("n"))
			})

			It("Should accept a legacy N.0.0 version string", func() {
				var env imex.Envelope
				Expect(json.Unmarshal(
					[]byte(`{"version":"4.0.0","type":"log","name":"n"}`), &env,
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

			It("Should reject a negative numeric version", func() {
				var env imex.Envelope
				Expect(json.Unmarshal([]byte(`{"version":-1}`), &env)).To(
					MatchError(ContainSubstring("invalid version number")),
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
				// Provide a valid type so the unmarshal flow reaches the name
				// type-assertion instead of short-circuiting on missing type.
				var env imex.Envelope
				Expect(json.Unmarshal(
					[]byte(`{"version":1,"type":"log","name":[]}`), &env,
				)).To(MatchError(ContainSubstring("name must be a string")))
			})

			It("Should reject a null payload because type is missing", func() {
				var env imex.Envelope
				Expect(json.Unmarshal([]byte(`null`), &env)).To(
					MatchError(ContainSubstring("type must be a non-empty string")),
				)
			})

			It("Should reject an empty type", func() {
				var env imex.Envelope
				Expect(json.Unmarshal(
					[]byte(`{"version":1,"type":"","name":"n"}`), &env,
				)).To(MatchError(ContainSubstring("type must be a non-empty string")))
			})

			It("Should reject an empty name", func() {
				var env imex.Envelope
				Expect(json.Unmarshal(
					[]byte(`{"version":1,"type":"log","name":""}`), &env,
				)).To(MatchError(ContainSubstring("name must be a non-empty string")))
			})

			It("Should error when the input is a bare JSON number", func() {
				var env imex.Envelope
				Expect(json.Unmarshal([]byte(`34`), &env)).To(
					MatchError(ContainSubstring("cannot unmarshal number")),
				)
			})

			It("Should preserve int64 precision on payload values past 2^53", func(ctx SpecContext) {
				// 1700000000000000000 is roughly a current Unix nanosecond timestamp,
				// well past float64's 53-bit mantissa (9007199254740992). The previous
				// float64-based decode lost the low-order bits; capturing raw bytes
				// preserves them all the way to the typed decode.
				const big int64 = 1700000000000000000
				src := []byte(`{"version":1,"type":"log","name":"n","ts":1700000000000000000}`)
				var env imex.Envelope
				Expect(json.Unmarshal(src, &env)).To(Succeed())
				type tsPayload struct {
					TS int64 `json:"ts"`
				}
				p := MustSucceed(imex.Decode[tsPayload](ctx, env))
				Expect(p.TS).To(Equal(big))
			})
		})

		Describe("MarshalJSON", func() {
			It("Should emit the body built by Encode, with headers at the top level", func() {
				env := MustSucceed(imex.Encode(
					wirePayload{Name: "n", Foo: 1, Bar: "x"},
					54,
					ontology.ResourceType("log"),
				))
				b := MustSucceed(json.Marshal(env))
				var round map[string]any
				Expect(json.Unmarshal(b, &round)).To(Succeed())
				Expect(round["version"]).To(BeNumerically("==", 54))
				Expect(round["type"]).To(Equal("log"))
				Expect(round["name"]).To(Equal("n"))
				Expect(round["foo"]).To(BeNumerically("==", 1))
				Expect(round["bar"]).To(Equal("x"))
			})

			It("Should error when marshaling a hand-constructed envelope with no body", func() {
				// Hand-constructed envelopes have a nil body. MarshalJSON refuses
				// rather than silently emitting JSON null, so a service that
				// accidentally returns an empty Envelope surfaces the bug at the
				// transport boundary rather than over the wire.
				env := imex.Envelope{Version: 1, Type: "log", Name: "n"}
				_, err := json.Marshal(env)
				Expect(err).To(MatchError(ContainSubstring("envelope has no body")))
			})
		})

		Describe("Decode", func() {
			It("Should decode the body into the requested type", func(ctx SpecContext) {
				src := []byte(`{"version":1,"type":"log","name":"n","foo":42,"bar":"x"}`)
				var env imex.Envelope
				Expect(json.Unmarshal(src, &env)).To(Succeed())
				p := MustSucceed(imex.Decode[wirePayload](ctx, env))
				Expect(p.Name).To(Equal("n"))
				Expect(p.Foo).To(Equal(42))
				Expect(p.Bar).To(Equal("x"))
			})

			It("Should fail on a type mismatch in the body", func(ctx SpecContext) {
				src := []byte(`{"version":1,"type":"log","name":"n","foo":"not a number"}`)
				var env imex.Envelope
				Expect(json.Unmarshal(src, &env)).To(Succeed())
				Expect(imex.Decode[wirePayload](ctx, env)).Error().To(
					MatchError(ContainSubstring("decode envelope body")),
				)
			})

		})

		Describe("Encode", func() {
			It("Should stamp the headers and surface them on the envelope", func() {
				env := MustSucceed(imex.Encode(
					wirePayload{Name: "n", Foo: 1},
					7,
					ontology.ResourceType("log"),
				))
				Expect(env.Version).To(Equal(imex.Version(7)))
				Expect(env.Type).To(Equal("log"))
				Expect(env.Name).To(Equal("n"))
			})

			It("Should fail when the data is missing a top-level name field", func() {
				type missingName struct {
					Foo int `json:"foo"`
				}
				Expect(imex.Encode(missingName{Foo: 1}, 1, "log")).Error().To(
					MatchError(ContainSubstring("name")),
				)
			})

			It("Should fail when the top-level name field is the wrong type", func() {
				type badName struct {
					Name int `json:"name"`
				}
				Expect(imex.Encode(badName{Name: 5}, 1, "log")).Error().To(
					MatchError(ContainSubstring("name")),
				)
			})

			It("Should fail when the top-level name field is empty", func() {
				Expect(imex.Encode(
					wirePayload{Name: "", Foo: 1}, 1, "log",
				)).Error().To(MatchError(ContainSubstring("name")))
			})

			It("Should fail when a field tagged json:\"-\" hides the name field", func() {
				type hidden struct {
					Name string `json:"-"`
					Foo  int    `json:"foo"`
				}
				Expect(imex.Encode(hidden{Name: "n", Foo: 1}, 1, "log")).Error().To(
					MatchError(ContainSubstring("name")),
				)
			})

			It("Should key the wire body by the json tag, not the Go field name", func() {
				type payload struct {
					Name     string `json:"name"`
					FieldOne string `json:"field_one"`
				}
				env := MustSucceed(imex.Encode(
					payload{Name: "n", FieldOne: "v"}, 1, "log",
				))
				b := MustSucceed(json.Marshal(env))
				var round map[string]any
				Expect(json.Unmarshal(b, &round)).To(Succeed())
				Expect(round).To(HaveKey("field_one"))
				Expect(round).NotTo(HaveKey("FieldOne"))
			})

			It("Should strip comma options like omitempty from the json tag", func() {
				type payload struct {
					Name string `json:"name"`
					Foo  int    `json:"foo,omitempty"`
				}
				env := MustSucceed(imex.Encode(
					payload{Name: "n", Foo: 1}, 1, "log",
				))
				b := MustSucceed(json.Marshal(env))
				var round map[string]any
				Expect(json.Unmarshal(b, &round)).To(Succeed())
				Expect(round).To(HaveKey("foo"))
				Expect(round).NotTo(HaveKey("foo,omitempty"))
			})

			It("Should skip fields tagged json:\"-\" from the wire body", func() {
				type payload struct {
					Name   string `json:"name"`
					Hidden string `json:"-"`
				}
				env := MustSucceed(imex.Encode(
					payload{Name: "n", Hidden: "secret"}, 1, "log",
				))
				b := MustSucceed(json.Marshal(env))
				var round map[string]any
				Expect(json.Unmarshal(b, &round)).To(Succeed())
				Expect(round).NotTo(HaveKey("Hidden"))
				Expect(round).NotTo(HaveKey("-"))
			})

			It("Should skip unexported fields even when they carry a json tag", func() {
				// Encode goes through structToMap, which only reflects over exported
				// fields. An unexported field with a json tag should never reach the
				// wire — calling Interface() on it would panic, and exposing it would
				// leak private state.
				type payload struct {
					Name string `json:"name"`
					//nolint:govet,staticcheck
					secret string `json:"secret"`
				}
				env := MustSucceed(imex.Encode(
					payload{Name: "n", secret: "shh"}, 1, "log",
				))
				b := MustSucceed(json.Marshal(env))
				var round map[string]any
				Expect(json.Unmarshal(b, &round)).To(Succeed())
				Expect(round).NotTo(HaveKey("secret"))
			})

			It("Should skip exported fields that have no json tag", func() {
				type payload struct {
					Name     string `json:"name"`
					Untagged string
				}
				env := MustSucceed(imex.Encode(
					payload{Name: "n", Untagged: "v"}, 1, "log",
				))
				b := MustSucceed(json.Marshal(env))
				var round map[string]any
				Expect(json.Unmarshal(b, &round)).To(Succeed())
				Expect(round).NotTo(HaveKey("Untagged"))
			})

			It("Should skip fields whose json tag has an empty name", func() {
				// `json:",omitempty"` has no name component; structToMap only
				// emits fields keyed by an explicit tag name.
				type payload struct {
					Name      string `json:"name"`
					EmptyName string `json:",omitempty"`
				}
				env := MustSucceed(imex.Encode(
					payload{Name: "n", EmptyName: "v"}, 1, "log",
				))
				b := MustSucceed(json.Marshal(env))
				var round map[string]any
				Expect(json.Unmarshal(b, &round)).To(Succeed())
				Expect(round).NotTo(HaveKey("EmptyName"))
			})

			It("Should reject a pointer to the resource", func() {
				Expect(imex.Encode(
					&wirePayload{Name: "n", Foo: 1}, 1, "log",
				)).Error().To(MatchError(ContainSubstring("expected struct")))
			})

			It("Should preserve nested struct, slice, and map fields on the wire", func() {
				type inner struct {
					A int `json:"a"`
				}
				type payload struct {
					Name     string            `json:"name"`
					Inner    inner             `json:"inner"`
					Channels []int             `json:"channels"`
					Labels   map[string]string `json:"labels"`
				}
				env := MustSucceed(imex.Encode(payload{
					Name:     "n",
					Inner:    inner{A: 1},
					Channels: []int{1, 2, 3},
					Labels:   map[string]string{"k": "v"},
				}, 1, "log"))
				b := MustSucceed(json.Marshal(env))
				var round map[string]any
				Expect(json.Unmarshal(b, &round)).To(Succeed())
				Expect(round["inner"]).To(HaveKeyWithValue("a", BeNumerically("==", 1)))
				Expect(round["channels"]).To(HaveLen(3))
				Expect(round["labels"]).To(HaveKeyWithValue("k", "v"))
			})
		})

		Describe("Round-trip", func() {
			It("Should preserve nested content across Encode → Marshal → Unmarshal → Decode", func(ctx SpecContext) {
				type payload struct {
					Name          string  `json:"name"`
					Channels      []int64 `json:"channels"`
					RemoteCreated bool    `json:"remote_created"`
				}
				src := payload{Name: "n", Channels: []int64{1, 2, 3}, RemoteCreated: true}
				env := MustSucceed(imex.Encode(src, 7, ontology.ResourceType("log")))
				b := MustSucceed(json.Marshal(env))
				var roundEnv imex.Envelope
				Expect(json.Unmarshal(b, &roundEnv)).To(Succeed())
				Expect(roundEnv.Version).To(Equal(imex.Version(7)))
				Expect(roundEnv.Type).To(Equal("log"))
				Expect(roundEnv.Name).To(Equal("n"))
				out := MustSucceed(imex.Decode[payload](ctx, roundEnv))
				Expect(out).To(Equal(src))
			})
		})
	})

	Describe("NewErrUnsupportedVersion", func() {
		It("Should produce a message naming the resource type, given, and supported versions", func() {
			Expect(imex.NewErrUnsupportedVersion("log", 5, 1)).To(And(
				MatchError(ContainSubstring("log version 5")),
				MatchError(ContainSubstring("newer than this Core supports")),
				MatchError(ContainSubstring("latest: 1")),
			))
		})

		It("Should mention validation in the error message", func() {
			Expect(imex.NewErrUnsupportedVersion("log", 5, 1)).
				To(MatchError(ContainSubstring("validation")))
		})

		It("Should be path-scoped to the version field", func() {
			Expect(imex.NewErrUnsupportedVersion("log", 5, 1)).
				To(MatchError(ContainSubstring("version")))
		})

		It("Should reflect the resource type passed in", func() {
			Expect(imex.NewErrUnsupportedVersion("schematic", 99, 4)).
				To(And(
					MatchError(ContainSubstring("schematic version 99")),
					MatchError(ContainSubstring("latest: 4")),
				))
		})
	})
})
