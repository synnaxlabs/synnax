// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package auth_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/oidc"
)

var _ = Describe("ServiceConfig", func() {
	full := func() apiauth.ServiceConfig {
		return apiauth.ServiceConfig{
			Auth: &auth.Service{},
			OIDC: &oidc.Service{},
		}
	}

	Describe("Validate", func() {
		It("Should succeed when every service is set", func() {
			Expect(full().Validate()).To(Succeed())
		})

		DescribeTable(
			"Should fail when a service is missing",
			func(clear func(*apiauth.ServiceConfig), msg string) {
				cfg := full()
				clear(&cfg)
				Expect(cfg.Validate()).To(MatchError(msg))
			},
			Entry(
				"auth",
				func(c *apiauth.ServiceConfig) { c.Auth = nil },
				"auth: must be non-nil",
			),
			Entry(
				"oidc",
				func(c *apiauth.ServiceConfig) { c.OIDC = nil },
				"oidc: must be non-nil",
			),
		)
	})

	Describe("Override", func() {
		It("Should take the other config's services when set", func() {
			cfg := full()
			merged := apiauth.ServiceConfig{}.Override(cfg)
			Expect(merged.Auth).To(BeIdenticalTo(cfg.Auth))
			Expect(merged.OIDC).To(BeIdenticalTo(cfg.OIDC))
		})

		It("Should keep the receiver's services when the other's are nil", func() {
			cfg := full()
			merged := cfg.Override(apiauth.ServiceConfig{})
			Expect(merged.Auth).To(BeIdenticalTo(cfg.Auth))
			Expect(merged.OIDC).To(BeIdenticalTo(cfg.OIDC))
		})
	})
})
