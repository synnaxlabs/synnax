// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package whip_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiwhip "github.com/synnaxlabs/synnax/pkg/api/whip"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/auth/apikey"
	"github.com/synnaxlabs/synnax/pkg/service/whip"
)

var _ = Describe("ServiceConfig", func() {
	full := func() apiwhip.ServiceConfig {
		return apiwhip.ServiceConfig{
			Factory: &whip.Factory{},
			APIKey:  &apikey.Service{},
			RBAC:    &rbac.Service{},
		}
	}

	Describe("Validate", func() {
		It("Should succeed when every service is set", func() {
			Expect(full().Validate()).To(Succeed())
		})

		DescribeTable(
			"Should fail when a service is missing",
			func(clear func(*apiwhip.ServiceConfig), msg string) {
				cfg := full()
				clear(&cfg)
				Expect(cfg.Validate()).To(MatchError(msg))
			},
			Entry(
				"factory",
				func(c *apiwhip.ServiceConfig) { c.Factory = nil },
				"factory: must be non-nil",
			),
			Entry(
				"api key",
				func(c *apiwhip.ServiceConfig) { c.APIKey = nil },
				"api_key: must be non-nil",
			),
			Entry(
				"rbac",
				func(c *apiwhip.ServiceConfig) { c.RBAC = nil },
				"rbac: must be non-nil",
			),
		)
	})

	Describe("Override", func() {
		It("Should take the other config's services when set", func() {
			cfg := full()
			merged := apiwhip.ServiceConfig{}.Override(cfg)
			Expect(merged.Factory).To(BeIdenticalTo(cfg.Factory))
			Expect(merged.APIKey).To(BeIdenticalTo(cfg.APIKey))
			Expect(merged.RBAC).To(BeIdenticalTo(cfg.RBAC))
		})

		It("Should keep the receiver's services when the other's are nil", func() {
			cfg := full()
			merged := cfg.Override(apiwhip.ServiceConfig{})
			Expect(merged.Factory).To(BeIdenticalTo(cfg.Factory))
			Expect(merged.APIKey).To(BeIdenticalTo(cfg.APIKey))
			Expect(merged.RBAC).To(BeIdenticalTo(cfg.RBAC))
		})
	})

	Describe("NewService", func() {
		It("Should reject a config with no factory", func() {
			cfg := full()
			cfg.Factory = nil
			Expect(apiwhip.NewService(cfg)).Error().
				To(MatchError("factory: must be non-nil"))
		})
	})
})
