// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package text_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Bare channel path check", func() {
	It("Should report a missing channel for a path of three segments whose head is "+
		"an import alias", func(ctx SpecContext) {
		_, diagnostics := text.Analyze(
			ctx,
			MustSucceed(text.Parse(text.Text{
				Raw: "import math as m\n\nm.inf.x -> stand.valve_cmd",
			})),
			NewRoot(nil, symbol.Symbol{
				Name: "stand.valve_cmd",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.F64()),
				ID:   20,
			}),
		)
		Expect(*diagnostics).To(ContainElement(HaveField(
			"Message",
			"channel does not exist: m.inf.x",
		)), diagnostics.String())
	})
})
