// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build !invariants && !race

package invariants_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/invariants"
)

var _ = Describe("Enabled", func() {
	It(
		"Should be false in a build with neither the invariants nor the race tag",
		func() {
			Expect(invariants.Enabled).To(BeFalse())
		},
	)
})
