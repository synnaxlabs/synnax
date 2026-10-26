// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apichannel "github.com/synnaxlabs/synnax/pkg/api/channel"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/ranger/alias"
	"github.com/synnaxlabs/synnax/pkg/service/status"
)

var _ = Describe("ServiceConfig", func() {
	full := func() apichannel.ServiceConfig {
		return apichannel.ServiceConfig{
			RBAC:    &rbac.Service{},
			Channel: &channel.Service{},
			Status:  &status.Service{},
			Alias:   &alias.Service{},
		}
	}

	Describe("Validate", func() {
		It("Should succeed when every service is set", func() {
			Expect(full().Validate()).To(Succeed())
		})

		DescribeTable(
			"Should fail when a service is missing",
			func(clear func(*apichannel.ServiceConfig), msg string) {
				cfg := full()
				clear(&cfg)
				Expect(cfg.Validate()).To(MatchError(msg))
			},
			Entry(
				"rbac",
				func(c *apichannel.ServiceConfig) { c.RBAC = nil },
				"rbac: must be non-nil",
			),
			Entry(
				"channel",
				func(c *apichannel.ServiceConfig) { c.Channel = nil },
				"channel: must be non-nil",
			),
			Entry(
				"status",
				func(c *apichannel.ServiceConfig) { c.Status = nil },
				"status: must be non-nil",
			),
			Entry(
				"alias",
				func(c *apichannel.ServiceConfig) { c.Alias = nil },
				"alias: must be non-nil",
			),
		)
	})

	Describe("Override", func() {
		It("Should take the other config's services when set", func() {
			cfg := full()
			merged := apichannel.ServiceConfig{}.Override(cfg)
			Expect(merged.RBAC).To(BeIdenticalTo(cfg.RBAC))
			Expect(merged.Channel).To(BeIdenticalTo(cfg.Channel))
			Expect(merged.Status).To(BeIdenticalTo(cfg.Status))
			Expect(merged.Alias).To(BeIdenticalTo(cfg.Alias))
		})

		It("Should keep the receiver's services when the other's are nil", func() {
			cfg := full()
			merged := cfg.Override(apichannel.ServiceConfig{})
			Expect(merged.RBAC).To(BeIdenticalTo(cfg.RBAC))
			Expect(merged.Channel).To(BeIdenticalTo(cfg.Channel))
			Expect(merged.Status).To(BeIdenticalTo(cfg.Status))
			Expect(merged.Alias).To(BeIdenticalTo(cfg.Alias))
		})
	})
})
