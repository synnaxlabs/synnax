// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package device_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apidevice "github.com/synnaxlabs/synnax/pkg/api/device"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/status"
)

var _ = Describe("ServiceConfig", func() {
	full := func() apidevice.ServiceConfig {
		return apidevice.ServiceConfig{
			Device:   &device.Service{},
			Status:   &status.Service{},
			RBAC:     &rbac.Service{},
			Ontology: &ontology.Ontology{},
		}
	}

	Describe("Validate", func() {
		It("Should succeed when every service is set", func() {
			Expect(full().Validate()).To(Succeed())
		})

		DescribeTable(
			"Should fail when a service is missing",
			func(clear func(*apidevice.ServiceConfig), msg string) {
				cfg := full()
				clear(&cfg)
				Expect(cfg.Validate()).To(MatchError(msg))
			},
			Entry(
				"device",
				func(c *apidevice.ServiceConfig) { c.Device = nil },
				"device: must be non-nil",
			),
			Entry(
				"status",
				func(c *apidevice.ServiceConfig) { c.Status = nil },
				"status: must be non-nil",
			),
			Entry(
				"rbac",
				func(c *apidevice.ServiceConfig) { c.RBAC = nil },
				"rbac: must be non-nil",
			),
			Entry(
				"ontology",
				func(c *apidevice.ServiceConfig) { c.Ontology = nil },
				"ontology: must be non-nil",
			),
		)
	})

	Describe("Override", func() {
		It("Should take the other config's services when set", func() {
			cfg := full()
			merged := apidevice.ServiceConfig{}.Override(cfg)
			Expect(merged.Device).To(BeIdenticalTo(cfg.Device))
			Expect(merged.Status).To(BeIdenticalTo(cfg.Status))
			Expect(merged.RBAC).To(BeIdenticalTo(cfg.RBAC))
			Expect(merged.Ontology).To(BeIdenticalTo(cfg.Ontology))
		})

		It("Should keep the receiver's services when the other's are nil", func() {
			cfg := full()
			merged := cfg.Override(apidevice.ServiceConfig{})
			Expect(merged.Device).To(BeIdenticalTo(cfg.Device))
			Expect(merged.Status).To(BeIdenticalTo(cfg.Status))
			Expect(merged.RBAC).To(BeIdenticalTo(cfg.RBAC))
			Expect(merged.Ontology).To(BeIdenticalTo(cfg.Ontology))
		})
	})
})
