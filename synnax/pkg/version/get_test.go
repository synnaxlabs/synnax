// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package version_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/synnaxlabs/synnax/pkg/version"
	"strings"
)

var _ = Describe("Get", func() {
	Describe("Get", func() {
		It("Should return the version name", func() {
			v := version.Prod()
			cv := strings.TrimSpace(strings.ReplaceAll(v, "\n", ""))
			Expect(strings.Count(cv, ".")).To(Equal(2))
		})
	})
})
