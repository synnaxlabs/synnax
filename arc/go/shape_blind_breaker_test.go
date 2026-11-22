// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
)

var _ = Describe("Shape blind spot breaker", func() {
	It("Should see a write to the signature of a shared module member", func() {
		shared := stl.NewSymbols()
		before := shape(shared)
		for _, s := range shared {
			if s.Name != "time" || s.Kind != symbol.KindModule {
				continue
			}
			format := s.FindChild("format")
			format.Type.Inputs[1].Value = nil
			format.Type.Outputs[0].Type = types.I64()
		}
		Expect(shape(shared)).ToNot(Equal(before))
	})
})
