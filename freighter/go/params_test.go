// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package freighter_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
)

var _ = Describe("Params", func() {
	Describe("Set", func() {
		It("Should set the value of the given key", func() {
			p := make(freighter.Params)
			p.Set("key", "value")
			v, _ := p.Get("key")
			Expect(v).To(Equal("value"))
		})
		It("Should override the value of the given key", func() {
			p := make(freighter.Params)
			p.Set("key", "value")
			p.Set("key", "new value")
			v, _ := p.Get("key")
			Expect(v).To(Equal("new value"))
		})
	})
	Describe("Get", func() {
		It("Should return the value of the given key", func() {
			p := make(freighter.Params)
			p.Set("key", "value")
			v, ok := p.Get("key")
			Expect(v).To(Equal("value"))
			Expect(ok).To(BeTrue())
		})
		It("Should return false if the key is not set", func() {
			p := make(freighter.Params)
			v, ok := p.Get("key")
			Expect(v).To(BeZero())
			Expect(ok).To(BeFalse())
		})
	})
	Describe("GetDefault", func() {
		It("Should return the value of the given key", func() {
			p := make(freighter.Params)
			p.Set("key", "value")
			Expect(p.GetDefault("key", "default")).To(Equal("value"))
		})
		It("Should return the fallback value if the key is not set", func() {
			p := make(freighter.Params)
			Expect(p.GetDefault("key", "default")).To(Equal("default"))
		})
	})
})
