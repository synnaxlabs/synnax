// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package impact_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiimpact "github.com/synnaxlabs/synnax/pkg/api/impact"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/impact"
)

var _ = Describe("ServiceConfig", func() {
	full := func() apiimpact.ServiceConfig {
		return apiimpact.ServiceConfig{RBAC: &rbac.Service{}, Impact: &impact.Service{}}
	}

	Describe("Validate", func() {
		It("Should succeed when every service is set", func() {
			Expect(full().Validate()).To(Succeed())
		})

		DescribeTable(
			"Should fail when a service is missing",
			func(unset func(*apiimpact.ServiceConfig), msg string) {
				cfg := full()
				unset(&cfg)
				Expect(cfg.Validate()).To(MatchError(msg))
			},
			Entry(
				"rbac",
				func(c *apiimpact.ServiceConfig) { c.RBAC = nil },
				"rbac: must be non-nil",
			),
			Entry(
				"impact",
				func(c *apiimpact.ServiceConfig) { c.Impact = nil },
				"impact: must be non-nil",
			),
		)
	})
})
