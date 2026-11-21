// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package writer_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/x/control"
)

var _ = Describe("Facts", func() {
	It("Should take the index, concurrency, and virtual flag of a channel", func() {
		ch := channel.Channel{
			Leaseholder: 1,
			LocalKey:    5,
			LocalIndex:  4,
			Concurrency: control.ConcurrencyShared,
			Virtual:     true,
		}
		Expect(writer.FactsOf(ch)).To(Equal(writer.Facts{
			Index:       channel.NewKey(1, 4),
			Concurrency: control.ConcurrencyShared,
			Virtual:     true,
		}))
	})
})
