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
	It("Should always report the configured node as the host", func() {
		p := mock.NewStaticHostProvider(node.Key(42))
		Expect(p.HostKey()).To(Equal(node.Key(42)))
		Expect(p.Host()).To(Equal(node.Node{Key: node.Key(42)}))
	})
})
