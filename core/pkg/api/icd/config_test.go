// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiicd "github.com/synnaxlabs/synnax/pkg/api/icd"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
)

var _ = Describe("ServiceConfig", func() {
	full := func() apiicd.ServiceConfig {
		return apiicd.ServiceConfig{
			ICD: &icd.Service{},
		}
	}

	Describe("Validate", func() {
		It("Should succeed when every service is set", func() {
			Expect(full().Validate()).To(Succeed())
		})

		DescribeTable(
			"Should fail when a service is missing",
			func(clear func(*apiicd.ServiceConfig), msg string) {
				cfg := full()
				clear(&cfg)
				Expect(cfg.Validate()).To(MatchError(msg))
			},
			Entry(
				"icd",
				func(c *apiicd.ServiceConfig) { c.ICD = nil },
				"icd: must be non-nil",
			),
		)
	})

	Describe("Override", func() {
		It("Should take the other config's services when set", func() {
			cfg := full()
			merged := apiicd.ServiceConfig{}.Override(cfg)
			Expect(merged.ICD).To(BeIdenticalTo(cfg.ICD))
		})

		It("Should keep the receiver's services when the other's are nil", func() {
			cfg := full()
			merged := cfg.Override(apiicd.ServiceConfig{})
			Expect(merged.ICD).To(BeIdenticalTo(cfg.ICD))
		})
	})
})
