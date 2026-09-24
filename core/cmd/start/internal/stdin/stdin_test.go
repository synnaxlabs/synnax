// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package stdin_test

import (
	"io"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/cmd/start/internal/stdin"
)

var _ = Describe("Watch", func() {
	// count runs Watch over the input to exhaustion and returns the number of stops.
	count := func(input string, stopOnClose bool) int {
		stops := 0
		stdin.Watch(strings.NewReader(input), stopOnClose, func() { stops++ })
		return stops
	}

	DescribeTable("Should stop on the keyword and, when set, on exhausted input",
		func(input string, stopOnClose bool, expected int) {
			Expect(count(input, stopOnClose)).To(Equal(expected))
		},
		Entry("keyword line", "stop\n", false, 1),
		Entry("keyword without a trailing newline", "stop", false, 1),
		Entry("keyword after other lines", "hello\nstop\n", false, 1),
		Entry("no keyword", "hello\nstopped\n STOP\n", false, 0),
		Entry("empty input", "", false, 0),
		Entry("empty input with stopOnClose", "", true, 1),
		Entry("other lines with stopOnClose", "hello\n", true, 1),
	)

	It("Should stop when the writing end of a pipe closes", func() {
		r, w := io.Pipe()
		stopped := make(chan struct{})
		go stdin.Watch(r, true, func() { close(stopped) })
		Consistently(stopped, 50*time.Millisecond).ShouldNot(BeClosed())
		Expect(w.Close()).To(Succeed())
		Eventually(stopped).Should(BeClosed())
	})
})
