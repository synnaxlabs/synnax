// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package status_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apistatus "github.com/synnaxlabs/synnax/pkg/api/status"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/status"
)

var _ = Describe("ServiceConfig", func() {
	full := func() apistatus.ServiceConfig {
		return apistatus.ServiceConfig{
			Status:   &status.Service{},
			Label:    &label.Service{},
			Ontology: &ontology.Ontology{},
			RBAC:     &rbac.Service{},
		}
	}

	Describe("Validate", func() {
		It("Should succeed when every service is set", func() {
			Expect(full().Validate()).To(Succeed())
		})

		DescribeTable(
			"Should fail when a service is missing",
			func(clear func(*apistatus.ServiceConfig), msg string) {
				cfg := full()
				clear(&cfg)
				Expect(cfg.Validate()).To(MatchError(msg))
			},
			Entry(
				"status",
				func(c *apistatus.ServiceConfig) { c.Status = nil },
				"status: must be non-nil",
			),
			Entry(
				"label",
				func(c *apistatus.ServiceConfig) { c.Label = nil },
				"label: must be non-nil",
			),
			Entry(
				"ontology",
				func(c *apistatus.ServiceConfig) { c.Ontology = nil },
				"ontology: must be non-nil",
			),
			Entry(
				"rbac",
				func(c *apistatus.ServiceConfig) { c.RBAC = nil },
				"rbac: must be non-nil",
			),
		)
	})

	Describe("Override", func() {
		It("Should take the other config's services when set", func() {
			cfg := full()
			merged := apistatus.ServiceConfig{}.Override(cfg)
			Expect(merged.Status).To(BeIdenticalTo(cfg.Status))
			Expect(merged.Label).To(BeIdenticalTo(cfg.Label))
			Expect(merged.Ontology).To(BeIdenticalTo(cfg.Ontology))
			Expect(merged.RBAC).To(BeIdenticalTo(cfg.RBAC))
		})

		It("Should keep the receiver's services when the other's are nil", func() {
			cfg := full()
			merged := cfg.Override(apistatus.ServiceConfig{})
			Expect(merged.Status).To(BeIdenticalTo(cfg.Status))
			Expect(merged.Label).To(BeIdenticalTo(cfg.Label))
			Expect(merged.Ontology).To(BeIdenticalTo(cfg.Ontology))
			Expect(merged.RBAC).To(BeIdenticalTo(cfg.RBAC))
		})
	})
})
