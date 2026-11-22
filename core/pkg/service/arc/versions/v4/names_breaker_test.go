// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v4_test

import (
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/literal"
	v4 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v4"
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("QuoteDashedNames breaker", func() {
	held := set.New("pt-01").Contains

	It("Should scan a program of line comments in linear time", func() {
		// v0.62 lexes 1 MiB of these in about 0.3 s.
		text := strings.Repeat("//\n", (1<<19)/3)
		start := time.Now()
		Expect(v4.QuoteDashedNames(text, held)).To(Equal(text))
		Expect(time.Since(start)).To(BeNumerically("<", time.Second))
	})

	It("Should quote a dashed name that a placeholder writes with an escape", func() {
		// v0.62 unescapes the body before it lexes the placeholder, so this reads
		// the channel pt-01.
		out := v4.QuoteDashedNames(`f"{p\u0074-01}"`, held)
		body, flags, ok := literal.StripQuotes(out)
		Expect(ok).To(BeTrue())
		segments := MustSucceed(literal.FmtStrSplit(body, flags))
		Expect(segments).To(HaveLen(1))
		Expect(segments[0].Text).To(Equal(`'pt-01'`))
	})
})
