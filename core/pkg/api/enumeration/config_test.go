// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package enumeration_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apienumeration "github.com/synnaxlabs/synnax/pkg/api/enumeration"
	"github.com/synnaxlabs/synnax/pkg/service/enumeration"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("ServiceConfig", func() {
	It("Should refuse a config with no enumeration service", func() {
		Expect(apienumeration.ServiceConfig{}.Validate()).To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("enumeration: must be non-nil")),
		))
	})

	Describe("Override", func() {
		It("Should take the other config's service when set", func() {
			other := apienumeration.ServiceConfig{Enumeration: &enumeration.Service{}}
			merged := apienumeration.ServiceConfig{}.Override(other)
			Expect(merged.Enumeration).To(BeIdenticalTo(other.Enumeration))
		})

		It("Should keep the receiver's service when the other's is nil", func() {
			cfg := apienumeration.ServiceConfig{Enumeration: &enumeration.Service{}}
			merged := cfg.Override(apienumeration.ServiceConfig{})
			Expect(merged.Enumeration).To(BeIdenticalTo(cfg.Enumeration))
		})
	})

	It("Should refuse to open a service with no enumeration service", func() {
		Expect(apienumeration.NewService()).Error().
			To(MatchError(ContainSubstring("enumeration: must be non-nil")))
	})
})
