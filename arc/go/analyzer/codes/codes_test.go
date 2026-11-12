// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package codes_test

import (
	"encoding/json"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer/codes"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("All", func() {
	It("Should give each kind a code that no other kind has", func() {
		seen := set.New(
			codes.TypeMismatch,
			codes.TypeConstraintViolation,
			codes.SymbolUndefined,
			codes.SymbolRedefined,
		)
		for _, kind := range codes.All {
			Expect(string(kind.Code)).To(MatchRegexp(`^ARC\d{4}$`))
			Expect(seen.Contains(kind.Code)).To(BeFalse(), string(kind.Code))
			seen.Add(kind.Code)
		}
	})

	It("Should match the generated file", func() {
		var generated map[diagnostics.ErrorCode]struct{ Message string }
		Expect(
			json.Unmarshal(MustSucceed(os.ReadFile("codes.gen.json")), &generated),
		).To(Succeed())
		Expect(generated).To(HaveLen(len(codes.All)))
		for _, kind := range codes.All {
			Expect(generated).To(HaveKey(kind.Code))
			Expect(generated[kind.Code].Message).To(Equal(kind.Message))
		}
	})
})
