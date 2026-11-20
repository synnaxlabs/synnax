// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/log"
)

var _ = Describe("Frame bound check", func() {
	It("Should refuse a frame larger than 16 MiB that passes its check", func() {
		entry := make([]byte, log.MaxFrame-log.FrameHeaderSize+8)
		f := log.Frame{Count: 1, Sequence: 1, Entries: entry}
		b := f.Append(nil, db, generation, log.FramesStart)
		Expect(log.DecodeFrame(b, db, generation, log.FramesStart)).Error().
			To(SatisfyAll(
				MatchError(ContainSubstring("past its bounds")),
				MatchError(ContainSubstring("frame at 8192")),
			))
	})
})
