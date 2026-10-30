// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	apiframer "github.com/synnaxlabs/synnax/pkg/api/framer"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
)

var _ = Describe("ServiceConfig", func() {
	full := func() apiframer.ServiceConfig {
		return apiframer.ServiceConfig{
			Instrumentation: alamos.New("test"),
			Framer:          &framer.Service{},
			Channel:         &channel.Service{},
			RBAC:            &rbac.Service{},
		}
	}

	Describe("Validate", func() {
		It("Should succeed when every service is set", func() {
			Expect(full().Validate()).To(Succeed())
		})

		DescribeTable(
			"Should fail when a service is missing",
			func(clear func(*apiframer.ServiceConfig), msg string) {
				cfg := full()
				clear(&cfg)
				Expect(cfg.Validate()).To(MatchError(msg))
			},
			Entry(
				"framer",
				func(c *apiframer.ServiceConfig) { c.Framer = nil },
				"framer: must be non-nil",
			),
			Entry(
				"channel",
				func(c *apiframer.ServiceConfig) { c.Channel = nil },
				"channel: must be non-nil",
			),
			Entry(
				"rbac",
				func(c *apiframer.ServiceConfig) { c.RBAC = nil },
				"rbac: must be non-nil",
			),
		)
	})

	Describe("Override", func() {
		It("Should take the other config's services when set", func() {
			cfg := full()
			merged := apiframer.ServiceConfig{}.Override(cfg)
			Expect(merged.Framer).To(BeIdenticalTo(cfg.Framer))
			Expect(merged.Channel).To(BeIdenticalTo(cfg.Channel))
			Expect(merged.RBAC).To(BeIdenticalTo(cfg.RBAC))
			Expect(merged.Meta).To(Equal(cfg.Meta))
		})

		It("Should keep the receiver's services when the other's are nil", func() {
			cfg := full()
			merged := cfg.Override(apiframer.ServiceConfig{})
			Expect(merged.Framer).To(BeIdenticalTo(cfg.Framer))
			Expect(merged.Channel).To(BeIdenticalTo(cfg.Channel))
			Expect(merged.RBAC).To(BeIdenticalTo(cfg.RBAC))
			Expect(merged.Meta).To(Equal(cfg.Meta))
		})
	})
})
