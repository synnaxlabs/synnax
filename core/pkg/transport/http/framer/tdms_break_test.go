// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/transport/http/framer"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("TDMS names (break)", func() {
	It("Should export two channels of one type whose names differ by a type suffix",
		func(ctx SpecContext) {
			index, channels := createIndexed(ctx, telem.Float32T, telem.Float32T)
			write(
				ctx,
				index,
				telem.NewSeriesSecondsTSV(1, 2),
				channels,
				telem.NewSeriesV[float32](1, 2),
				telem.NewSeriesV[float32](3, 4),
			)
			// No series of either channel holds another type, so no typed column
			// exists for a name to clash with.
			channels[1].Name = channels[0].Name + " (int16)"
			_, err := framer.NewTDMSEncoder(ctx).Encode(read(ctx, index, channels...))
			Expect(err).To(Succeed())
		},
	)
})
