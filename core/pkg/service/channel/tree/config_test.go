// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package tree_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/channel/tree"
	"github.com/synnaxlabs/synnax/pkg/service/group"
)

var _ = Describe("ServiceConfig", func() {
	It("Should fail without a group service", func() {
		Expect(tree.ServiceConfig{}.Validate()).
			To(MatchError("group: must be non-nil"))
	})

	It("Should take the other config's group service when set", func() {
		groups := &group.Service{}
		Expect(tree.ServiceConfig{}.Override(tree.ServiceConfig{Group: groups}).Group).
			To(BeIdenticalTo(groups))
	})
})
