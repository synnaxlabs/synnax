// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package connectivity_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiconnectivity "github.com/synnaxlabs/synnax/pkg/api/connectivity"
	"github.com/synnaxlabs/synnax/pkg/service/license"
)

var _ = Describe("ServiceConfig", func() {
	full := func() apiconnectivity.ServiceConfig {
		return apiconnectivity.ServiceConfig{
			Identity: node.Identity,
			License:  &license.Service{},
		}
	}

	Describe("Validate", func() {
		It("Should succeed when every service is set", func() {
			Expect(full().Validate()).To(Succeed())
		})

		DescribeTable(
			"Should fail when a service is missing",
			func(clear func(*apiconnectivity.ServiceConfig), msg string) {
				cfg := full()
				clear(&cfg)
				Expect(cfg.Validate()).To(MatchError(msg))
			},
			Entry(
				"identity key",
				func(c *apiconnectivity.ServiceConfig) { c.Identity.Key = uuid.Nil() },
				"identity.key: must be set",
			),
			Entry(
				"identity host key",
				func(c *apiconnectivity.ServiceConfig) { c.Identity.HostKey = 0 },
				"identity.host_key: must be non-zero",
			),
			Entry(
				"license",
				func(c *apiconnectivity.ServiceConfig) { c.License = nil },
				"license: must be non-nil",
			),
		)
	})

	Describe("Override", func() {
		It("Should take the other config's services when set", func() {
			cfg := full()
			merged := apiconnectivity.ServiceConfig{}.Override(cfg)
			Expect(merged.Identity).To(Equal(cfg.Identity))
			Expect(merged.License).To(BeIdenticalTo(cfg.License))
		})

		It("Should keep the receiver's services when the other's are nil", func() {
			cfg := full()
			merged := cfg.Override(apiconnectivity.ServiceConfig{})
			Expect(merged.Identity).To(Equal(cfg.Identity))
			Expect(merged.License).To(BeIdenticalTo(cfg.License))
		})
	})
})
