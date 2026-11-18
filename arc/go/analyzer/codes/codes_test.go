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
	"go.lsp.dev/protocol"
)

var _ = Describe("All", func() {
	It("Should give each kind a code that no other kind has", func() {
		seen := set.New[diagnostics.ErrorCode]()
		for _, kind := range codes.All {
			Expect(string(kind.Code)).To(MatchRegexp(`^[EW][1-5]\d{3}$`))
			Expect(seen.Contains(kind.Code)).To(BeFalse(), string(kind.Code))
			seen.Add(kind.Code)
		}
	})

	It("Should start each code with the letter of its severity", func() {
		letters := map[protocol.DiagnosticSeverity]byte{
			protocol.DiagnosticSeverityError:   'E',
			protocol.DiagnosticSeverityWarning: 'W',
		}
		for _, kind := range codes.All {
			Expect(kind.Code[0]).To(Equal(letters[kind.Severity]), string(kind.Code))
		}
	})

	It("Should have a problem text in the generated file for each kind", func() {
		var generated map[diagnostics.ErrorCode]struct {
			Problem string `json:"problem"`
		}
		Expect(
			json.Unmarshal(MustSucceed(os.ReadFile("codes.gen.json")), &generated),
		).To(Succeed())
		Expect(generated).To(HaveLen(len(codes.All)))
		for _, kind := range codes.All {
			Expect(generated).To(HaveKey(kind.Code))
			Expect(generated[kind.Code].Problem).ToNot(BeEmpty(), string(kind.Code))
		}
	})
})
