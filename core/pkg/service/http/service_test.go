// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package http_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/http"
	"github.com/synnaxlabs/synnax/pkg/service/http/versions/legacy"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Service", func() {
	var svc *http.Service
	BeforeEach(func(ctx SpecContext) {
		svc = MustOpen(http.OpenService(ctx, http.ServiceConfig{
			DB: db,
		}))
	})

	Describe("OpenService", func() {
		It("Should reject a config missing the DB", func(ctx SpecContext) {
			Expect(http.OpenService(ctx, http.ServiceConfig{})).Error().
				To(MatchError(ContainSubstring("db: must be non-nil")))
		})
	})

	Describe("Stores", func() {
		It("Should expose one store per HTTP task type", func() {
			types := []string{}
			for _, s := range svc.Stores() {
				types = append(types, s.Type())
			}
			Expect(types).To(ConsistOf(
				"http_read",
				"http_write",
				"http_scan",
			))
		})
	})

	Describe("Normalize", func() {
		DescribeTable(
			"Should import both read shapes released Cores stamped at version 2",
			func(ctx SpecContext, data msgpack.EncodedJSON, nanFill bool) {
				out := MustSucceed(svc.Read.Normalize(ctx, 2, data))
				Expect(out).To(HaveKeyWithValue("nan_fill_enabled", nanFill))
				Expect(out).To(HaveKeyWithValue("device", "dev-1"))
			},
			Entry("v2", msgpack.EncodedJSON{"device": "dev-1"}, false),
			Entry(
				"v3",
				msgpack.EncodedJSON{"device": "dev-1", "nan_fill_enabled": true},
				true,
			),
		)

		It(
			"Should export read configs above every version released Cores stamped",
			func() {
				Expect(svc.Read.Version()).To(BeNumerically(">", legacy.LastVersion))
			},
		)
	})

	Describe("Write", func() {
		It("Should store a decoded read config under the given key", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			MustSucceed(svc.Read.Write(ctx, nil, key, msgpack.EncodedJSON{
				"device": "dev-1",
				"rate":   25,
				"endpoints": []any{map[string]any{
					"key":    "ep-1",
					"method": "GET",
					"path":   "/telemetry",
				}},
			}))
			data := MustSucceed(svc.Read.Read(ctx, nil, key))
			Expect(data["key"]).To(Equal(key.String()))
			Expect(data["device"]).To(Equal("dev-1"))
			Expect(data["rate"]).To(BeNumerically("==", 25))
		})

		It("Should apply read config schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			MustSucceed(svc.Read.Write(ctx, nil, key, msgpack.EncodedJSON{}))
			data := MustSucceed(svc.Read.Read(ctx, nil, key))
			Expect(data["rate"]).To(BeNumerically("==", 1))
		})

		It("Should apply write endpoint schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			MustSucceed(svc.Write.Write(ctx, nil, key, msgpack.EncodedJSON{
				"endpoints": []any{map[string]any{"key": "ep-1"}},
			}))
			data := MustSucceed(svc.Write.Read(ctx, nil, key))
			Expect(data["endpoints"]).To(HaveExactElements(
				HaveKeyWithValue("method", "POST"),
			))
		})

		It("Should apply scan config schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			MustSucceed(svc.Scan.Write(ctx, nil, key, msgpack.EncodedJSON{}))
			data := MustSucceed(svc.Scan.Read(ctx, nil, key))
			Expect(data["rate"]).To(BeNumerically("==", 0.2))
		})

		It("Should return the read validation error for an invalid endpoint", func(
			ctx SpecContext,
		) {
			Expect(svc.Read.Write(ctx, nil, uuid.New(), msgpack.EncodedJSON{
				"endpoints": []any{map[string]any{"method": "BOGUS"}},
			})).Error().To(MatchError(ContainSubstring("invalid method: BOGUS")))
		})

		It("Should return the write validation error for an invalid endpoint", func(
			ctx SpecContext,
		) {
			Expect(svc.Write.Write(ctx, nil, uuid.New(), msgpack.EncodedJSON{
				"endpoints": []any{map[string]any{"method": "BOGUS"}},
			})).Error().To(MatchError(ContainSubstring("invalid method: BOGUS")))
		})
	})
})
