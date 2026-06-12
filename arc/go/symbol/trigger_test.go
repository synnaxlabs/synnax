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
	"github.com/synnaxlabs/arc/symbol"
)

var _ = Describe("TriggerBinding", func() {
	It("Should leave Target empty for TriggerOnly", func() {
		Expect(symbol.TriggerOnly.Target).To(BeEmpty())
	})

	It("Should bind Target to the named param for TriggerInput", func() {
		Expect(symbol.TriggerInput("value")).To(Equal(symbol.TriggerBinding{Target: "value"}))
		Expect(symbol.TriggerInput("value").Target).To(Equal("value"))
	})

	It("Should treat TriggerInput with an empty name as TriggerOnly", func() {
		Expect(symbol.TriggerInput("")).To(Equal(symbol.TriggerOnly))
	})

	It("Should distinguish a bound input from pure activation", func() {
		Expect(symbol.TriggerInput("value")).ToNot(Equal(symbol.TriggerOnly))
	})
})
