// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package label_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apilabel "github.com/synnaxlabs/synnax/pkg/api/label"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/label"
)

var _ = Describe("ServiceConfig", func() {
	full := func() apilabel.ServiceConfig {
		return apilabel.ServiceConfig{
			Label: &label.Service{},
			RBAC:  &rbac.Service{},
		}
	}

	Describe("Validate", func() {
		It("Should succeed when every service is set", func() {
			Expect(full().Validate()).To(Succeed())
		})

		DescribeTable(
			"Should fail when a service is missing",
			func(clear func(*apilabel.ServiceConfig), msg string) {
				cfg := full()
				clear(&cfg)
				Expect(cfg.Validate()).To(MatchError(msg))
			},
			Entry(
				"label",
				func(c *apilabel.ServiceConfig) { c.Label = nil },
				"label: must be non-nil",
			),
			Entry(
				"rbac",
				func(c *apilabel.ServiceConfig) { c.RBAC = nil },
				"rbac: must be non-nil",
			),
		)
	})

	Describe("Override", func() {
		It("Should take the other config's services when set", func() {
			cfg := full()
			merged := apilabel.ServiceConfig{}.Override(cfg)
			Expect(merged.Label).To(BeIdenticalTo(cfg.Label))
			Expect(merged.RBAC).To(BeIdenticalTo(cfg.RBAC))
		})

		It("Should keep the receiver's services when the other's are nil", func() {
			cfg := full()
			merged := cfg.Override(apilabel.ServiceConfig{})
			Expect(merged.Label).To(BeIdenticalTo(cfg.Label))
			Expect(merged.RBAC).To(BeIdenticalTo(cfg.RBAC))
		})
	})
})
