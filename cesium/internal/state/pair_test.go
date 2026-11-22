// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package state_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/state"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Pair", func() {
	It("Should report a publish after the loads", func() {
		s := MustSucceed(state.Open(config(&appended{}), sample()))
		run(s)
		_, _, count := s.Pair(6, 5)
		Expect(s.Unchanged(count)).To(BeTrue())
		Expect(s.Durable(10, []log.Record{&log.Drop{Key: 6}})).To(Succeed())
		Eventually(func() bool { return s.Unchanged(count) }).Should(BeFalse())
	})
})
