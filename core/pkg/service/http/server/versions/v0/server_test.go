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
	"io"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/synnax/pkg/service/http/server/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	http "github.com/synnaxlabs/x/http/versions/v0"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/types"
	"github.com/synnaxlabs/x/validate"
	"github.com/vmihailenco/msgpack/v5"
)

var _ = Describe("Server", func() {
	server := func() v0.Server {
		return v0.Server{
			Key: uuid.MustParse(
				"22222222-2222-4222-8222-222222222222",
			),
			Rack:                  1,
			Name:                  "Weather API",
			Scheme:                v0.Scheme{Variant: v0.HttpsScheme{}},
			Host:                  "api.example.com",
			Timeout:               telem.Second,
			MaxConcurrentRequests: 6,
			Auth:                  v0.Auth{Variant: v0.NoneAuth{}},
			HealthCheck:           v0.HealthCheck{Method: http.MethodGet},
		}
	}

	read := func(pointer string) v0.Read {
		return v0.Read{
			Request: v0.ReadRequest{Method: http.MethodGet},
			Pointer: pointer,
			Index:   1,
			Channel: 2,
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
			s.Reads = []v0.Read{{
				Request: v0.ReadRequest{
					Method: http.MethodGet,
					Path:   "/weather",
					Query:  []http.QueryParam{{Parameter: "units", Value: "si"}},
				},
				Pointer: "/temp",
				Index:   1,
				Channel: 2,
			}}
			s.Writes = []v0.Write{{
				Request: v0.Request{
					Method: http.MethodPut,
					Path:   "/valve",
					Query:  []http.QueryParam{{Parameter: "id", Value: "1"}},
				},
				Channel: 3,
			}}
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
		Entry("a host with a scheme", func(s *v0.Server) {
			s.Host = "https://api.example.com"
		}, "host"),
		Entry("a host with a port", func(s *v0.Server) {
			s.Host = "api.example.com:8080"
		}, "host"),
		Entry("an IPv4 host with a port", func(s *v0.Server) {
			s.Host = "192.168.1.10:8080"
		}, "host"),
		Entry("a hex host with a port", func(s *v0.Server) {
			s.Host = "cafe.beef:8080"
		}, "host"),
		Entry("a host with a path", func(s *v0.Server) {
			s.Host = "api.example.com/v1"
		}, "host"),
		Entry("port 0", func(s *v0.Server) { s.Port = new(uint16(0)) }, "port"),
		Entry("a base path with no leading slash", func(s *v0.Server) {
			s.BasePath = "api"
		}, "base_path"),
		Entry("a base path with a query", func(s *v0.Server) {
			s.BasePath = "/api?v=1"
		}, "base_path"),
		Entry("a health check path with a fragment", func(s *v0.Server) {
			s.HealthCheck.Path = "/health#top"
		}, "path"),
		Entry("a read path with a query", func(s *v0.Server) {
			r := read("")
			r.Request.Path = "/now?units=si"
			s.Reads = []v0.Read{r}
		}, "path"),
		Entry("a write path with whitespace", func(s *v0.Server) {
			s.Writes = []v0.Write{{
				Request: v0.Request{Method: http.MethodPost, Path: "/a b"},
				Channel: 3,
			}}
		}, "path"),
		Entry("no scheme", func(s *v0.Server) { s.Scheme = v0.Scheme{} }, "type"),
		Entry("a timeout under 10 ms", func(s *v0.Server) {
			s.Timeout = 9 * telem.Millisecond
		}, "timeout"),
		Entry(
			"no concurrent requests",
			func(s *v0.Server) { s.MaxConcurrentRequests = 0 },
			"max_concurrent_requests",
		),
		Entry(
			"more than 64 concurrent requests",
			func(s *v0.Server) { s.MaxConcurrentRequests = 65 },
			"max_concurrent_requests",
		),
		Entry("a read with no index", func(s *v0.Server) {
			s.Reads = []v0.Read{{Channel: 2}}
		}, "index"),
		Entry("a read with no channel", func(s *v0.Server) {
			s.Reads = []v0.Read{{Index: 1}}
		}, "channel"),
		Entry("a read pointer with no leading slash", func(s *v0.Server) {
			s.Reads = []v0.Read{read("temp")}
		}, "pointer"),
		Entry("a read pointer with a bad escape", func(s *v0.Server) {
			s.Reads = []v0.Read{read("/a~2b")}
		}, "pointer"),
		Entry("a write with no channel", func(s *v0.Server) {
			s.Writes = []v0.Write{{}}
		}, "channel"),
		Entry("a health check pointer with a bad escape", func(s *v0.Server) {
			s.HealthCheck.Response = &v0.Response{
				Variant: v0.NullResponse{Pointer: "/a~"},
			}
		}, "pointer"),
		Entry("no auth", func(s *v0.Server) { s.Auth = v0.Auth{} }, "type"),
		Entry("a bearer token with no secret", func(s *v0.Server) {
			s.Auth = v0.Auth{Variant: v0.BearerAuth{}}
		}, "token"),
		Entry("basic auth with no password secret", func(s *v0.Server) {
			s.Auth = v0.Auth{Variant: v0.BasicAuth{Username: "admin"}}
		}, "password"),
		Entry("an API key with no secret", func(s *v0.Server) {
			s.Auth = v0.Auth{Variant: v0.APIKeyAuth{SendAs: "header", Name: "k"}}
		}, "key"),
		Entry("basic auth with no user name", func(s *v0.Server) {
			s.Auth = v0.Auth{Variant: v0.BasicAuth{Password: uuid.New()}}
		}, "username"),
		Entry("basic auth with a colon in the user name", func(s *v0.Server) {
			s.Auth = v0.Auth{Variant: v0.BasicAuth{
				Username: "ad:min",
				Password: uuid.New(),
			}}
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

	DescribeTable("Should accept a valid address and pointer",
		func(change func(*v0.Server)) {
			s := server()
			change(&s)
			Expect(s.Validate()).To(Succeed())
		},
		Entry("an IPv4 host", func(s *v0.Server) { s.Host = "192.168.1.10" }),
		Entry("an IPv6 host", func(s *v0.Server) { s.Host = "fe80::1" }),
		Entry("the IPv6 loopback", func(s *v0.Server) { s.Host = "::1" }),
		Entry("a full IPv6 host", func(s *v0.Server) {
			s.Host = "2001:db8:0:0:0:0:0:1"
		}),
		Entry("an IPv4-mapped IPv6 host", func(s *v0.Server) {
			s.Host = "0:0:0:0:0:ffff:192.168.1.10"
		}),
		Entry("a single label host", func(s *v0.Server) { s.Host = "localhost" }),
		Entry("a port and base path", func(s *v0.Server) {
			s.Port = new(uint16(8080))
			s.BasePath = "/api/v1"
		}),
		Entry("plain HTTP", func(s *v0.Server) {
			s.Scheme = v0.Scheme{Variant: v0.HttpScheme{}}
		}),
		Entry("a whole body pointer", func(s *v0.Server) {
			s.Reads = []v0.Read{read("")}
		}),
		Entry("an escaped pointer", func(s *v0.Server) {
			s.Reads = []v0.Read{read("/a~1b/~0c/0")}
		}),
	)
})

var _ = Describe("msgpack", func() {
	DescribeTable("Should round trip a scheme",
		func(sc v0.Scheme) {
			var decoded v0.Scheme
			Expect(msgpack.Unmarshal(MustSucceed(msgpack.Marshal(sc)), &decoded)).
				To(Succeed())
			Expect(decoded).To(Equal(sc))
		},
		Entry("HTTPS", v0.Scheme{Variant: v0.HttpsScheme{
			CertificateValidationDisabled: true,
		}}),
		Entry("HTTP", v0.Scheme{Variant: v0.HttpScheme{}}),
		Entry("no variant", v0.Scheme{}),
	)

	It("Should reject an unknown scheme type", func() {
		raw := MustSucceed(msgpack.Marshal(map[string]any{"type": "ftp"}))
		var decoded v0.Scheme
		Expect(msgpack.Unmarshal(raw, &decoded)).
			To(MatchError(validate.ErrValidation))
	})

	DescribeTable("Should round trip an auth",
		func(a v0.Auth) {
			var decoded v0.Auth
			Expect(msgpack.Unmarshal(MustSucceed(msgpack.Marshal(a)), &decoded)).
				To(Succeed())
			Expect(decoded).To(Equal(a))
		},
		Entry("none", v0.Auth{Variant: v0.NoneAuth{}}),
		Entry("bearer", v0.Auth{Variant: v0.BearerAuth{Token: uuid.New()}}),
		Entry("basic", v0.Auth{Variant: v0.BasicAuth{
			Username: "admin",
			Password: uuid.New(),
		}}),
		Entry("API key", v0.Auth{Variant: v0.APIKeyAuth{
			Key:    uuid.New(),
			SendAs: v0.SendAsQueryParam,
			Name:   "key",
		}}),
		Entry("no variant", v0.Auth{}),
	)

	DescribeTable("Should round trip a response",
		func(r v0.Response) {
			var decoded v0.Response
			Expect(msgpack.Unmarshal(MustSucceed(msgpack.Marshal(r)), &decoded)).
				To(Succeed())
			Expect(decoded).To(Equal(r))
		},
		Entry("string", v0.Response{Variant: v0.StringResponse{
			Pointer:       "/status",
			ExpectedValue: "ok",
		}}),
		Entry("number", v0.Response{Variant: v0.NumberResponse{
			Pointer:       "/code",
			ExpectedValue: 200,
		}}),
		Entry("boolean", v0.Response{Variant: v0.BooleanResponse{
			Pointer:       "/up",
			ExpectedValue: true,
		}}),
		Entry("null", v0.Response{Variant: v0.NullResponse{Pointer: "/error"}}),
	)

	It("Should apply the defaults of an API key", func() {
		raw := MustSucceed(msgpack.Marshal(map[string]any{
			"type": "api_key",
			"name": "X-Key",
		}))
		var decoded v0.Auth
		Expect(msgpack.Unmarshal(raw, &decoded)).To(Succeed())
		Expect(decoded.Variant).To(Equal(v0.APIKeyAuth{
			SendAs: v0.SendAsHeader,
			Name:   "X-Key",
		}))
	})

	It("Should reject an unknown auth type", func() {
		raw := MustSucceed(msgpack.Marshal(map[string]any{"type": "oauth"}))
		var decoded v0.Auth
		Expect(msgpack.Unmarshal(raw, &decoded)).
			To(MatchError(validate.ErrValidation))
	})

	It("Should reject an unknown expected value type", func() {
		raw := MustSucceed(msgpack.Marshal(map[string]any{
			"expected_value_type": "date",
		}))
		var decoded v0.Response
		Expect(msgpack.Unmarshal(raw, &decoded)).
			To(MatchError(validate.ErrValidation))
	})

	DescribeTable("Should reject a variant whose fields have the wrong type",
		func(target any, fields map[string]any) {
			raw := MustSucceed(msgpack.Marshal(fields))
			Expect(msgpack.Unmarshal(raw, target)).
				To(MatchError(ContainSubstring("msgpack: ")))
		},
		Entry("HTTPS", &v0.Scheme{}, map[string]any{
			"type":                            "https",
			"certificate_validation_disabled": "yes",
		}),
		Entry("bearer", &v0.Auth{}, map[string]any{"type": "bearer", "token": 1}),
		Entry("basic", &v0.Auth{}, map[string]any{"type": "basic", "username": 1}),
		Entry("API key", &v0.Auth{}, map[string]any{"type": "api_key", "name": 1}),
		Entry("none", &v0.Auth{}, map[string]any{"type": []int{1}}),
		Entry("string", &v0.Response{}, map[string]any{
			"expected_value_type": "string",
			"expected_value":      1,
		}),
		Entry("number", &v0.Response{}, map[string]any{
			"expected_value_type": "number",
			"expected_value":      "one",
		}),
		Entry("boolean", &v0.Response{}, map[string]any{
			"expected_value_type": "boolean",
			"expected_value":      "yes",
		}),
		Entry("null", &v0.Response{}, map[string]any{
			"expected_value_type": "null",
			"pointer":             1,
		}),
	)

	DescribeTable("Should reject truncated input",
		func(target any) {
			Expect(msgpack.Unmarshal([]byte{0x81}, target)).
				To(MatchError(io.EOF))
		},
		Entry("scheme", &v0.Scheme{}),
		Entry("auth", &v0.Auth{}),
		Entry("response", &v0.Response{}),
	)
})
