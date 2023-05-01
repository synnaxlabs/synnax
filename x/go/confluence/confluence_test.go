// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package confluence_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
)

var _ = Describe("Confluence", func() {

	Describe("EmptyFlow", func() {

		It("Should do nothing", func() {
			ctx, cancel := signal.Isolated()
			defer cancel()
			Expect(func() {
				NopFlow{}.Flow(ctx)
			}).ToNot(Panic())
		})

	})

})
