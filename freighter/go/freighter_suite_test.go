// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package freighter_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
)

func TestGo(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Freighter Suite")
}

var _ = Describe("SenderNopCloser", func() {
	It("Should implement the freighter.StreamSenderCloser interface", func() {
		var closer freighter.StreamSenderCloser[int] = freighter.SenderNopCloser[int]{}
		Expect(closer.CloseSend()).To(Succeed())
	})
})
