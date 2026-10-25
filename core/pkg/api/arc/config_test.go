// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	apiarc "github.com/synnaxlabs/synnax/pkg/api/arc"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/status"
)

var _ = Describe("ServiceConfig", func() {
	full := func() apiarc.ServiceConfig {
		return apiarc.ServiceConfig{
			Instrumentation: alamos.New("test"),
			RBAC:            &rbac.Service{},
			Arc:             &arc.Service{},
			Status:          &status.Service{},
		}
	}

	Describe("Validate", func() {
		It("Should succeed when every service is set", func() {
			Expect(full().Validate()).To(Succeed())
		})

		DescribeTable(
			"Should fail when a service is missing",
			func(clear func(*apiarc.ServiceConfig), msg string) {
				cfg := full()
				clear(&cfg)
				Expect(cfg.Validate()).To(MatchError(msg))
			},
			Entry(
				"rbac",
				func(c *apiarc.ServiceConfig) { c.RBAC = nil },
				"rbac: must be non-nil",
			),
			Entry(
				"arc",
				func(c *apiarc.ServiceConfig) { c.Arc = nil },
				"arc: must be non-nil",
			),
			Entry(
				"status",
				func(c *apiarc.ServiceConfig) { c.Status = nil },
				"status: must be non-nil",
			),
		)
	})

	Describe("Override", func() {
		It("Should take the other config's services when set", func() {
			cfg := full()
			merged := apiarc.ServiceConfig{}.Override(cfg)
			Expect(merged.RBAC).To(BeIdenticalTo(cfg.RBAC))
			Expect(merged.Arc).To(BeIdenticalTo(cfg.Arc))
			Expect(merged.Status).To(BeIdenticalTo(cfg.Status))
			Expect(merged.Meta).To(Equal(cfg.Meta))
		})

		It("Should keep the receiver's services when the other's are nil", func() {
			cfg := full()
			merged := cfg.Override(apiarc.ServiceConfig{})
			Expect(merged.RBAC).To(BeIdenticalTo(cfg.RBAC))
			Expect(merged.Arc).To(BeIdenticalTo(cfg.Arc))
			Expect(merged.Status).To(BeIdenticalTo(cfg.Status))
			Expect(merged.Meta).To(Equal(cfg.Meta))
		})
	})
})
