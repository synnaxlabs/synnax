// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package project_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiproject "github.com/synnaxlabs/synnax/pkg/api/project"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/project"
)

var _ = Describe("ServiceConfig", func() {
	full := func() apiproject.ServiceConfig {
		return apiproject.ServiceConfig{
			RBAC:    &rbac.Service{},
			Project: &project.Service{},
		}
	}

	Describe("Validate", func() {
		It("Should succeed when every service is set", func() {
			Expect(full().Validate()).To(Succeed())
		})

		DescribeTable(
			"Should fail when a service is missing",
			func(clear func(*apiproject.ServiceConfig), msg string) {
				cfg := full()
				clear(&cfg)
				Expect(cfg.Validate()).To(MatchError(msg))
			},
			Entry(
				"rbac",
				func(c *apiproject.ServiceConfig) { c.RBAC = nil },
				"rbac: must be non-nil",
			),
			Entry(
				"project",
				func(c *apiproject.ServiceConfig) { c.Project = nil },
				"project: must be non-nil",
			),
		)
	})

	Describe("Override", func() {
		It("Should take the other config's services when set", func() {
			cfg := full()
			merged := apiproject.ServiceConfig{}.Override(cfg)
			Expect(merged.RBAC).To(BeIdenticalTo(cfg.RBAC))
			Expect(merged.Project).To(BeIdenticalTo(cfg.Project))
		})

		It("Should keep the receiver's services when the other's are nil", func() {
			cfg := full()
			merged := cfg.Override(apiproject.ServiceConfig{})
			Expect(merged.RBAC).To(BeIdenticalTo(cfg.RBAC))
			Expect(merged.Project).To(BeIdenticalTo(cfg.Project))
		})
	})
})
