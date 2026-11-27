// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v5_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v4 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v4"
	v5 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v5"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("StoredMethod", func() {
	DescribeTable("Should accept every method a stored endpoint can hold",
		func(m v5.StoredMethod) { Expect(m.IsValid()).To(BeTrue()) },
		Entry("QUERY", v5.StoredMethodQuery),
		Entry("GET", v5.StoredMethodGet),
		Entry("TRACE", v5.StoredMethodTrace),
		Entry("CONNECT", v5.StoredMethodConnect),
	)

	It("Should refuse an unknown method", func() {
		Expect(v5.StoredMethod("FETCH").IsValid()).To(BeFalse())
		Expect(v5.ReadEndpoint{Method: "FETCH"}.Validate()).
			To(MatchError(validate.ErrValidation))
	})

	It("Should keep a TRACE method through the v4 migration", func(ctx SpecContext) {
		ep := MustSucceed(v5.MigrateReadEndpoint(ctx, v4.ReadEndpoint{
			Key:    "ep",
			Method: "TRACE",
		}))
		Expect(ep.Method).To(Equal(v5.StoredMethodTrace))
	})
})

var _ = Describe("MigrateWriteConfig", func() {
	It("Should keep the endpoints and their methods", func(ctx SpecContext) {
		old := v4.WriteConfig{
			Device: "dev",
			Endpoints: []v4.WriteEndpoint{
				{Key: "ep", Method: v4.MethodPut, Path: "/valve"},
			},
		}
		cfg := MustSucceed(v5.MigrateWriteConfig(ctx, old))
		Expect(cfg.Device).To(Equal("dev"))
		Expect(cfg.Endpoints).To(HaveLen(1))
		Expect(cfg.Endpoints[0].Method).To(Equal(v5.StoredMethodPut))
		Expect(cfg.Endpoints[0].Path).To(Equal("/valve"))
	})

	It("Should keep a CONNECT method of an endpoint", func(ctx SpecContext) {
		ep := MustSucceed(v5.MigrateWriteEndpoint(ctx, v4.WriteEndpoint{
			Key:    "ep",
			Method: "CONNECT",
		}))
		Expect(ep.Method).To(Equal(v5.StoredMethodConnect))
	})

	It("Should store the shape that v4 stored under the same table name", func() {
		Expect(v5.WriteConfig{}.CustomTypeName()).
			To(Equal(v4.WriteConfig{}.CustomTypeName()))
	})
})
