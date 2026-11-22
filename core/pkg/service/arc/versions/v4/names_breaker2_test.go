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
	v4 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v4"
	"github.com/synnaxlabs/x/set"
)

var _ = Describe("RenameChannels breaker 2", func() {
	It("Should scan a program of unclosed block comments in linear time", func() {
		text := strings.Repeat("/*a", (1<<19)/3)
		start := time.Now()
		Expect(
			v4.RenameChannels(text, dashed(set.New("pt-01").Contains)),
		).To(Equal(text))
		Expect(time.Since(start)).To(BeNumerically("<", time.Second))
	})
})
