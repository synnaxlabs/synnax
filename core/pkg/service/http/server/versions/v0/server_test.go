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
	"encoding/json/v2"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v1"
	v0 "github.com/synnaxlabs/synnax/pkg/service/http/server/versions/v0"
	http "github.com/synnaxlabs/synnax/pkg/service/http/versions/v4"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/types"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Server", func() {
	server := func() v0.Server {
		return v0.Server{
			Key: uuid.MustParse(
				"22222222-2222-4222-8222-222222222222",
			),
			Rack:                  1,
			Name:                  "Weather API",
			Scheme:                v0.SchemeHTTPS,
			Host:                  "api.example.com",
			TimeoutMs:             100,
			MaxConcurrentRequests: 6,
			Auth:                  v0.Auth{Variant: v0.NoneAuth{}},
			HealthCheck:           v0.HealthCheck{Method: http.MethodGet},
		}
	}

	It("Should have a table prefix of its own", func() {
		Expect(types.Name[v0.Server]()).To(Equal("http_server"))
	})

	It("Should identify itself by its key", func() {
		s := server()
		Expect(s.GorpKey()).To(Equal(s.Key))
		Expect(s.ResourceID()).To(Equal(resource.ID{
			Type: resource.TypeHTTPServer,
			Key:  s.Key.String(),
		}))
	})

	It("Should apply the defaults", func() {
		var s v0.Server
		Expect(json.Unmarshal([]byte(`{
			"key": "22222222-2222-4222-8222-222222222222",
			"rack": 1,
			"name": "Weather API",
			"host": "api.example.com"
		}`), &s)).To(Succeed())
		Expect(s).To(Equal(server()))
		Expect(s.Validate()).To(Succeed())
	})

	It("Should apply the defaults of an API key", func() {
		var a v0.Auth
		Expect(json.Unmarshal(
			[]byte(`{"type": "api_key", "name": "X-API-Key"}`),
			&a,
		)).To(Succeed())
		Expect(a.Variant).To(Equal(v0.APIKeyAuth{
			SendAs: v0.SendAsHeader,
			Name:   "X-API-Key",
		}))
	})

	DescribeTable("Should round trip a server through JSON",
		func(auth v0.AuthVariant, response *v0.Response) {
			s := server()
			s.Auth = v0.Auth{Variant: auth}
			s.HealthCheck.Response = response
			s.HealthCheck.Headers = []http.Header{{Name: "Accept", Value: "json"}}
			s.HealthCheck.QueryParams = []http.QueryParam{{Parameter: "q", Value: "1"}}
			s.ReadChannels = map[string]v0.ReadChannels{
				"/weather": {Index: 1, Channels: map[string]channel.Key{"/temp": 2}},
			}
			s.WriteChannels = map[string]channel.Key{"/valve": 3}
			var decoded v0.Server
			Expect(json.Unmarshal(MustSucceed(json.Marshal(s)), &decoded)).To(Succeed())
			Expect(decoded).To(Equal(s))
		},
		Entry("bearer", v0.BearerAuth{Token: uuid.New()}, nil),
		Entry(
			"basic",
			v0.BasicAuth{Username: "admin", Password: uuid.New()},
			&v0.Response{Variant: v0.NumberResponse{
				Pointer:       "/code",
				ExpectedValue: 200,
			}},
		),
		Entry(
			"API key",
			v0.APIKeyAuth{Key: uuid.New(), SendAs: v0.SendAsQueryParam, Name: "key"},
			&v0.Response{Variant: v0.NullResponse{Pointer: "/error"}},
		),
	)

	DescribeTable("Should reject an invalid server",
		func(change func(*v0.Server), path string) {
			s := server()
			change(&s)
			Expect(s.Validate()).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(path)),
			))
		},
		Entry("no name", func(s *v0.Server) { s.Name = "" }, "name"),
		Entry("no host", func(s *v0.Server) { s.Host = "" }, "host"),
		Entry("an unknown scheme", func(s *v0.Server) { s.Scheme = "ftp" }, "scheme"),
		Entry("a zero timeout", func(s *v0.Server) { s.TimeoutMs = 0 }, "timeout_ms"),
		Entry(
			"no concurrent requests",
			func(s *v0.Server) { s.MaxConcurrentRequests = 0 },
			"max_concurrent_requests",
		),
		Entry("no auth", func(s *v0.Server) { s.Auth = v0.Auth{} }, "type"),
		Entry("basic auth with no user name", func(s *v0.Server) {
			s.Auth = v0.Auth{Variant: v0.BasicAuth{Password: uuid.New()}}
		}, "username"),
		Entry("an API key with no name", func(s *v0.Server) {
			s.Auth = v0.Auth{Variant: v0.APIKeyAuth{Key: uuid.New(), SendAs: "header"}}
		}, "name"),
		Entry("an unknown place for an API key", func(s *v0.Server) {
			s.Auth = v0.Auth{Variant: v0.APIKeyAuth{
				Key:    uuid.New(),
				SendAs: "body",
				Name:   "key",
			}}
		}, "send_as"),
		Entry("an unknown health check method", func(s *v0.Server) {
			s.HealthCheck.Method = "FETCH"
		}, "method"),
	)
})
