// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package segment_test

import (
	"bytes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/place"
	"github.com/synnaxlabs/cesium/internal/segment"
)

var _ = Describe("Read offset check", func() {
	It("Should refuse a read at a negative stream offset as corrupt", func() {
		com := segment.Committed{ID: 3, Key: 6, Length: 100}
		negative := SatisfyAll(
			MatchError(place.ErrCorrupt),
			MatchError(ContainSubstring(
				"read of 10 bytes at stream offset -1 is past the committed length "+
					"100 of segment 3 of key 6",
			)),
		)
		file := bytes.NewReader(make([]byte, 200))
		Expect(segment.NewReader().Read(file, com, make([]byte, 10), -1)).To(negative)
		Expect(com.ReadStored(file, make([]byte, 10), -1)).To(negative)
	})
})
