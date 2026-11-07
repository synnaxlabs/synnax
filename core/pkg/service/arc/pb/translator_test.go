// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pb_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	arcpb "github.com/synnaxlabs/synnax/pkg/service/arc/pb"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("ArcFromPB", func() {
	It("Should give an Arc whose message omits its source a text source", func() {
		a := MustSucceed(arcpb.ArcFromPB(&arcpb.Arc{
			Base:    &arcpb.Base{Key: uuid.New().String(), Name: "no-source"},
			Variant: &arcpb.Arc_Automation{Automation: &arcpb.ArcAutomationPayload{}},
		}))
		Expect(a.Base().Source.Variant).To(Equal(arc.TextSource{}))
		Expect(a.Base().Name).To(Equal("no-source"))
	})

	It("Should keep the source that the message carries", func() {
		a := MustSucceed(arcpb.ArcFromPB(&arcpb.Arc{
			Base: &arcpb.Base{
				Key:  uuid.New().String(),
				Name: "graph",
				Source: &arcpb.Source{
					Variant: &arcpb.Source_Graph{Graph: &arcpb.SourceGraphPayload{}},
				},
			},
			Variant: &arcpb.Arc_Automation{Automation: &arcpb.ArcAutomationPayload{}},
		}))
		Expect(a.Base().Source.Variant).To(BeAssignableToTypeOf(arc.GraphSource{}))
	})
})
