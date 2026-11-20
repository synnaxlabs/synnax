// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/io/fs/testutil/crash"
)

var _ = Describe("One clock crash check", func() {
	for _, p := range []struct {
		name    string
		profile crash.Profile
	}{
		{"Strict", crash.Strict},
		{"SyncErrors", crash.SyncErrors},
		{"FullDisk", crash.FullDisk},
		{"OneTimeReadFaults", crash.OneTimeReadFaults},
	} {
		for key := range uint64(32) {
			It(fmt.Sprintf("Should hold Visible, %s, run key %d", p.name, key),
				func(ctx SpecContext) {
					Expect(run(ctx, crash.Visible, p.profile, key, false)).To(Succeed())
				},
			)
		}
	}
})
