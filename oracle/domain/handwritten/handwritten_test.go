// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package handwritten_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/domain/handwritten"
	"github.com/synnaxlabs/oracle/resolution"
)

var _ = Describe("IsStruct", func() {
	It("should return true when handwritten expression exists", func() {
		s := &resolution.StructEntry{
			Domains: map[string]*resolution.DomainEntry{
				"ts": {Expressions: []*resolution.ExpressionEntry{{Name: "handwritten"}}},
			},
		}
		Expect(handwritten.IsStruct(s, "ts")).To(BeTrue())
	})

	It("should return false when domain missing", func() {
		s := &resolution.StructEntry{Domains: map[string]*resolution.DomainEntry{}}
		Expect(handwritten.IsStruct(s, "ts")).To(BeFalse())
	})

	It("should return false when handwritten not in expressions", func() {
		s := &resolution.StructEntry{
			Domains: map[string]*resolution.DomainEntry{
				"ts": {Expressions: []*resolution.ExpressionEntry{{Name: "output"}}},
			},
		}
		Expect(handwritten.IsStruct(s, "ts")).To(BeFalse())
	})

	It("should check correct domain", func() {
		s := &resolution.StructEntry{
			Domains: map[string]*resolution.DomainEntry{
				"py": {Expressions: []*resolution.ExpressionEntry{{Name: "handwritten"}}},
			},
		}
		Expect(handwritten.IsStruct(s, "ts")).To(BeFalse())
		Expect(handwritten.IsStruct(s, "py")).To(BeTrue())
	})
})

var _ = Describe("IsEnum", func() {
	It("should return true when handwritten expression exists", func() {
		e := &resolution.EnumEntry{
			Domains: map[string]*resolution.DomainEntry{
				"ts": {Expressions: []*resolution.ExpressionEntry{{Name: "handwritten"}}},
			},
		}
		Expect(handwritten.IsEnum(e, "ts")).To(BeTrue())
	})

	It("should return false when domain missing", func() {
		e := &resolution.EnumEntry{Domains: map[string]*resolution.DomainEntry{}}
		Expect(handwritten.IsEnum(e, "ts")).To(BeFalse())
	})

	It("should return false when handwritten not in expressions", func() {
		e := &resolution.EnumEntry{
			Domains: map[string]*resolution.DomainEntry{
				"ts": {Expressions: []*resolution.ExpressionEntry{{Name: "output"}}},
			},
		}
		Expect(handwritten.IsEnum(e, "ts")).To(BeFalse())
	})
})
