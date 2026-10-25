// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package group_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apigroup "github.com/synnaxlabs/synnax/pkg/api/group"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/group"
)

var _ = Describe("ServiceConfig", func() {
	full := func() apigroup.ServiceConfig {
		return apigroup.ServiceConfig{
			RBAC:  &rbac.Service{},
			Group: &group.Service{},
		}
	}

	Describe("Validate", func() {
		It("Should succeed when every service is set", func() {
			Expect(full().Validate()).To(Succeed())
		})

		DescribeTable(
			"Should fail when a service is missing",
			func(clear func(*apigroup.ServiceConfig), msg string) {
				cfg := full()
				clear(&cfg)
				Expect(cfg.Validate()).To(MatchError(msg))
			},
			Entry(
				"rbac",
				func(c *apigroup.ServiceConfig) { c.RBAC = nil },
				"rbac: must be non-nil",
			),
			Entry(
				"group",
				func(c *apigroup.ServiceConfig) { c.Group = nil },
				"group: must be non-nil",
			),
		)
	})

	Describe("Override", func() {
		It("Should take the other config's services when set", func() {
			cfg := full()
			merged := apigroup.ServiceConfig{}.Override(cfg)
			Expect(merged.RBAC).To(BeIdenticalTo(cfg.RBAC))
			Expect(merged.Group).To(BeIdenticalTo(cfg.Group))
		})

		It("Should keep the receiver's services when the other's are nil", func() {
			cfg := full()
			merged := cfg.Override(apigroup.ServiceConfig{})
			Expect(merged.RBAC).To(BeIdenticalTo(cfg.RBAC))
			Expect(merged.Group).To(BeIdenticalTo(cfg.Group))
		})
	})
})
