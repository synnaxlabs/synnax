// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mock_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
)

var _ = Describe("StaticHostProvider", func() {
	It("Should report the configured key as the host", func() {
		p := mock.NewStaticHostProvider(42)
		Expect(p.HostKey()).To(Equal(node.Key(42)))
		Expect(p.Host().Key).To(Equal(node.Key(42)))
	})

	It("Should consistently report the same host across calls", func() {
		p := mock.NewStaticHostProvider(7)
		Expect(p.HostKey()).To(Equal(p.Host().Key))
		Expect(p.HostKey()).To(Equal(node.Key(7)))
	})
})
