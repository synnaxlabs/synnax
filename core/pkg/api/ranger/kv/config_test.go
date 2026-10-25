// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kv_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apikv "github.com/synnaxlabs/synnax/pkg/api/ranger/kv"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/ranger/kv"
)

var _ = Describe("ServiceConfig", func() {
	full := func() apikv.ServiceConfig {
		return apikv.ServiceConfig{
			RBAC: &rbac.Service{},
			KV:   &kv.Service{},
		}
	}

	Describe("Validate", func() {
		It("Should succeed when every service is set", func() {
			Expect(full().Validate()).To(Succeed())
		})

		DescribeTable(
			"Should fail when a service is missing",
			func(clear func(*apikv.ServiceConfig), msg string) {
				cfg := full()
				clear(&cfg)
				Expect(cfg.Validate()).To(MatchError(msg))
			},
			Entry(
				"rbac",
				func(c *apikv.ServiceConfig) { c.RBAC = nil },
				"rbac: must be non-nil",
			),
			Entry(
				"kv",
				func(c *apikv.ServiceConfig) { c.KV = nil },
				"kv: must be non-nil",
			),
		)
	})

	Describe("Override", func() {
		It("Should take the other config's services when set", func() {
			cfg := full()
			merged := apikv.ServiceConfig{}.Override(cfg)
			Expect(merged.RBAC).To(BeIdenticalTo(cfg.RBAC))
			Expect(merged.KV).To(BeIdenticalTo(cfg.KV))
		})

		It("Should keep the receiver's services when the other's are nil", func() {
			cfg := full()
			merged := cfg.Override(apikv.ServiceConfig{})
			Expect(merged.RBAC).To(BeIdenticalTo(cfg.RBAC))
			Expect(merged.KV).To(BeIdenticalTo(cfg.KV))
		})
	})
})
