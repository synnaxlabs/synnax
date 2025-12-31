// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package omit_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/resolution"
)

var _ = Describe("IsStruct", func() {
	It("should return true when omit expression exists", func() {
		s := resolution.Struct{
			Domains: map[string]resolution.Domain{
				"ts": {Expressions: []resolution.Expression{{Name: "omit"}}},
			},
		}
		Expect(omit.IsStruct(s, "ts")).To(BeTrue())
	})

	It("should return false when domain missing", func() {
		s := resolution.Struct{Domains: map[string]resolution.Domain{}}
		Expect(omit.IsStruct(s, "ts")).To(BeFalse())
	})

	It("should return false when omit not in expressions", func() {
		s := resolution.Struct{
			Domains: map[string]resolution.Domain{
				"ts": {Expressions: []resolution.Expression{{Name: "output"}}},
			},
		}
		Expect(omit.IsStruct(s, "ts")).To(BeFalse())
	})

	It("should check correct domain", func() {
		s := resolution.Struct{
			Domains: map[string]resolution.Domain{
				"py": {Expressions: []resolution.Expression{{Name: "omit"}}},
			},
		}
		Expect(omit.IsStruct(s, "ts")).To(BeFalse())
		Expect(omit.IsStruct(s, "py")).To(BeTrue())
	})
})

var _ = Describe("IsEnum", func() {
	It("should return true when omit expression exists", func() {
		e := resolution.Enum{
			Domains: map[string]resolution.Domain{
				"ts": {Expressions: []resolution.Expression{{Name: "omit"}}},
			},
		}
		Expect(omit.IsEnum(e, "ts")).To(BeTrue())
	})

	It("should return false when domain missing", func() {
		e := resolution.Enum{Domains: map[string]resolution.Domain{}}
		Expect(omit.IsEnum(e, "ts")).To(BeFalse())
	})

	It("should return false when omit not in expressions", func() {
		e := resolution.Enum{
			Domains: map[string]resolution.Domain{
				"ts": {Expressions: []resolution.Expression{{Name: "output"}}},
			},
		}
		Expect(omit.IsEnum(e, "ts")).To(BeFalse())
	})
})
