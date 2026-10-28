// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranger_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiranger "github.com/synnaxlabs/synnax/pkg/api/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
)

var _ = Describe("ServiceConfig", func() {
	full := func() apiranger.ServiceConfig {
		return apiranger.ServiceConfig{
			RBAC:   &rbac.Service{},
			Ranger: &ranger.Service{},
			Label:  &label.Service{},
		}
	}

	Describe("Validate", func() {
		It("Should succeed when every service is set", func() {
			Expect(full().Validate()).To(Succeed())
		})

		DescribeTable(
			"Should fail when a service is missing",
			func(clear func(*apiranger.ServiceConfig), msg string) {
				cfg := full()
				clear(&cfg)
				Expect(cfg.Validate()).To(MatchError(msg))
			},
			Entry(
				"rbac",
				func(c *apiranger.ServiceConfig) { c.RBAC = nil },
				"rbac: must be non-nil",
			),
			Entry(
				"ranger",
				func(c *apiranger.ServiceConfig) { c.Ranger = nil },
				"ranger: must be non-nil",
			),
			Entry(
				"label",
				func(c *apiranger.ServiceConfig) { c.Label = nil },
				"label: must be non-nil",
			),
		)
	})

	Describe("Override", func() {
		It("Should take the other config's services when set", func() {
			cfg := full()
			merged := apiranger.ServiceConfig{}.Override(cfg)
			Expect(merged.RBAC).To(BeIdenticalTo(cfg.RBAC))
			Expect(merged.Ranger).To(BeIdenticalTo(cfg.Ranger))
			Expect(merged.Label).To(BeIdenticalTo(cfg.Label))
		})

		It("Should keep the receiver's services when the other's are nil", func() {
			cfg := full()
			merged := cfg.Override(apiranger.ServiceConfig{})
			Expect(merged.RBAC).To(BeIdenticalTo(cfg.RBAC))
			Expect(merged.Ranger).To(BeIdenticalTo(cfg.Ranger))
			Expect(merged.Label).To(BeIdenticalTo(cfg.Label))
		})
	})
})
