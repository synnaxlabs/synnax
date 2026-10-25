// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apisymbol "github.com/synnaxlabs/synnax/pkg/api/schematic/symbol"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
)

var _ = Describe("ServiceConfig", func() {
	full := func() apisymbol.ServiceConfig {
		return apisymbol.ServiceConfig{
			Symbol: &symbol.Service{},
			RBAC:   &rbac.Service{},
		}
	}

	Describe("Validate", func() {
		It("Should succeed when every service is set", func() {
			Expect(full().Validate()).To(Succeed())
		})

		DescribeTable(
			"Should fail when a service is missing",
			func(clear func(*apisymbol.ServiceConfig), msg string) {
				cfg := full()
				clear(&cfg)
				Expect(cfg.Validate()).To(MatchError(msg))
			},
			Entry(
				"symbol",
				func(c *apisymbol.ServiceConfig) { c.Symbol = nil },
				"symbol: must be non-nil",
			),
			Entry(
				"rbac",
				func(c *apisymbol.ServiceConfig) { c.RBAC = nil },
				"rbac: must be non-nil",
			),
		)
	})

	Describe("Override", func() {
		It("Should take the other config's services when set", func() {
			cfg := full()
			merged := apisymbol.ServiceConfig{}.Override(cfg)
			Expect(merged.Symbol).To(BeIdenticalTo(cfg.Symbol))
			Expect(merged.RBAC).To(BeIdenticalTo(cfg.RBAC))
		})

		It("Should keep the receiver's services when the other's are nil", func() {
			cfg := full()
			merged := cfg.Override(apisymbol.ServiceConfig{})
			Expect(merged.Symbol).To(BeIdenticalTo(cfg.Symbol))
			Expect(merged.RBAC).To(BeIdenticalTo(cfg.RBAC))
		})
	})
})
