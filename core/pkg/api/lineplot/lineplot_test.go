// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lineplot_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apilineplot "github.com/synnaxlabs/synnax/pkg/api/lineplot"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Create", func() {
	It("Should stamp the dispatch key onto the create broadcast", func(
		ctx SpecContext,
	) {
		u := newCreator(ctx)
		rec := &Recorder[lineplot.Key, lineplot.Action]{}
		DeferCleanup(svc.LinePlot.OnAction(rec.Record))
		res := MustSucceed(
			apiSvc.Create(AuthedCtx(ctx, u), nil, apilineplot.CreateRequest{
				Project:     proj.Key,
				LinePlots:   []lineplot.LinePlot{{Name: "broadcast"}},
				DispatchKey: "dk-create",
			}),
		)
		frames := rec.Snapshot()
		Expect(frames).To(HaveLen(1))
		Expect(frames[0].Key).To(Equal(res.LinePlots[0].Key))
		Expect(frames[0].DispatchKey).To(Equal("dk-create"))
	})
})
