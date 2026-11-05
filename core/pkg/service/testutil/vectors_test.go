// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	. "github.com/synnaxlabs/synnax/pkg/service/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("ReadVectors", func() {
	It("Should read the records and cases of an entity", func() {
		v := MustSucceed(ReadVectors[ranger.Range, ranger.Where]("ranges"))
		Expect(v.Records).ToNot(BeEmpty())
		Expect(v.Cases).ToNot(BeEmpty())
	})

	It("Should return an error for an entity with no vectors", func() {
		Expect(ReadVectors[ranger.Range, ranger.Where]("dogs")).Error().
			To(MatchError("no filter vectors for dogs"))
	})

	It("Should return an error for a record with an unknown member", func() {
		Expect(ReadVectors[struct{}, ranger.Where]("ranges")).Error().
			To(MatchError(ContainSubstring("unknown object member name")))
	})
})
