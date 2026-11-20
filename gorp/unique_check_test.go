// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/query"
)

var _ = Describe("UniqueError", func() {
	It("Should name the holder and the space", func() {
		err := gorp.UniqueError{
			Space:  "namespace.item",
			Holder: gorp.NewTarget("item", uint32(7)),
		}
		Expect(err).To(MatchError(query.ErrUniqueViolation))
		Expect(err.Error()).
			To(Equal("[gorp] - item 7 already holds the value in namespace.item"))
	})
})
