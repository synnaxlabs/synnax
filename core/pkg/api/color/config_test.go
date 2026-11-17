// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package color_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apicolor "github.com/synnaxlabs/synnax/pkg/api/color"
	"github.com/synnaxlabs/synnax/pkg/service/color"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("ServiceConfig", func() {
	It("Should refuse a config with no color service", func() {
		Expect(apicolor.ServiceConfig{}.Validate()).To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("color: must be non-nil")),
		))
	})

	Describe("Override", func() {
		It("Should take the other config's service when set", func() {
			other := apicolor.ServiceConfig{Color: &color.Service{}}
			merged := apicolor.ServiceConfig{}.Override(other)
			Expect(merged.Color).To(BeIdenticalTo(other.Color))
		})

		It("Should keep the receiver's service when the other's is nil", func() {
			cfg := apicolor.ServiceConfig{Color: &color.Service{}}
			merged := cfg.Override(apicolor.ServiceConfig{})
			Expect(merged.Color).To(BeIdenticalTo(cfg.Color))
		})
	})

	It("Should refuse to open a service with no color service", func() {
		Expect(apicolor.NewService()).Error().
			To(MatchError(ContainSubstring("color: must be non-nil")))
	})
})
