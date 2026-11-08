// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package bind_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
)

var _ = Describe("ServiceConfig", func() {
	full := func() bind.ServiceConfig {
		return bind.ServiceConfig{DB: &gorp.DB{}, RBAC: &rbac.Service{}}
	}

	Describe("Validate", func() {
		It("Should succeed when every field is set", func() {
			Expect(full().Validate()).To(Succeed())
		})

		DescribeTable(
			"Should fail when a field is missing",
			func(clear func(*bind.ServiceConfig), msg string) {
				cfg := full()
				clear(&cfg)
				Expect(cfg.Validate()).To(MatchError(msg))
			},
			Entry(
				"db",
				func(c *bind.ServiceConfig) { c.DB = nil },
				"db: must be non-nil",
			),
			Entry(
				"rbac",
				func(c *bind.ServiceConfig) { c.RBAC = nil },
				"rbac: must be non-nil",
			),
		)
	})

	Describe("Override", func() {
		It("Should take the other config's fields when set", func() {
			cfg := full()
			merged := bind.ServiceConfig{}.Override(cfg)
			Expect(merged.DB).To(BeIdenticalTo(cfg.DB))
			Expect(merged.RBAC).To(BeIdenticalTo(cfg.RBAC))
		})

		It("Should keep the receiver's fields when the other's are nil", func() {
			cfg := full()
			merged := cfg.Override(bind.ServiceConfig{})
			Expect(merged.DB).To(BeIdenticalTo(cfg.DB))
			Expect(merged.RBAC).To(BeIdenticalTo(cfg.RBAC))
		})
	})
})

var _ = Describe("NewService", func() {
	It("Should return the error of an invalid config", func() {
		Expect(bind.NewService(bind.ServiceConfig{RBAC: &rbac.Service{}})).Error().
			To(MatchError("db: must be non-nil"))
	})
})
