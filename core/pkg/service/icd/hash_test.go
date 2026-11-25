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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/x/content"
)

var _ = Describe("HashMessage", func() {
	var key icd.MessageKey
	BeforeEach(func() { key = uuid.New() })
	// pointedAt returns a message with key that reads one float field at pointer.
	pointedAt := func(pointer string) icd.Message {
		m := jsonMessage(icd.JSONField{Variant: icd.FloatJSONField{
			Name: "v", Scale: 1, Pointer: pointer,
		}})
		m.Key = key
		return m
	}
	sum := func(ic icd.ICD) string {
		GinkgoHelper()
		h := content.NewHash()
		Expect(icd.HashMessage(&h, ic, key)).To(Succeed())
		return h.Sum()
	}

	It("Should give the same sum for the same message", func() {
		ic := icd.ICD{Messages: []icd.Message{pointedAt("/v")}}
		Expect(sum(ic)).To(Equal(sum(ic)))
	})

	It("Should change when the payload of the message changes", func() {
		Expect(sum(icd.ICD{Messages: []icd.Message{pointedAt("/v")}})).
			ToNot(Equal(sum(icd.ICD{Messages: []icd.Message{pointedAt("/w")}})))
	})

	It("Should not change when another message changes", func() {
		other := jsonMessage()
		other.Key = uuid.New()
		changed := jsonMessage(icd.JSONField{Variant: icd.FloatJSONField{
			Name: "w", Scale: 1, Pointer: "/w",
		}})
		changed.Key = other.Key
		Expect(sum(icd.ICD{Messages: []icd.Message{pointedAt("/v"), other}})).
			To(Equal(sum(icd.ICD{Messages: []icd.Message{pointedAt("/v"), changed}})))
	})

	It("Should mark a message that the ICD lacks", func() {
		unwritten := content.NewHash()
		Expect(sum(icd.ICD{})).ToNot(Equal(unwritten.Sum()))
	})
})
