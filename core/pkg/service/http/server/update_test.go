// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package server_test

import (
	"encoding/json/v2"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/http"
	"github.com/synnaxlabs/synnax/pkg/service/http/server"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"github.com/vmihailenco/msgpack/v5"
)

var _ = Describe("Update", func() {
	identity := server.Update{Variant: server.IdentityUpdate{
		Identity: server.Identity{
			Rack:                  1,
			Name:                  "Weather API",
			Scheme:                server.Scheme{Variant: server.HttpScheme{}},
			Host:                  "weather.example.com",
			Port:                  new(uint16(8080)),
			BasePath:              "/api",
			Timeout:               250 * telem.Millisecond,
			MaxConcurrentRequests: 4,
			Auth: server.Auth{Variant: server.BasicAuth{
				Username: "admin",
				Password: uuid.MustParse("33333333-3333-4333-8333-333333333333"),
			}},
			HealthCheck: server.HealthCheck{
				Method: http.MethodPost,
				Path:   "/health",
				Response: &server.Response{Variant: server.NumberResponse{
					Pointer:       "/code",
					ExpectedValue: 200,
				}},
			},
		},
	}}
	configuration := server.Update{Variant: server.ConfigurationUpdate{
		Configuration: server.Configuration{
			Reads: []server.Read{{
				Request: server.ReadRequest{
					Method: http.MethodGet,
					Path:   "/weather",
					Query:  []http.QueryParam{{Parameter: "units", Value: "si"}},
				},
				Pointer: "/temp",
				Index:   1,
				Channel: 2,
			}},
			Writes: []server.Write{{
				Request: server.Request{Method: http.MethodPut, Path: "/valve"},
				Channel: 3,
			}},
		},
	}}

	DescribeTable("Should keep the group through msgpack",
		func(update server.Update) {
			var got server.Update
			Expect(msgpack.Unmarshal(
				MustSucceed(msgpack.Marshal(update)),
				&got,
			)).To(Succeed())
			Expect(got).To(Equal(update))
		},
		Entry("identity", identity),
		Entry("configuration", configuration),
	)

	DescribeTable("Should keep the group through JSON",
		func(update server.Update) {
			var got server.Update
			Expect(json.Unmarshal(
				MustSucceed(json.Marshal(update)),
				&got,
			)).To(Succeed())
			Expect(got.Variant).To(BeAssignableToTypeOf(update.Variant))
			Expect(MustSucceed(json.Marshal(got))).
				To(MatchJSON(MustSucceed(json.Marshal(update))))
		},
		Entry("identity", identity),
		Entry("configuration", configuration),
	)

	It("Should reject an unknown group in JSON", func() {
		var got server.Update
		Expect(json.Unmarshal([]byte(`{"group":"other"}`), &got)).
			To(MatchError(ContainSubstring("unknown group")))
	})

	It("Should match the JSON shape in msgpack", func() {
		var wire map[string]any
		Expect(msgpack.Unmarshal(
			MustSucceed(msgpack.Marshal(identity)),
			&wire,
		)).To(Succeed())
		var fromJSON map[string]any
		Expect(json.Unmarshal(
			MustSucceed(json.Marshal(identity)),
			&fromJSON,
		)).To(Succeed())
		Expect(wire["group"]).To(Equal(fromJSON["group"]))
		Expect(wire["name"]).To(Equal(fromJSON["name"]))
	})

	It("Should encode an empty update as nil", func() {
		var got server.Update
		Expect(msgpack.Unmarshal(
			MustSucceed(msgpack.Marshal(server.Update{})),
			&got,
		)).To(Succeed())
		Expect(got.Variant).To(BeNil())
	})

	It("Should reject msgpack that is not an update", func() {
		var got server.Update
		Expect(msgpack.Unmarshal(
			MustSucceed(msgpack.Marshal("not an update")),
			&got,
		)).To(MatchError(ContainSubstring("msgpack")))
	})

	It("Should apply the identity defaults that msgpack leaves out", func() {
		raw := MustSucceed(msgpack.Marshal(map[string]any{
			"group": "identity",
			"rack":  1,
			"name":  "Weather API",
			"host":  "weather.example.com",
		}))
		var got server.Update
		Expect(msgpack.Unmarshal(raw, &got)).To(Succeed())
		var expected server.Identity
		expected.SetDefaults()
		expected.Rack = 1
		expected.Name = "Weather API"
		expected.Host = "weather.example.com"
		Expect(got.Variant).To(Equal(server.IdentityUpdate{Identity: expected}))
		Expect(expected.Validate()).To(Succeed())
	})

	It("Should reject an unknown group", func() {
		raw := MustSucceed(msgpack.Marshal(map[string]any{"group": "other"}))
		var got server.Update
		Expect(msgpack.Unmarshal(raw, &got)).To(MatchError(validate.ErrValidation))
	})
})
