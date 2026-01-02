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

var _ = Describe("IsType", func() {
	It("should return true when omit expression exists for struct", func() {
		t := resolution.Type{
			Form: resolution.StructForm{},
			Domains: map[string]resolution.Domain{
				"ts": {Expressions: []resolution.Expression{{Name: "omit"}}},
			},
		}
		Expect(omit.IsType(t, "ts")).To(BeTrue())
	})

	It("should return false when domain missing", func() {
		t := resolution.Type{
			Form:    resolution.StructForm{},
			Domains: map[string]resolution.Domain{},
		}
		Expect(omit.IsType(t, "ts")).To(BeFalse())
	})

	It("should return false when omit not in expressions", func() {
		t := resolution.Type{
			Form: resolution.StructForm{},
			Domains: map[string]resolution.Domain{
				"ts": {Expressions: []resolution.Expression{{Name: "output"}}},
			},
		}
		Expect(omit.IsType(t, "ts")).To(BeFalse())
	})

	It("should check correct domain", func() {
		t := resolution.Type{
			Form: resolution.StructForm{},
			Domains: map[string]resolution.Domain{
				"py": {Expressions: []resolution.Expression{{Name: "omit"}}},
			},
		}
		Expect(omit.IsType(t, "ts")).To(BeFalse())
		Expect(omit.IsType(t, "py")).To(BeTrue())
	})

	It("should return true when omit expression exists for enum", func() {
		t := resolution.Type{
			Form: resolution.EnumForm{},
			Domains: map[string]resolution.Domain{
				"ts": {Expressions: []resolution.Expression{{Name: "omit"}}},
			},
		}
		Expect(omit.IsType(t, "ts")).To(BeTrue())
	})

	It("should return false when enum domain missing", func() {
		t := resolution.Type{
			Form:    resolution.EnumForm{},
			Domains: map[string]resolution.Domain{},
		}
		Expect(omit.IsType(t, "ts")).To(BeFalse())
	})

	It("should return false when omit not in enum expressions", func() {
		t := resolution.Type{
			Form: resolution.EnumForm{},
			Domains: map[string]resolution.Domain{
				"ts": {Expressions: []resolution.Expression{{Name: "output"}}},
			},
		}
		Expect(omit.IsType(t, "ts")).To(BeFalse())
	})
})
